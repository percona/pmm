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
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"
)

// provisionerFixture wires a Provisioner up to mocks and a temporary directory, so the reconcile
// logic can be exercised without a server.
type provisionerFixture struct {
	provisioner *Provisioner
	grafana     *mockGrafanaProvisioningClient
	supervisord *mockSupervisordService
	leader      *mockLeaderService
	dbMock      sqlmock.Sqlmock
	// gfMock stands in for Grafana's own database, which the provisioner reads to resolve the
	// datasource UID and to check who owns its rule UIDs.
	gfMock sqlmock.Sqlmock
	dir    string
}

// settingsJSON is what the settings table holds. Percona Alerting is the only setting this feature
// still reads; the two per-bundle toggles are environment variables.
func settingsJSON(alerting bool) string {
	value := "false"
	if alerting {
		value = "true"
	}
	return `{"alerting":{"enabled":` + value + `}}`
}

func newProvisionerFixture(t *testing.T, haEnabled bool) *provisionerFixture {
	t.Helper()

	sqlDB, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	grafana := newMockGrafanaProvisioningClient(t)
	supervisord := newMockSupervisordService(t)
	leader := newMockLeaderService(t)
	dir := t.TempDir()

	provisioner := NewProvisioner(ProvisionerParams{
		DB:                     reform.NewDB(sqlDB, postgresql.Dialect, nil),
		GrafanaCli:             grafana,
		Supervisord:            supervisord,
		Leader:                 leader,
		HAEnabled:              haEnabled,
		HAAlertsEnabled:        true,
		ComponentAlertsEnabled: true,
		Dir:                    dir,
		TemplatesDir:           shippedTemplatesDir,
	})
	// Pre-resolve the datasource. Without this every test would try to reach a Grafana database
	// that is not there, and an unreachable database now fails closed, so tests about rendering,
	// applying and rollback would all be testing datasource resolution instead of themselves.
	gfDB, gfMock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = gfDB.Close() })

	provisioner.grafanaDB = newGrafanaReader("dsn-not-used", logrus.WithField("test", t.Name()))
	provisioner.grafanaDB.db = gfDB
	// Pre-resolve the UID so the tests exercise rendering and applying rather than resolution,
	// which has its own suite. The conflict query still runs, queued by expectSettings.
	provisioner.grafanaDB.uid = deriveMetricsDatasourceUID()
	provisioner.grafanaDB.checkedAt = time.Now()
	// Keep a failed restart from holding the test for the real two-minute timeout.
	provisioner.readyTimeout = 50 * time.Millisecond
	provisioner.readyPollInterval = 10 * time.Millisecond

	return &provisionerFixture{
		provisioner: provisioner,
		grafana:     grafana,
		supervisord: supervisord,
		leader:      leader,
		dbMock:      dbMock,
		gfMock:      gfMock,
		dir:         dir,
	}
}

// expectSettings queues the two reads every reconcile makes: PMM's own settings, and Grafana's
// record of who owns the catalog UIDs (no conflicts unless a test says otherwise).
func (f *provisionerFixture) expectSettings(times int, alerting bool) {
	for range times {
		f.dbMock.ExpectQuery("SELECT settings FROM settings").
			WillReturnRows(sqlmock.NewRows([]string{"settings"}).AddRow(settingsJSON(alerting)))
		f.expectNoConflicts(1)
	}
}

// expectNoConflicts queues ownership queries that come back empty.
func (f *provisionerFixture) expectNoConflicts(times int) {
	for range times {
		f.gfMock.ExpectQuery("FROM alert_rule").
			WillReturnRows(sqlmock.NewRows([]string{"uid", "provenance"}))
	}
}

func (f *provisionerFixture) fileContent(t *testing.T) string {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(f.dir, provisioningFileName))
	require.NoError(t, err)
	return string(content)
}

func TestProvisionerWritesAndAppliesOnStartup(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)
	f.expectSettings(1, true)

	// A nil status is "state unknown", which supervisord's contract says to leave alone: the file
	// is written and whoever starts Grafana next reads it.
	f.supervisord.On("IsSupervisedServiceRunning", grafanaProgramName).Return(nil, nil)

	f.provisioner.reconcile(context.Background(), triggerStartup)

	content := f.fileContent(t)
	assert.Contains(t, content, "PMM High Availability")
	assert.Contains(t, content, "PMM Server")
}

