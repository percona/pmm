// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package alerting

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/dir"
)

const (
	// ProvisioningDir is where PMM writes the rules. Grafana looks for them in
	// /usr/share/grafana/conf/provisioning/alerting, which the image symlinks here.
	//
	// The real directory is under /srv for two reasons: it is guaranteed writable whatever UID the
	// process runs as, which the image directory is not under OpenShift, and it survives container
	// recreation, so an upgrade finds the file already in place instead of having to restart
	// Grafana to apply it. Writing the real path rather than the symlink also means the directory
	// can simply be created when it does not exist yet, which is the case on every fresh container
	// because /srv starts empty.
	provisioningDir = "/srv/grafana/provisioning/alerting"

	// ProvisioningFileName is the file PMM owns. Grafana reads every file in the directory, so
	// anything else in there belongs to someone else and is left alone.
	provisioningFileName = "pmm-builtin.json"

	provisioningFilePerm = os.FileMode(0o664)
	provisioningDirPerm  = os.FileMode(0o775)

	// GrafanaProgramName is the supervisord program Grafana runs under.
	grafanaProgramName = "grafana"

	// ReconcileInterval is slow on purpose, and the tick is the only thing that notices anything
	// changing while the process runs: the bundle toggles are read once at start, so what is left
	// is the Percona Alerting setting and a rule UID being squatted or released. Nothing triggers a
	// reconcile directly - a settings change reaches one node of a cluster, but every node has to
	// converge, because each writes its own copy of the file and its own Grafana reads that copy at
	// its next start. A follower left behind would revert the cluster, which is also why writing is
	// not leader-gated. Five minutes is the compromise: soon enough that a toggle feels like it
	// worked, rare enough to be invisible.
	reconcileInterval = 5 * time.Minute

	// A datasource that cannot be resolved blocks the whole bundle, so it is retried far faster
	// than the ordinary tick rather than leaving a server without rules for minutes. The first
	// delay matches setup()'s own two-second cadence, and the cap keeps it well inside one tick.
	datasourceRetryInitial = 2 * time.Second
	datasourceRetryMax     = time.Minute
	datasourceRetryFactor  = 2

	// GrafanaReadyTimeout bounds how long a restart is given to come back before it is called a
	// failure and the previous file restored.
	grafanaReadyTimeout = 2 * time.Minute

	// GrafanaReadyPollInterval is how often a restarting Grafana is asked whether it is back.
	grafanaReadyPollInterval = time.Second
)

// provisioningTrigger says what caused a reconcile, which decides how far the provisioner may go to
// make Grafana pick the file up.
type provisioningTrigger int

const (
	// triggerStartup runs while pmm-managed starts, and by then Grafana is normally already
	// running: setup() writes grafana.ini and starts it from UpdateSettingsFromEnv, before this
	// runs. Applying therefore usually does mean restarting Grafana - once, on the boot where the
	// content changed, and not at all on a restart that renders the same file.
	triggerStartup provisioningTrigger = iota
	// TriggerTick is the periodic reconcile. It never restarts Grafana except as the last-resort
	// leader fallback.
	triggerTick
	// TriggerRetry is the fast follow-up to a reconcile that could not finish: an unresolvable
	// datasource UID, a squatted rule UID, or an apply action that failed. It carries the tick's
	// restart permission: a shared Grafana database coming back reaches every node at once, so this
	// needs the gate that elects one of them to act.
	triggerRetry
)

// String names the trigger for the log lines that report what a reconcile did.
func (t provisioningTrigger) String() string {
	switch t {
	case triggerStartup:
		return "startup"
	case triggerTick:
		return "reconcile"
	case triggerRetry:
		return "retry"
	default:
		return "unknown"
	}
}

