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

	// ReconcileInterval is slow on purpose. The tick is the only thing that notices anything
	// changing while the process runs, and the bundle toggles are not it: those are read once at
	// start and cannot change without recreating the container. Three things can:
	//
	//   - the Percona Alerting setting, which gates both bundles;
	//   - ownership of a rule UID, which a user can take while PMM is not provisioning it and give
	//     back at any time;
	//   - the Metrics datasource UID, whose re-check is deliberately tied to this same interval.
	//
	// Do not assume the first of those carries the tick on its own: a deployment can pin Percona
	// Alerting with PMM_ENABLE_ALERTING, which makes the setting unchangeable through the API and
	// leaves the other two as the only reasons to wake up.
	//
	// Nothing triggers a reconcile directly. A settings change reaches one node of a cluster, but
	// every node has to converge, because each writes its own copy of the file and its own Grafana
	// reads that copy at its next start. A follower left behind would revert the cluster, which is
	// also why writing is not leader-gated. Five minutes is the compromise: soon enough that a
	// change feels like it worked, rare enough to be invisible.
	reconcileInterval = 5 * time.Minute

	// DeferralWindow is how long a follower that left an apply to the leader keeps it in mind, in
	// case it becomes the leader itself before the old one has applied. The leader applies at its
	// next tick, at most one jittered interval after the change; a node that took over before then
	// reaches its own next tick at most one more interval later. Past that the old leader has had
	// its turn, and applying again would only restart Grafana for nothing.
	deferralWindow = 2 * (reconcileInterval + reconcileInterval/jitterFraction)

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

	// MaxApplyAttemptsPerRevision is how many times one rendered revision may take Grafana down
	// before PMM stops offering it. Applying means restarting Grafana, and a restart it does not
	// come back from costs a whole grafanaReadyTimeout with the interface unavailable. Without a
	// budget that repeats for as long as the content stays the same, because a failed apply leaves
	// applyPending set and the next reconcile therefore skips its "nothing changed" early return.
	//
	// Two rather than one: the restart may have failed for a reason that has nothing to do with the
	// rules - a Grafana database briefly out of reach, a host too loaded to start one inside the
	// timeout - and a single quick retry settles that far more cheaply than leaving a good revision
	// unapplied. Beyond that the evidence points at the content, and repeating only costs
	// availability. The count is per revision and in memory: any change to the rendered file, and
	// any restart of pmm-managed, offers the content afresh.
	maxApplyAttemptsPerRevision = 2
)

// provisioningTrigger says what caused a reconcile, which decides how far the provisioner may go to
// make Grafana pick the file up.
type provisioningTrigger int