// TestProvisionerOnStandaloneWritesComponentsOnly is the standalone half of the contract: PMM's own
// component rules apply everywhere, the High Availability ones only to a cluster.
func TestProvisionerOnStandaloneWritesComponentsOnly(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, false)
	f.expectSettings(1, true)
	f.supervisord.On("IsSupervisedServiceRunning", grafanaProgramName).Return(nil, nil)

	f.provisioner.reconcile(context.Background(), triggerStartup)

	content := f.fileContent(t)
	assert.Contains(t, content, "PMM Server")
	assert.NotContains(t, content, "PMM High Availability")
	// The High Availability rules are not merely absent from the file: they are listed for
	// deletion, because Grafana keeps a rule that only disappears.
	assert.Contains(t, content, `"uid": "pmm-ha-no-leader"`)
}

// TestProvisionerRemovesEverythingWhenAlertingIsOff covers the gate that stops PMM-owned rules
// paging from a feature the user has switched off.
func TestProvisionerRemovesEverythingWhenAlertingIsOff(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)
	f.expectSettings(1, false)
	f.supervisord.On("IsSupervisedServiceRunning", grafanaProgramName).Return(nil, nil)

	f.provisioner.reconcile(context.Background(), triggerStartup)

	content := f.fileContent(t)
	assert.Contains(t, content, `"groups": []`)
	assert.Contains(t, content, `"uid": "pmm-clickhouse-down"`)
}

// TestProvisionerKeepsStateOutOfGrafanasReach guards a detail found on a live server: Grafana reads
// every file in its provisioning directory and warns about anything that is not .yaml, .yml or
// .json, once per start and per reload.
func TestProvisionerKeepsStateOutOfGrafanasReach(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)
	f.expectSettings(1, true)
	f.supervisord.On("IsSupervisedServiceRunning", grafanaProgramName).Return(nil, nil)

	f.provisioner.reconcile(context.Background(), triggerStartup)

	entries, err := os.ReadDir(f.dir)
	require.NoError(t, err)
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	assert.Equal(t, []string{provisioningFileName}, names,
		"only the provisioning file itself may live in the directory Grafana reads")
}

// TestProvisionerToleratesConcurrentCallers is a regression test for a race found in review. Three
// goroutines reach the provisioner in production - the startup path, which setup() retries in a
// goroutine of its own; the gRPC handler serving a settings change; and the Run loop - and they
// share both the counters and the files. Run this with -race.
func TestProvisionerToleratesConcurrentCallers(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)
	f.dbMock.MatchExpectationsInOrder(false)
	f.gfMock.MatchExpectationsInOrder(false)
	for range 40 {
		f.dbMock.ExpectQuery("SELECT settings FROM settings").
			WillReturnRows(sqlmock.NewRows([]string{"settings"}).AddRow(settingsJSON(true)))
	}
	f.expectNoConflicts(40)
	f.supervisord.On("IsSupervisedServiceRunning", grafanaProgramName).Return(nil, nil).Maybe()

	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			f.provisioner.ProvisionAtStartup(context.Background())
		})

		wg.Go(func() {
			f.provisioner.reconcile(context.Background(), triggerTick)
		})

		wg.Go(func() {
			f.provisioner.reconcile(context.Background(), triggerRetry)
		})
	}
	wg.Wait()

	// Whichever caller won, the file is the one the renderer produces, never a partial write.
	content := f.fileContent(t)
	require.NoError(t, validateProvisioningFile([]byte(content)))
}

func TestProvisionerDoesNothingWhenNothingChanged(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)
	f.expectSettings(2, true)
	f.supervisord.On("IsSupervisedServiceRunning", grafanaProgramName).Return(nil, nil).Once()

	f.provisioner.reconcile(context.Background(), triggerStartup)
	first := f.fileContent(t)

	// The second pass must not consult supervisord or Grafana at all: steady state is the common
	// case, and a reconcile that touched Grafana every five minutes would be a liability.
	f.provisioner.reconcile(context.Background(), triggerTick)
	assert.Equal(t, first, f.fileContent(t))
}