// Provisioner keeps PMM's built-in alert rules in Grafana.
//
// It renders a Grafana provisioning file from the shipped templates and makes Grafana ingest it.
// Every node of a cluster renders and writes the same file, but only one of them has to apply it:
// alert rules live in the Grafana database that all nodes share, and each Grafana re-reads them
// from there within a scheduler tick. A node that stopped writing would silently revert the cluster
// the next time its own Grafana restarted, which is why writing is not leader-gated.
//
// Nothing here may prevent PMM from starting, and nothing may hand Grafana a file it cannot parse:
// Grafana treats bad provisioning as a fatal startup error, which would take down the whole user
// interface and API rather than just alerting.
type Provisioner struct {
	db          *reform.DB
	grafana     grafanaProvisioningClient
	supervisord supervisordService
	leader      leaderService
	grafanaDB   *grafanaReader
	metrics     *ProvisioningMetrics

	gates         bundleGates
	dirPath       string
	templatesPath string
	l             *logrus.Entry

	// How long a restarted Grafana is given to answer again, and how often it is asked. Fields
	// rather than constants so tests do not have to wait them out.
	readyTimeout      time.Duration
	readyPollInterval time.Duration

	// m serialises the work. Two goroutines call in: the startup path, which setup() retries in a
	// goroutine of its own, and the Run loop. They write the same file and the same counters below,
	// so they must not overlap.
	m sync.Mutex

	// reportedConflicts is the set of squatted UIDs last logged, so a standing conflict is reported
	// once rather than on every tick.
	reportedConflicts string
	// retryBackoff is how long to wait before trying again after a reconcile that could not finish:
	// an unresolvable datasource UID, a squatted rule UID, or an apply action that failed. Zero
	// means no retry is owed. It is held through the recovery that follows, so the whole recovery
	// runs on the fast cadence instead of handing the last step back to the five-minute tick.
	retryBackoff time.Duration
	// applyPending records that an apply action PMM performs itself failed, so the next reconcile
	// must try again even though the file on disk is already the content it would write.
	applyPending bool
}

// ProvisionerParams holds Provisioner configuration.
type ProvisionerParams struct {
	DB          *reform.DB
	GrafanaCli  grafanaProvisioningClient
	Supervisord supervisordService
	Leader      leaderService
	// GrafanaDBAddr and GrafanaDBSSLParams describe PMM's own PostgreSQL server, which is where
	// the bundled Grafana keeps its database. They are only consulted when the deployment has not
	// pointed Grafana at an external database through GF_DATABASE_*.
	GrafanaDBAddr      string
	GrafanaDBSSLParams string
	// HAEnabled reports whether this server runs as part of a High Availability cluster.
	HAEnabled bool
	// HAAlertsEnabled and ComponentAlertsEnabled are PMM_ENABLE_HA_ALERTS and
	// PMM_ENABLE_COMPONENT_ALERTS. Both default to true and are fixed for the process lifetime.
	HAAlertsEnabled        bool
	ComponentAlertsEnabled bool
	// Dir overrides the provisioning directory. Empty means the location Grafana reads in the image.
	Dir string
	// TemplatesDir overrides where the built-in templates are read from. Empty means the location
	// they are installed to in the image.
	TemplatesDir string
}

// NewProvisioner creates a new Provisioner.
func NewProvisioner(params ProvisionerParams) *Provisioner {
	l := logrus.WithField("component", "alerting/provisioning")

	provisioningDirPath := params.Dir
	if provisioningDirPath == "" {
		provisioningDirPath = provisioningDir
	}

	templatesPath := params.TemplatesDir
	if templatesPath == "" {
		templatesPath = builtinTemplatesDir
	}

	// Where Grafana's database lives is worked out once: it is what the real datasource UID is read
	// from, and it does not change while the process runs.
	dsn, err := grafanaDatasourceDSN(grafanaDBFallback{Addr: params.GrafanaDBAddr, SSLParams: params.GrafanaDBSSLParams})
	if err != nil {
		// Nothing recomputes this, so unlike a database that is merely unreachable, no amount of
		// retrying will fix it: built-in rules stay unprovisioned until the configuration changes.
		l.Errorf("Cannot locate Grafana's database, so no built-in alert rules will be provisioned "+
			"until PMM is restarted with a working configuration: %s.", err)
	}

	return &Provisioner{
		db:          params.DB,
		grafana:     params.GrafanaCli,
		supervisord: params.Supervisord,
		leader:      params.Leader,
		grafanaDB:   newGrafanaReader(dsn, l),
		metrics:     newProvisioningMetrics(),
		gates: bundleGates{
			haEnabled:              params.HAEnabled,
			haAlertsEnabled:        params.HAAlertsEnabled,
			componentAlertsEnabled: params.ComponentAlertsEnabled,
		},
		dirPath:           provisioningDirPath,
		templatesPath:     templatesPath,
		l:                 l,
		readyTimeout:      grafanaReadyTimeout,
		readyPollInterval: grafanaReadyPollInterval,
	}
}