const (
	// TriggerStartup is the first reconcile of a boot, which setup() asks Run for, and by then
	// Grafana is normally already running: setup() writes grafana.ini and starts it from
	// UpdateSettingsFromEnv, before it asks. Applying therefore usually does mean restarting Grafana - once, on the boot where the
	// content changed, and not at all on a restart that renders the same file. That restart does
	// not wait for a leader: it settles a debt only this node can see, recorded in startupApplyOwed.
	triggerStartup provisioningTrigger = iota
	// TriggerTick is the periodic reconcile. It never restarts Grafana except as the last-resort
	// leader fallback.
	triggerTick
	// TriggerRetry is the fast follow-up to a reconcile that could not finish: an unresolvable
	// datasource UID, a squatted rule UID, or an apply action that failed. It carries no restart
	// permission of its own: whether it may restart without the leader depends on whose work it is
	// finishing. A boot whose render failed still owes its restart, while a shared Grafana database
	// coming back at runtime reaches every node at once and needs the gate that elects one to act.
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

	// startup is closed by ProvisionAtStartup to ask Run for the startup reconcile, and
	// startupOnce makes sure that happens once however often setup() is retried.
	startup     chan struct{}
	startupOnce sync.Once

	// m serialises the work. Run is the only caller in production, but reconcile must stay safe to
	// call from more than one goroutine: it writes the file and the counters below.
	m sync.Mutex

	// reportedConflicts is the set of squatted UIDs last logged, so a standing conflict is reported
	// once rather than on every tick.
	reportedConflicts string
	// retryBackoff is how long to wait before trying again after a reconcile that could not finish:
	// an unresolvable datasource UID, a squatted rule UID, or an apply action that failed. Zero
	// means no retry is owed. It is held through the recovery that follows, so the whole recovery
	// runs on the fast cadence instead of handing the last step back to the five-minute tick, and
	// it keeps growing across consecutive failures of any stage, an apply included: it is cleared
	// only once a reconcile ends with nothing owed.
	retryBackoff time.Duration
	// applyPending records that an apply action PMM performs itself failed, so the next reconcile
	// must try again even though the file on disk is already the content it would write.
	applyPending bool
	// deferredAt is when this node last wrote content and left applying it to the leader, and zero
	// when there is no such deferral. It is kept apart from applyPending, which would report every
	// healthy follower as pending, and it expires after deferralWindow: see deferralOwedLocked.
	deferredAt time.Time
	// startupApplyOwed records that this process has started but has not yet established that its
	// own Grafana runs on the file currently on disk. Grafana starts before the first reconcile and
	// reads its provisioning only then, so a boot that changes the content owes a restart of the
	// local Grafana whatever this node's role: no node is leader yet at that point, a starting node
	// serves no traffic, and no other node can tell that this one's Grafana is behind. Cleared once
	// a reconcile finds the file unchanged or applies it, so the debt survives a render that fails
	// at boot and is settled by the retry that finishes the recovery. From then on only the leader
	// restarts: a later change is visible to every node, and the leader applies it for all.
	startupApplyOwed bool
	// rejectedHash is the content Grafana last refused to come back from, and rejectedApplies is
	// how many times it has been tried. Together they stop PMM from spending the interface on a
	// revision already shown to break it: see maxApplyAttemptsPerRevision. Rendering anything else
	// clears both, so a revision is only ever held against itself.
	rejectedHash    string
	rejectedApplies int
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
		startup:     make(chan struct{}),
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

	retry := p.retryAfter()
	startup := p.startup

	for {
		select {
		case <-ctx.Done():
			err := p.grafanaDB.Close()
			if err != nil {
				p.l.Debugf("Failed to close the Grafana database connection: %s.", err)
			}
			return

		case <-startup:
			// Once only: a closed channel is always ready, and a nil one never is.
			startup = nil
			p.reconcile(ctx, triggerStartup)

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

// ProvisionAtStartup asks Run for the startup reconcile, and returns without waiting for it.
//
// That reconcile may restart Grafana and wait up to readyTimeout for it to answer again: Grafana is
// normally already running by the time PMM gets here, on a fresh container as well as an existing
// one, because setup() writes grafana.ini and starts it from UpdateSettingsFromEnv first. And
// setup() runs before PMM's API servers start, so it must not wait on that. The restart does not wait for a
// leader, and the debt survives a render that fails: the retry that finishes the recovery settles
// it. See startupApplyOwed.
//
// It is safe to call on every attempt of setup(), which is retried every couple of seconds while
// it fails: only the first call counts, so the startup reconcile runs once and every retry after it
// keeps to the backoff Run holds.
func (p *Provisioner) ProvisionAtStartup() {
	p.startupOnce.Do(func() {
		close(p.startup)
	})
}

// reconcile renders the provisioning file, writes it if it changed, and makes Grafana pick it up as
// far as this trigger allows. It never returns an error: a failure is logged and counted, and the
// next tick or retry tries again.
func (p *Provisioner) reconcile(ctx context.Context, trigger provisioningTrigger) {
	// Skip rather than queue behind the reconcile in flight: reconciling is idempotent, so the one
	// already running does the same work.
	if !p.m.TryLock() {
		p.l.Debugf("A reconcile is already running, skipping this %s.", trigger)
		return
	}
	defer p.m.Unlock()

	if trigger == triggerStartup {
		// Nothing has shown yet that this node's Grafana runs on the file currently on disk.
		p.startupApplyOwed = true
	}

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
	if hash != p.rejectedHash {
		// Whatever was rejected is no longer what PMM wants to apply, so it is no longer held
		// against anything. This is the escape hatch: the settings change, the datasource resolves,
		// an upgrade ships different templates, and the budget starts over.
		p.rejectedHash = ""
		p.rejectedApplies = 0
	}
	if p.rejectedApplies >= maxApplyAttemptsPerRevision {
		// Deliberately before the write, not just before the apply. Leaving content Grafana cannot
		// start from on disk would turn the next restart - an upgrade, a host reboot, anything -
		// into an outage nobody connected to alert rules, and the rollback that put the working
		// file back only runs as part of an apply this branch no longer performs.
		p.l.Debugf("Not applying alert rules Grafana already refused to come back from, on %s.", trigger)
		p.recoverGrafanaLocked(ctx)
		return
	}

	previous, changed, err := p.write(content)
	if err != nil {
		p.l.Errorf("Failed to write the alert rule provisioning file: %s.", err)
		p.metrics.recordError(stageWrite)
		return
	}

	p.metrics.setRendered(hash, bundles)
	p.metrics.setWritten(hash)

	if !changed && !p.applyPending && !p.deferralOwedLocked() {
		// The file on disk is already the content we would write, and PMM owes no apply of its own,
		// so there is nothing to do. This is the ordinary case on every restart: Grafana started
		// after the file was last written and read exactly this content.
		p.retryBackoff = 0
		p.startupApplyOwed = false
		return
	}

	err = p.apply(ctx, trigger, previous)
	switch {
	case err == nil:
		p.applyPending = false
		p.metrics.setApplyPending(false)
		p.retryBackoff = 0
		p.startupApplyOwed = false
		p.deferredAt = time.Time{}

	case errors.Is(err, errDeferredToLeader):
		// Not a failure and not this node's work to retry: the change is visible to every node, and
		// the leader applies it for the cluster. Counting this would make every healthy follower
		// look broken.
		p.applyPending = false
		p.metrics.setApplyPending(false)
		p.retryBackoff = 0
		p.deferredAt = time.Now()
		p.l.Debugf("Alert rules written; %s.", err)

	default:
		// An action PMM performs itself failed - starting a Grafana supervisord has given up on, or
		// restarting a running one. Nothing else will retry it: the file is already correct, so
		// every later tick would return early on "unchanged". Arm the backoff instead.
		p.applyPending = true
		p.metrics.setApplyPending(true)
		p.metrics.recordError(stageApply)
		// ApplyPending carries the debt from here on.
		p.deferredAt = time.Time{}

		if errors.Is(err, errGrafanaNotBack) {
			// Grafana was taken down for this content and did not return, and the file has been
			// rolled back to what it last accepted. Charge the revision for it. Any other failure
			// is a command that would not run (awaitGrafana tells the two apart), which says
			// nothing about the content and leaves Grafana where it was, so it costs the revision
			// nothing.
			p.rejectedHash = hash
			p.rejectedApplies++
		}

		if p.rejectedApplies >= maxApplyAttemptsPerRevision {
			// Out of attempts. Stop the fast retry as well: everything it would reach now returns
			// at the gate above, and only a change to the rendered content can make this worth
			// trying again. The slow tick keeps checking for that.
			p.retryBackoff = 0
			p.l.Errorf("Giving up on applying these alert rules: Grafana did not come back %d times, "+
				"and the file it last accepted has been restored. The rules Grafana holds are the "+
				"previous ones, and PMM will try again once the rendered rules change: %s.", p.rejectedApplies, err)

			// The attempt that ran out the budget may have left Grafana FATAL, and with the fast retry
			// gone the next pass is a whole tick away. The file it accepts is already back on disk, so
			// bring it back now rather than leave the interface down until then.
			p.recoverGrafanaLocked(ctx)
			return
		}

		p.armRetryLocked()
		p.l.Warnf("Alert rules are written but not applied yet: %s.", err)
	}
}

// deferralOwedLocked reports whether an apply this node left to the leader is now its own to do,
// because it has become the leader before the deferral expired. Leaving the apply to the leader
// assumes the node that is leader when a follower defers is the one that applies, and leadership
// can move in between: the new leader then finds its file unchanged, the old one defers as a
// follower, and nobody applies, so the shared database keeps the old rules. Called with m held.
func (p *Provisioner) deferralOwedLocked() bool {
	if p.deferredAt.IsZero() {
		return false
	}
	if time.Since(p.deferredAt) > deferralWindow {
		p.deferredAt = time.Time{}
		return false
	}
	return p.leader.IsLeader()
}

// errRuleUIDTaken reports that PMM could not establish whether its rule UIDs are still its own.
var errRuleUIDTaken = errors.New("could not check who owns the built-in rule UIDs")

// errDeferredToLeader reports that this node wrote the file and left applying it to the leader. It
// is the design working, not a failure: the rules live in a database every node shares, so one
// ingestion serves the cluster, and a follower restarting its own Grafana would be pure disruption.
// It is returned rather than silently ignored so the caller can tell it apart from an apply that
// genuinely failed and is owed a retry.
var errDeferredToLeader = errors.New("left to the leader to apply")

// errGrafanaStillStarting reports that the boot restart was put off because Grafana had not
// finished starting. It says nothing about the content, so it does not count against it.
var errGrafanaStillStarting = errors.New("grafana is still starting, so it was not restarted")

// errGrafanaNotBack reports that PMM took Grafana down to apply a file and it did not answer again
// within the timeout, or died so quickly that supervisord gave up on it. It is the one apply failure that says something about the content, so it is
// told apart from a supervisord command that would not run: only this one counts against the
// revision that caused it.
var errGrafanaNotBack = errors.New("grafana did not come back")

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
		p.l.Errorf("Not provisioning the built-in alert rule %s: that UID already belongs to %s. "+
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
	running := p.supervisord.ProgramState(ctx, grafanaProgramName)
	if running == nil {
		// The state could not be determined, and that covers two situations which need opposite
		// answers. The usual one is that Grafana is not configured yet, on a container whose first
		// boot has not reached UpdateConfiguration: supervisord starts it moments later and it
		// reads this file as it goes, so there is nothing to apply. The other is a status command
		// that would not run - ProgramState reports the same nil for output it cannot parse - while
		// Grafana is up and serving the rules it read before this file changed. Treating that as
		// "leave it alone" would discharge the apply for good: the content is already on disk, so
		// every later reconcile finds it unchanged and returns without ever restarting anything.
		//
		// Asking Grafana itself tells the two apart, and it is the question that actually matters:
		// a Grafana that answers is serving rules from a file it has already read, and one that
		// does not is either not started or still starting, which is to say still to read this one.
		err := p.grafana.IsReady(ctx)
		if err != nil {
			p.l.Debugf("Grafana's state is unknown and it is not serving yet, leaving it alone on %s: %s.", trigger, err)
			return nil
		}

		p.l.Warnf("Grafana's state could not be determined, but it is serving, so it is treated as running.")
		running = new(true)
	}

	switch {
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
		err := p.startGrafana(ctx)
		if err != nil {
			// Same reasoning as the restart below: a Grafana that will not start on this file has
			// to be left the one it last accepted, or it can never come up again.
			p.rollback(previous)
			return err
		}
		return nil

	default:
		// One restart applies the change for the whole cluster: this node's Grafana ingests the file
		// into the database every node shares, and the other schedulers pick the rules up from
		// there. So the gate elects a single actor rather than keeping a pool of backends alive -
		// PMM HA is active-passive, HAProxy routes to the leader alone, and the standbys serve
		// nobody. Leadership is the only single-actor primitive the cluster has, and IsLeader
		// reports true on a standalone server.
		//
		// The boot is the exception. Grafana started before this file was written and read the
		// previous one, and no other node can see that: the leader renders the same content from
		// the same shared state and finds its own file unchanged, so it applies nothing. Nor is
		// anyone leader yet while this node starts. A starting node serves no traffic, so
		// restarting its Grafana costs nothing, and it is the only way the new rules reach the
		// cluster before some Grafana happens to restart.
		if !p.startupApplyOwed && !p.leader.IsLeader() {
			return fmt.Errorf("%w on %s", errDeferredToLeader, trigger)
		}

		if p.startupApplyOwed {
			// At boot Grafana may still be starting: supervisord reports it running from the first
			// second, while a cold start can spend minutes in database migrations. Restarting it
			// then only throws that work away, and a Grafana slower than readyTimeout would be cut
			// short on every attempt and charged each time, until the budget ran out and left the
			// server without its rules. So let it finish first. One that does not is left alone and
			// tried again on the backoff, without counting against the content: a Grafana that
			// keeps dying instead ends up FATAL, and the start path above charges that.
			err := p.waitForGrafana(ctx)
			if err != nil {
				return fmt.Errorf("%w: %w", errGrafanaStillStarting, err)
			}
		}

		return p.restartGrafana(ctx, previous)
	}
}

// restartGrafana restarts Grafana and waits for it to answer again, restoring the previous file if
// it does not.
func (p *Provisioner) restartGrafana(ctx context.Context, previous []byte) error {
	p.l.Infof("Restarting Grafana to apply alert rule changes.")

	err := p.supervisord.RestartSupervisedService(ctx, grafanaProgramName)
	err = p.awaitGrafana(ctx, "restart", err)
	if errors.Is(err, errGrafanaNotBack) {
		p.rollback(previous)
	}

	return err
}

// startGrafana starts a Grafana supervisord has given up on and waits for it to answer.
func (p *Provisioner) startGrafana(ctx context.Context) error {
	err := p.supervisord.StartSupervisedService(grafanaProgramName)
	return p.awaitGrafana(ctx, "start", err)
}

// awaitGrafana works out what a supervisorctl restart or start of Grafana did, given the error the
// command returned, and waits for Grafana to answer when that is still possible.
//
// A failed command does not mean the command never ran. Grafana runs with startsecs = 1, and
// supervisorctl exits non-zero with "abnormal termination" or "spawn error" when the process dies
// within that second - which is exactly what content Grafana cannot start from makes it do. So a
// failure is followed by a status check, and only a status that cannot be determined either is
// taken as a command that would not run: that says nothing about the content, so it must not
// count against it. A Grafana supervisord has stopped retrying did not come back, and one it is
// still retrying gets the same wait as after a command that succeeded.
func (p *Provisioner) awaitGrafana(ctx context.Context, command string, cmdErr error) error {
	if cmdErr != nil {
		running := p.supervisord.ProgramState(ctx, grafanaProgramName)
		if running == nil {
			return fmt.Errorf("failed to %s Grafana: %w", command, cmdErr)
		}
		if !*running {
			return fmt.Errorf("%w after supervisorctl %s: %w", errGrafanaNotBack, command, cmdErr)
		}
	}

	err := p.waitForGrafana(ctx)
	if err != nil {
		return fmt.Errorf("%w after supervisorctl %s: %w", errGrafanaNotBack, command, err)
	}

	return nil
}

// recoverGrafanaLocked brings Grafana back if it is down and supervisord will not do it, without
// touching the provisioning file. It is what is left to do for a revision PMM has stopped applying:
// the file on disk is the one Grafana last accepted, so starting it is safe, and a Grafana left
// FATAL by the revision that was given up on would otherwise stay down with nothing else coming for
// it. Called with m held.
func (p *Provisioner) recoverGrafanaLocked(ctx context.Context) {
	running := p.supervisord.ProgramState(ctx, grafanaProgramName)
	if running == nil || *running {
		return
	}

	p.l.Warnf("Grafana is down and supervisord will not restart it; starting it on the file it last accepted.")
	err := p.startGrafana(ctx)
	if err != nil {
		p.l.Errorf("Failed to bring Grafana back: %s.", err)
	}
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