func TestProvisionerRestartsGrafanaWhenRequested(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)
	f.expectSettings(1, true)

	f.leader.On("IsLeader").Return(true)
	f.supervisord.On("IsSupervisedServiceRunning", grafanaProgramName).Return(new(true), nil)
	f.supervisord.On("RestartSupervisedService", grafanaProgramName).Return(nil)
	f.grafana.On("IsReady", mock.Anything).Return(nil)

	f.provisioner.reconcile(context.Background(), triggerStartup)
}

// TestProvisionerTickWaitsForTheLeader is the guard against a settings change bouncing Grafana on
// every node at once, when one restart already applies it for the whole cluster.
func TestProvisionerTickWaitsForTheLeader(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)
	f.expectSettings(3, true)

	f.supervisord.On("IsSupervisedServiceRunning", grafanaProgramName).Return(new(true), nil)
	f.leader.On("IsLeader").Return(false)

	for range 3 {
		f.provisioner.reconcile(context.Background(), triggerTick)
	}
	f.supervisord.AssertNotCalled(t, "RestartSupervisedService", grafanaProgramName)
}

// TestProvisionerRollsBackAFailedRestart proves the promise the whole design rests on: PMM must
// never leave Grafana unable to start.
func TestProvisionerRollsBackAFailedRestart(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)

	// A first pass leaves a known good file in place.
	f.expectSettings(1, true)
	f.supervisord.On("IsSupervisedServiceRunning", grafanaProgramName).Return(nil, nil).Once()
	f.provisioner.reconcile(context.Background(), triggerStartup)
	good := f.fileContent(t)

	// Then a real change - Percona Alerting switched off empties the file - which Grafana refuses
	// to come back from. Only a change reaches the apply step: an identical render is a no-op.
	f.expectSettings(1, false)
	f.leader.On("IsLeader").Return(true)
	f.supervisord.On("IsSupervisedServiceRunning", grafanaProgramName).Return(new(true), nil)
	f.supervisord.On("RestartSupervisedService", grafanaProgramName).Return(nil)
	f.grafana.On("IsReady", mock.Anything).Return(errors.New("connection refused"))

	f.provisioner.reconcile(context.Background(), triggerStartup)

	assert.Equal(t, good, f.fileContent(t), "the file Grafana last started from must be restored")
}

func TestProvisionerRefusesToWriteWhenItCannotRender(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)

	// Anything that stops a complete file being produced - here settings that cannot be read - has
	// to leave the directory untouched. Handing Grafana a partial file would stop it starting, and
	// that takes the whole PMM user interface with it.
	f.dbMock.ExpectQuery("SELECT settings FROM settings").WillReturnError(errors.New("database is down"))

	f.provisioner.reconcile(context.Background(), triggerStartup)

	_, err := os.Stat(filepath.Join(f.dir, provisioningFileName))
	assert.True(t, os.IsNotExist(err), "no file should have been written")

	// The failure has to be visible: provisioned rules are built to stay quiet on an execution
	// error, so the metric is the only thing that separates "nothing changed" from "nothing worked".
	assert.Equal(t, stateError, bundleState(t, f.provisioner, componentsBundleID))
}

func TestProvisionerRefusesToGuessTheDatasourceUID(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)
	f.expectSettings(1, true)

	// A UID that cannot be read is not a UID that can be guessed: deriving it is wrong on every
	// server whose datasource predates Grafana 8.3.4, and such a rule reports healthy while querying
	// nothing. No rules at all is the failure someone can actually see.
	dsDB, dsMock, err := sqlmock.New()
	require.NoError(t, err)
	defer dsDB.Close()
	dsMock.ExpectQuery("SELECT uid FROM data_source").WillReturnError(errors.New("connection refused"))

	f.provisioner.grafanaDB = newGrafanaReader("dsn-not-used", logrus.WithField("test", t.Name()))
	f.provisioner.grafanaDB.db = dsDB

	f.provisioner.reconcile(context.Background(), triggerStartup)

	_, err = os.Stat(filepath.Join(f.dir, provisioningFileName))
	assert.True(t, err != nil && os.IsNotExist(err), "no file should have been written")

	assert.Equal(t, stateError, bundleState(t, f.provisioner, componentsBundleID))
	assert.InDelta(t, 1, errorCount(t, f.provisioner, stageDatasource), 0,
		"the datasource stage is what separates an unreachable database from a template bug")
	assert.Zero(t, errorCount(t, f.provisioner, stageRender))

	// And it must come back quickly rather than waiting out a five-minute tick.
	assert.NotNil(t, f.provisioner.retryAfter(), "a retry should be owed")
}