// Collector returns the Prometheus collector reporting what this node has rendered and whether
// Grafana has it. Several ways this can fail are invisible otherwise, and provisioned rules are
// deliberately built so that an execution error stays quiet.
func (p *Provisioner) Collector() *ProvisioningMetrics {
	return p.metrics
}

// armRetryLocked grows the backoff owed before the next attempt. Called with m held.
func (p *Provisioner) armRetryLocked() {
	if p.retryBackoff == 0 {
		p.retryBackoff = datasourceRetryInitial
		return
	}
	p.retryBackoff = min(p.retryBackoff*datasourceRetryFactor, datasourceRetryMax)
}

// retryAfter returns a channel that fires once the owed backoff has passed, or nil when none is
// owed. A nil channel blocks forever in a select, which is exactly "nothing to retry".
func (p *Provisioner) retryAfter() <-chan time.Time {
	p.m.Lock()
	defer p.m.Unlock()

	if p.retryBackoff == 0 {
		return nil
	}
	return time.After(p.retryBackoff)
}

// Run reconciles on request and on a slow tick until ctx is canceled.
func (p *Provisioner) Run(ctx context.Context) {
	p.l.Info("Starting...")
	defer p.l.Info("Done.")

	// Spread the tick across the nodes of a cluster so that they do not all wake together.
	ticker := time.NewTicker(jitter(reconcileInterval))
	defer ticker.Stop()

	// Read before the first select: the startup reconcile runs before this loop exists, so a
	// backoff it armed would otherwise wait out a whole tick.
	retry := p.retryAfter()

	for {
		select {
		case <-ctx.Done():
			err := p.grafanaDB.Close()
			if err != nil {
				p.l.Debugf("Failed to close the Grafana database connection: %s.", err)
			}
			return

		case <-ticker.C:
			p.reconcile(ctx, triggerTick)
			ticker.Reset(jitter(reconcileInterval))

		case <-retry:
			p.reconcile(ctx, triggerRetry)
		}

		retry = p.retryAfter()
	}
}

// jitterFraction is how far either side of the interval a tick may land, as a divisor: 5 means the
// tick happens within plus or minus a fifth of the interval.
const jitterFraction = 5

func jitter(d time.Duration) time.Duration {
	spread := d / jitterFraction
	//nolint:gosec // Spreading timers across nodes does not need a cryptographic random source.
	offset := time.Duration(rand.Int64N(int64(spread + spread)))
	return d - spread + offset
}

// ProvisionAtStartup writes the rules while PMM starts.
//
// Grafana is normally already running by this point, on a fresh container as well as an existing
// one: setup() writes grafana.ini and starts it from UpdateSettingsFromEnv, before this runs. So
// this may restart Grafana - once, on the boot where the content changed.
func (p *Provisioner) ProvisionAtStartup(ctx context.Context) {
	p.reconcile(ctx, triggerStartup)
}