func TestProvisionerDatasourceRetryBacksOff(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)

	assert.Nil(t, f.provisioner.retryAfter(), "nothing is owed before a failure")

	f.provisioner.armRetryLocked()
	assert.Equal(t, datasourceRetryInitial, f.provisioner.retryBackoff)

	f.provisioner.armRetryLocked()
	assert.Equal(t, datasourceRetryInitial*datasourceRetryFactor, f.provisioner.retryBackoff)

	// A database that stays down must not back off past the cap, or recovery would take longer than
	// the ordinary tick it exists to beat.
	for range 20 {
		f.provisioner.armRetryLocked()
	}
	assert.Equal(t, datasourceRetryMax, f.provisioner.retryBackoff)
	assert.Less(t, datasourceRetryMax, reconcileInterval)
}

func TestProvisionerRunPicksUpARetryArmedAtStartup(t *testing.T) {
	t.Parallel()

	// ProvisionAtStartup runs before Run exists, so a backoff it armed has to be read before the
	// first select rather than after a whole tick has passed.
	f := newProvisionerFixture(t, true)
	f.expectSettings(1, true)
	f.supervisord.On("IsSupervisedServiceRunning", grafanaProgramName).Return(nil, nil).Maybe()

	f.provisioner.retryBackoff = 10 * time.Millisecond

	go f.provisioner.Run(t.Context())

	assert.Eventually(t, func() bool {
		_, err := os.Stat(filepath.Join(f.dir, provisioningFileName))
		return err == nil
	}, 5*time.Second, 10*time.Millisecond, "the retry should have reconciled well inside one tick")
}

// TestProvisionerStartsGrafanaWhenSupervisordWillNot covers the case that used to leave a server
// with its interface down: Grafana died on a file PMM has since repaired, supervisord gave up
// retrying, and nothing was left to start it.
func TestProvisionerStartsGrafanaWhenSupervisordWillNot(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)
	f.expectSettings(1, true)

	// false is what parseStatus returns for FATAL and STOPPED - "will not be restarted".
	f.supervisord.On("IsSupervisedServiceRunning", grafanaProgramName).Return(new(false), nil)
	f.supervisord.On("StartSupervisedService", grafanaProgramName).Return(nil)
	f.grafana.On("IsReady", mock.Anything).Return(nil)

	f.provisioner.reconcile(context.Background(), triggerStartup)

	// Deliberately no leader stub: starting a dead Grafana is a repair, not a rollout action, so it
	// is not leader-gated. A mockLeaderService call would fail the test.
	f.supervisord.AssertCalled(t, "StartSupervisedService", grafanaProgramName)
	f.supervisord.AssertNotCalled(t, "RestartSupervisedService", grafanaProgramName)
}

// TestProvisionerLeavesGrafanaAloneWhenStateIsUnknown pins supervisord's documented contract: a nil
// status means "leave it alone". The usual cause is a first boot where Grafana is not configured
// yet, and supervisord starts it moments later with this file already in place.
func TestProvisionerLeavesGrafanaAloneWhenStateIsUnknown(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)
	f.expectSettings(1, true)
	f.supervisord.On("IsSupervisedServiceRunning", grafanaProgramName).Return(nil, nil)

	f.provisioner.reconcile(context.Background(), triggerStartup)

	require.NotEmpty(t, f.fileContent(t), "the file is still written")
	f.supervisord.AssertNotCalled(t, "StartSupervisedService", grafanaProgramName)
	f.supervisord.AssertNotCalled(t, "RestartSupervisedService", grafanaProgramName)
}

// TestProvisionerLeavesRulesItDoesNotOwnAlone is the guard against the two harms a squatted UID
// causes: a rule under provenance "api" makes Grafana refuse to start, and a rule a user made in
// the interface would be silently overwritten or deleted.
func TestProvisionerLeavesRulesItDoesNotOwnAlone(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, false) // standalone: the components bundle only
	f.dbMock.ExpectQuery("SELECT settings FROM settings").
		WillReturnRows(sqlmock.NewRows([]string{"settings"}).AddRow(settingsJSON(true)))
	f.gfMock.ExpectQuery("FROM alert_rule").
		WillReturnRows(sqlmock.NewRows([]string{"uid", "provenance"}).
			AddRow("pmm-clickhouse-down", "api").
			AddRow("pmm-grafana-down", "")) // made in the interface: no provenance at all
	f.supervisord.On("IsSupervisedServiceRunning", grafanaProgramName).Return(nil, nil)

	f.provisioner.reconcile(context.Background(), triggerStartup)

	content := f.fileContent(t)
	assert.NotContains(t, content, "pmm-clickhouse-down", "an api-owned UID must not be claimed")
	assert.NotContains(t, content, "pmm-grafana-down", "a user's own rule must not be touched")
	assert.Contains(t, content, "pmm-qan-api2-down", "the rules PMM does own are still provisioned")
	assert.InDelta(t, 2, conflictCount(t, f.provisioner), 0)
}

// TestProvisionerOmitsSquattedUIDsFromDeletions is the other half: a disabled bundle lists its UIDs
// for deletion, and deleting a rule PMM does not own would destroy someone else's work.
func TestProvisionerOmitsSquattedUIDsFromDeletions(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)
	f.dbMock.ExpectQuery("SELECT settings FROM settings").
		WillReturnRows(sqlmock.NewRows([]string{"settings"}).AddRow(settingsJSON(false))) // alerting off
	f.gfMock.ExpectQuery("FROM alert_rule").
		WillReturnRows(sqlmock.NewRows([]string{"uid", "provenance"}).AddRow("pmm-ha-no-leader", ""))
	f.supervisord.On("IsSupervisedServiceRunning", grafanaProgramName).Return(nil, nil)

	f.provisioner.reconcile(context.Background(), triggerStartup)

	content := f.fileContent(t)
	assert.NotContains(t, content, "pmm-ha-no-leader", "a squatted UID must not be deleted either")
	assert.Contains(t, content, `"uid": "pmm-clickhouse-down"`, "the rest are still deleted")
}

// TestProvisionerDeferralIsNotAFailure pins the distinction the metrics rest on. A follower that
// leaves applying to the leader is the design working: the rules live in a database every node
// shares, so one ingestion serves the cluster. Counting it would put an apply error on every
// healthy node of every HA cluster, which is exactly what it used to do.
func TestProvisionerDeferralIsNotAFailure(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)
	f.expectSettings(1, true)

	f.supervisord.On("IsSupervisedServiceRunning", grafanaProgramName).Return(new(true), nil)
	f.leader.On("IsLeader").Return(false)

	f.provisioner.reconcile(context.Background(), triggerTick)

	assert.Zero(t, errorCount(t, f.provisioner, stageApply), "a deferral must not count as a failure")
	assert.Equal(t, stateWritten, bundleState(t, f.provisioner, haBundleID))
	assert.False(t, f.provisioner.applyPending, "the leader applies, so this node owes no retry")
	assert.Zero(t, f.provisioner.retryBackoff)
}