// reconcile renders the provisioning file, writes it if it changed, and makes Grafana pick it up as
// far as this trigger allows. It never returns an error: a failure is logged and counted, and the
// next tick tries again. Returning one would only give PMM's startup path something it must not act
// on, since provisioning may never stop the server from starting.
func (p *Provisioner) reconcile(ctx context.Context, trigger provisioningTrigger) {
	// Skip rather than queue behind the reconcile in flight: reconciling is idempotent, so the one
	// already running does the same work.
	if !p.m.TryLock() {
		p.l.Debugf("A reconcile is already running, skipping this %s.", trigger)
		return
	}
	defer p.m.Unlock()

	content, bundles, err := p.render(ctx)
	if err != nil {
		// An unresolvable datasource is the one render failure that is not PMM's own doing, and the
		// one worth retrying quickly: it usually means Grafana's database is briefly out of reach.
		stage := stageRender
		switch {
		case errors.Is(err, errDatasourceUnresolved):
			stage = stageDatasource
			p.armRetryLocked()
		case errors.Is(err, errRuleUIDTaken):
			stage = stageConflict
			p.armRetryLocked()
		}
		p.l.Errorf("Failed to render the alert rule provisioning file: %s.", err)
		p.metrics.recordError(stage)
		return
	}

	err = validateProvisioningFile(content)
	if err != nil {
		// Refusing to write is the whole point: a file Grafana cannot parse stops it from starting.
		p.l.Errorf("Refusing to write an invalid alert rule provisioning file: %s.", err)
		p.metrics.recordError(stageValidate)
		return
	}

	hash := contentHash(content)
	previous, changed, err := p.write(content)
	if err != nil {
		p.l.Errorf("Failed to write the alert rule provisioning file: %s.", err)
		p.metrics.recordError(stageWrite)
		return
	}

	p.metrics.setRendered(hash, bundles)
	p.metrics.setWritten(hash)
	p.retryBackoff = 0

	if !changed && !p.applyPending {
		// The file on disk is already the content we would write, and PMM owes no apply of its own,
		// so there is nothing to do. This is the ordinary case on every restart.
		return
	}

	err = p.apply(ctx, trigger, previous)
	switch {
	case err == nil:
		p.applyPending = false
		p.metrics.setApplyPending(false)

	case errors.Is(err, errDeferredToLeader):
		// Not a failure and not this node's work to retry: the leader applies, or Grafana reads the
		// file itself as it starts. Counting this would make every healthy follower look broken.
		p.applyPending = false
		p.metrics.setApplyPending(false)
		p.l.Debugf("Alert rules written; %s.", err)

	default:
		// An action PMM performs itself failed - starting a Grafana supervisord has given up on, or
		// restarting a running one. Nothing else will retry it: the file is already correct, so
		// every later tick would return early on "unchanged". Arm the backoff instead.
		p.applyPending = true
		p.metrics.setApplyPending(true)
		p.armRetryLocked()
		p.l.Warnf("Alert rules are written but not applied yet: %s.", err)
		p.metrics.recordError(stageApply)
	}
}

// errRuleUIDTaken reports that PMM could not establish whether its rule UIDs are still its own.
var errRuleUIDTaken = errors.New("could not check who owns the built-in rule UIDs")

// errDeferredToLeader reports that this node wrote the file and left applying it to the leader. It
// is the design working, not a failure: the rules live in a database every node shares, so one
// ingestion serves the cluster, and a follower restarting its own Grafana would be pure disruption.
// It is returned rather than silently ignored so the caller can tell it apart from an apply that
// genuinely failed and is owed a retry.
var errDeferredToLeader = errors.New("left to the leader to apply")

// reportConflictsLocked logs and counts UIDs that belong to someone else, once per distinct set so
// a standing conflict does not fill the log every tick. Called with m held.
func (p *Provisioner) reportConflictsLocked(notOurs map[string]string) {
	if len(notOurs) == 0 {
		p.reportedConflicts = ""
		p.metrics.setConflicts(0)
		return
	}

	uids := make([]string, 0, len(notOurs))
	for uid := range notOurs {
		uids = append(uids, uid)
	}
	sort.Strings(uids)

	p.metrics.setConflicts(len(uids))

	fingerprint := strings.Join(uids, ",")
	if fingerprint == p.reportedConflicts {
		return
	}
	p.reportedConflicts = fingerprint

	for _, uid := range uids {
		owner := notOurs[uid]
		if owner == "" {
			owner = "a rule created in the interface"
		} else {
			owner = "a rule with provenance " + owner
		}
		p.l.Errorf("Not provisioning the built-in alert rule %q: that UID already belongs to %s. "+
			"PMM leaves it alone rather than overwrite it; delete or re-point that rule to get the "+
			"built-in one back.", uid, owner)
	}
}

// render builds the file content for this server, and reports which bundles it covers.
func (p *Provisioner) render(ctx context.Context) ([]byte, map[string]bool, error) {
	// Read the shipped templates directly rather than the service's merged view: that view lets a
	// user file in /srv/alerting/templates shadow a built-in template by name, and a user must not
	// be able to change what PMM provisions.
	loaded, err := loadBuiltinTemplatesFromDir(p.templatesPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load built-in templates: %w", err)
	}

	templates := make(map[string]models.Template, len(loaded))
	for _, template := range loaded {
		templates[template.Name] = *template
	}

	settings, err := models.GetSettings(p.db)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get settings: %w", err)
	}

	datasourceUID, err := p.grafanaDB.ResolveDatasourceUID(ctx)
	if err != nil {
		return nil, nil, err
	}

	// A UID is only PMM's while PMM provisions it, so check before claiming one back.
	notOurs, err := p.grafanaDB.ConflictingRules(ctx, catalogUIDs())
	if err != nil {
		// The datasource resolve above uses the same connection, so it has already failed closed on
		// anything that would break this query. Treat a failure here the same way rather than
		// writing a file whose safety is unknown.
		return nil, nil, fmt.Errorf("%w: %w", errRuleUIDTaken, err)
	}
	p.reportConflictsLocked(notOurs)

	content, err := renderProvisioningFile(templates, datasourceUID, settings, p.gates, notOurs)
	if err != nil {
		return nil, nil, err
	}

	bundles := make(map[string]bool, len(builtinBundles))
	for _, bundle := range builtinBundles {
		bundles[bundle.id] = settings.IsAlertingEnabled() && bundle.enabled(p.gates)
	}

	return content, bundles, nil
}

// write puts the content on disk, reporting what was there before so a failed apply can be rolled
// back, and whether anything actually changed. Unchanged means Grafana already read this exact file
// when it last started, so nothing has to be done to make it pick the rules up.
func (p *Provisioner) write(content []byte) ([]byte, bool, error) {
	err := dir.CreateDataDir(p.dirPath, provisioningDirPerm)
	if err != nil {
		return nil, false, err
	}

	path := p.filePath()
	previous, err := os.ReadFile(path) //nolint:gosec
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, false, err
	}

	if previous != nil && bytes.Equal(previous, content) {
		return previous, false, nil
	}

	// Grafana may be reading this directory right now, so the file is replaced atomically rather
	// than truncated and rewritten.
	err = dir.WriteFileAtomic(path, content, provisioningFilePerm)
	if err != nil {
		return nil, false, err
	}

	p.l.Infof("Wrote alert rule provisioning file %s.", path)
	return previous, true, nil
}