// TestProvisionerRetriesAnApplyItOwes is the D2 regression test. A start that fails leaves the file
// on disk already correct, so every later reconcile used to return early on "unchanged" and PMM
// never tried again - leaving a Grafana that supervisord had given up on dead for good, which is
// the outage the start path exists to prevent.
func TestProvisionerRetriesAnApplyItOwes(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)
	f.expectSettings(2, true)

	f.supervisord.On("IsSupervisedServiceRunning", grafanaProgramName).Return(new(false), nil)
	f.supervisord.On("StartSupervisedService", grafanaProgramName).
		Return(errors.New("boom")).Once()
	f.supervisord.On("StartSupervisedService", grafanaProgramName).Return(nil).Once()
	f.grafana.On("IsReady", mock.Anything).Return(nil)

	f.provisioner.reconcile(context.Background(), triggerStartup)

	require.True(t, f.provisioner.applyPending, "a failed start is owed a retry")
	assert.Equal(t, statePending, bundleState(t, f.provisioner, haBundleID))
	assert.InDelta(t, 1, errorCount(t, f.provisioner, stageApply), 0)
	assert.Positive(t, f.provisioner.retryBackoff, "the retry must be armed")

	// The content has not changed, so this is precisely the reconcile that used to return early.
	f.provisioner.reconcile(context.Background(), triggerRetry)

	f.supervisord.AssertNumberOfCalls(t, "StartSupervisedService", 2)
	assert.False(t, f.provisioner.applyPending, "a successful retry clears the debt")
	assert.Equal(t, stateWritten, bundleState(t, f.provisioner, haBundleID))
}

// TestProvisionerStillDoesNothingWhenNothingChanged guards the other side of that condition: the
// steady-state no-op must survive. A server that reconciles every five minutes forever must not
// touch Grafana once its file is correct.
func TestProvisionerStillDoesNothingWhenNothingChanged(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)
	f.expectSettings(3, true)

	f.supervisord.On("IsSupervisedServiceRunning", grafanaProgramName).Return(new(true), nil).Once()
	f.leader.On("IsLeader").Return(true).Once()
	f.supervisord.On("RestartSupervisedService", grafanaProgramName).Return(nil).Once()
	f.grafana.On("IsReady", mock.Anything).Return(nil).Once()

	for range 3 {
		f.provisioner.reconcile(context.Background(), triggerTick)
	}

	// One restart for the first write; the two reconciles after it are true no-ops.
	f.supervisord.AssertNumberOfCalls(t, "RestartSupervisedService", 1)
	assert.Equal(t, stateWritten, bundleState(t, f.provisioner, haBundleID))
}

// conflictCount reads the conflicting-rules gauge.
func conflictCount(t *testing.T, p *Provisioner) float64 {
	t.Helper()

	registry := prom.NewPedanticRegistry()
	require.NoError(t, registry.Register(p.Collector()))

	families, err := registry.Gather()
	require.NoError(t, err)

	for _, family := range families {
		if family.GetName() == "pmm_alerting_provisioning_conflicting_rules" {
			return family.GetMetric()[0].GetGauge().GetValue()
		}
	}
	return -1
}

// errorCount reads the counter the collector reports for one failure stage.
func errorCount(t *testing.T, p *Provisioner, stage string) float64 {
	t.Helper()

	registry := prom.NewPedanticRegistry()
	require.NoError(t, registry.Register(p.Collector()))

	families, err := registry.Gather()
	require.NoError(t, err)

	for _, family := range families {
		if family.GetName() != "pmm_alerting_provisioning_errors_total" {
			continue
		}
		for _, metric := range family.GetMetric() {
			for _, label := range metric.GetLabel() {
				if label.GetName() == "stage" && label.GetValue() == stage {
					return metric.GetCounter().GetValue()
				}
			}
		}
	}
	return -1
}

// bundleState reads the state label the collector reports for one bundle.
func bundleState(t *testing.T, p *Provisioner, bundleID string) string {
	t.Helper()

	registry := prom.NewPedanticRegistry()
	require.NoError(t, registry.Register(p.Collector()))

	families, err := registry.Gather()
	require.NoError(t, err)

	for _, family := range families {
		if family.GetName() != "pmm_alerting_provisioning_info" {
			continue
		}
		for _, metric := range family.GetMetric() {
			var state, bundle string
			for _, label := range metric.GetLabel() {
				switch label.GetName() {
				case "bundle":
					bundle = label.GetValue()
				case "state":
					state = label.GetValue()
				}
			}
			if bundle == bundleID {
				return state
			}
		}
	}
	return ""
}