// apply makes Grafana pick up a file that has just changed.
//
// Grafana reads its provisioning only while starting, and pmm-managed has no Grafana credentials
// with which to ask for a reload, so restarting Grafana is the only lever there is. It is rarely
// needed: the file changes only when the configuration or the shipped templates change, and both of
// those arrive by recreating the container, which restarts Grafana anyway. What is
// left is the boot where the rendered content changed, and the case where the datasource resolved
// late.
func (p *Provisioner) apply(ctx context.Context, trigger provisioningTrigger, previous []byte) error {
	running, err := p.supervisord.IsSupervisedServiceRunning(grafanaProgramName)
	if err != nil {
		p.l.Debugf("Could not determine Grafana's state: %s.", err)
	}

	switch {
	case running == nil:
		// The state could not be determined, which supervisord's own contract says to treat as
		// "leave it alone". The usual cause is that Grafana is not configured yet, on a container
		// whose first boot has not reached UpdateConfiguration: supervisord starts it moments later
		// and it reads this file as it goes.
		p.l.Debugf("Grafana's state is unknown, leaving it alone on %s.", trigger)
		return nil

	case !*running:
		// Not running, and supervisord will not start it: this is FATAL or STOPPED, the states
		// parseStatus documents as "will not be restarted". Nobody is coming, so PMM has to be the
		// one to start it - otherwise a Grafana that died on a file PMM has since repaired stays
		// dead, taking the whole interface with it.
		//
		// Deliberately not leader-gated. The gate exists to elect a single actor for a disruptive
		// action across nodes sharing one database; a dead Grafana serves nobody, so there is no
		// blast radius to contain, and at startup no node is leader yet.
		p.l.Warnf("Grafana is down and supervisord will not restart it; starting it to apply the alert rules.")
		err = p.supervisord.StartSupervisedService(grafanaProgramName)
		if err != nil {
			return fmt.Errorf("failed to start Grafana: %w", err)
		}
		return p.waitForGrafana(ctx)

	default:
		// One restart applies the change for the whole cluster: this node's Grafana ingests the file
		// into the database every node shares, and the other schedulers pick the rules up from
		// there. So the gate elects a single actor rather than keeping a pool of backends alive -
		// PMM HA is active-passive, HAProxy routes to the leader alone, and the standbys serve
		// nobody. Leadership is the only single-actor primitive the cluster has, and IsLeader
		// reports true on a standalone server.
		if !p.leader.IsLeader() {
			return fmt.Errorf("%w on %s", errDeferredToLeader, trigger)
		}

		return p.restartGrafana(ctx, previous)
	}
}

// restartGrafana restarts Grafana and waits for it to answer again, restoring the previous file if
// it does not.
func (p *Provisioner) restartGrafana(ctx context.Context, previous []byte) error {
	p.l.Infof("Restarting Grafana to apply alert rule changes.")

	err := p.supervisord.RestartSupervisedService(grafanaProgramName)
	if err != nil {
		return fmt.Errorf("failed to restart Grafana: %w", err)
	}

	err = p.waitForGrafana(ctx)
	if err != nil {
		p.rollback(previous)
		return fmt.Errorf("grafana did not come back after the restart: %w", err)
	}

	return nil
}

func (p *Provisioner) waitForGrafana(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, p.readyTimeout)
	defer cancel()

	ticker := time.NewTicker(p.readyPollInterval)
	defer ticker.Stop()

	// Ask straight away: Grafana is often back before the first tick, and a restart that has
	// already succeeded should not be reported as pending for a whole poll interval.
	lastErr := p.grafana.IsReady(ctx)
	if lastErr == nil {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			if lastErr != nil {
				return lastErr
			}
			return ctx.Err()

		case <-ticker.C:
			lastErr = p.grafana.IsReady(ctx)
			if lastErr == nil {
				return nil
			}
		}
	}
}

// rollback puts back the content Grafana last started from, so that a server left in this state
// still has a Grafana that starts.
func (p *Provisioner) rollback(previous []byte) {
	if previous == nil {
		err := os.Remove(p.filePath())
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			p.l.Errorf("Failed to remove the alert rule provisioning file: %s.", err)
		}
		return
	}

	err := dir.WriteFileAtomic(p.filePath(), previous, provisioningFilePerm)
	if err != nil {
		p.l.Errorf("Failed to restore the previous alert rule provisioning file: %s.", err)
		return
	}
	p.l.Warnf("Restored the previous alert rule provisioning file.")
}

func (p *Provisioner) filePath() string {
	return filepath.Join(p.dirPath, provisioningFileName)
}

func contentHash(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
