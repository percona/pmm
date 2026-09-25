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
	"io/fs"
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

// expectProgramStates queues supervisord's answers about Grafana's state, in the order they are
// asked for. A nil entry is a status that cannot be determined.
func (f *provisionerFixture) expectProgramStates(states ...*bool) {
	for _, state := range states {
		f.supervisord.On("ProgramState", mock.Anything, grafanaProgramName).Return(state).Once()
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

	// A nil status is "state unknown", and a Grafana that does not answer either has not read any
	// file yet, so it is left alone: the file is written and whoever starts Grafana next reads it.
	f.supervisord.On("ProgramState", mock.Anything, grafanaProgramName).Return(nil)
	f.grafana.On("IsReady", mock.Anything).Return(errors.New("connection refused"))

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
	f.supervisord.On("ProgramState", mock.Anything, grafanaProgramName).Return(nil)
	f.grafana.On("IsReady", mock.Anything).Return(errors.New("connection refused"))

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
	f.supervisord.On("ProgramState", mock.Anything, grafanaProgramName).Return(nil)
	f.grafana.On("IsReady", mock.Anything).Return(errors.New("connection refused"))

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
	f.supervisord.On("ProgramState", mock.Anything, grafanaProgramName).Return(nil)
	f.grafana.On("IsReady", mock.Anything).Return(errors.New("connection refused"))

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

// TestProvisionerToleratesConcurrentCallers is a regression test for a race found in review. Only
// Run reconciles in production now, but setup() asks for the startup reconcile from a retry
// goroutine of its own, and reconcile has to stay safe for more than one caller: they share both
// the counters and the files. Run this with -race.
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
	f.supervisord.On("ProgramState", mock.Anything, grafanaProgramName).Return(nil).Maybe()
	f.grafana.On("IsReady", mock.Anything).Return(errors.New("connection refused")).Maybe()

	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			f.provisioner.ProvisionAtStartup()
			f.provisioner.reconcile(context.Background(), triggerStartup)
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
	f.supervisord.On("ProgramState", mock.Anything, grafanaProgramName).Return(nil).Once()
	f.grafana.On("IsReady", mock.Anything).Return(errors.New("connection refused")).Once()

	f.provisioner.reconcile(context.Background(), triggerStartup)
	first := f.fileContent(t)

	// The second pass must not consult supervisord or Grafana at all: steady state is the common
	// case, and a reconcile that touched Grafana every five minutes would be a liability.
	f.provisioner.reconcile(context.Background(), triggerTick)
	assert.Equal(t, first, f.fileContent(t))
}

// TestProvisionerRestartsGrafanaWhenRequested is the leader's side of a change at runtime: once a
// node is leader, a tick that changes the content restarts its Grafana.
func TestProvisionerRestartsGrafanaWhenRequested(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)
	f.expectSettings(1, true)

	f.leader.On("IsLeader").Return(true)
	f.supervisord.On("ProgramState", mock.Anything, grafanaProgramName).Return(new(true))
	f.supervisord.On("RestartSupervisedService", mock.Anything, grafanaProgramName).Return(nil)
	f.grafana.On("IsReady", mock.Anything).Return(nil)

	f.provisioner.reconcile(context.Background(), triggerTick)
}

// TestProvisionerTickWaitsForTheLeader is the guard against a settings change bouncing Grafana on
// every node at once, when one restart already applies it for the whole cluster.
func TestProvisionerTickWaitsForTheLeader(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)
	f.expectSettings(3, true)

	f.supervisord.On("ProgramState", mock.Anything, grafanaProgramName).Return(new(true))
	f.leader.On("IsLeader").Return(false)

	for range 3 {
		f.provisioner.reconcile(context.Background(), triggerTick)
	}
	f.supervisord.AssertNotCalled(t, "RestartSupervisedService", mock.Anything, grafanaProgramName)
}

// TestProvisionerStartupRestartsWithoutWaitingForTheLeader covers the boot of an HA node, where the
// leader gate cannot be the answer: leadership is only established after the startup reconcile, so
// every node deferred and nobody applied, and no other node can see that this one's Grafana started
// on the previous file. The new rules then only reached the cluster when some Grafana happened to
// restart.
func TestProvisionerStartupRestartsWithoutWaitingForTheLeader(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)
	f.expectSettings(1, true)

	// Deliberately no leader stub: a call would fail the test.
	f.supervisord.On("ProgramState", mock.Anything, grafanaProgramName).Return(new(true))
	f.supervisord.On("RestartSupervisedService", mock.Anything, grafanaProgramName).Return(nil).Once()
	// Once before the boot restart, to let Grafana finish starting, and once after it.
	f.grafana.On("IsReady", mock.Anything).Return(nil).Times(2)

	f.provisioner.reconcile(context.Background(), triggerStartup)

	assert.False(t, f.provisioner.startupApplyOwed, "a restart settles the boot's debt")
	assert.Zero(t, errorCount(t, f.provisioner, stageApply))
	assert.Equal(t, stateWritten, bundleState(t, f.provisioner, haBundleID))
}

// TestProvisionerStartupDebtSurvivesARenderFailure is the case the boot exemption exists for. A
// render that fails at boot leaves Grafana running on the previous file; the retry that finishes
// the recovery is the one that writes, and it must restart without a leader just as the startup
// reconcile would have, or the recovery ends with a file nobody has read.
func TestProvisionerStartupDebtSurvivesARenderFailure(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)
	for range 2 {
		f.dbMock.ExpectQuery("SELECT settings FROM settings").
			WillReturnRows(sqlmock.NewRows([]string{"settings"}).AddRow(settingsJSON(true)))
	}

	// Grafana's database is unreachable at boot and back for the retry.
	dsDB, dsMock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = dsDB.Close() })
	dsMock.ExpectQuery("SELECT uid FROM data_source").WillReturnError(errors.New("connection refused"))
	dsMock.ExpectQuery("SELECT uid FROM data_source").
		WillReturnRows(sqlmock.NewRows([]string{"uid"}).AddRow(deriveMetricsDatasourceUID()))
	dsMock.ExpectQuery("FROM alert_rule").WillReturnRows(sqlmock.NewRows([]string{"uid", "provenance"}))
	f.provisioner.grafanaDB = newGrafanaReader("dsn-not-used", logrus.WithField("test", t.Name()))
	f.provisioner.grafanaDB.db = dsDB

	f.supervisord.On("ProgramState", mock.Anything, grafanaProgramName).Return(new(true))
	f.supervisord.On("RestartSupervisedService", mock.Anything, grafanaProgramName).Return(nil).Once()
	// Once before the boot restart, to let Grafana finish starting, and once after it.
	f.grafana.On("IsReady", mock.Anything).Return(nil).Times(2)

	f.provisioner.reconcile(context.Background(), triggerStartup)
	require.True(t, f.provisioner.startupApplyOwed, "a boot that wrote nothing still owes its restart")
	require.Positive(t, f.provisioner.retryBackoff, "the datasource failure arms a retry")

	f.provisioner.reconcile(context.Background(), triggerRetry)

	assert.False(t, f.provisioner.startupApplyOwed)
	assert.Zero(t, f.provisioner.retryBackoff)
	assert.Zero(t, errorCount(t, f.provisioner, stageApply))
	assert.Equal(t, stateWritten, bundleState(t, f.provisioner, haBundleID))
}

// TestProvisionerBootExemptionEndsWithTheBoot guards the tick side: once the boot's debt is settled,
// a change at runtime on a follower is left to the leader as before, because the leader sees the
// same change and applies it for the cluster.
func TestProvisionerBootExemptionEndsWithTheBoot(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)
	f.expectSettings(1, true)
	f.supervisord.On("ProgramState", mock.Anything, grafanaProgramName).Return(new(true))
	f.supervisord.On("RestartSupervisedService", mock.Anything, grafanaProgramName).Return(nil).Once()
	// Once before the boot restart, to let Grafana finish starting, and once after it.
	f.grafana.On("IsReady", mock.Anything).Return(nil).Times(2)

	f.provisioner.reconcile(context.Background(), triggerStartup)
	require.False(t, f.provisioner.startupApplyOwed)

	// Percona Alerting switched off changes the content, on a node that is not the leader.
	f.expectSettings(1, false)
	f.leader.On("IsLeader").Return(false).Once()

	f.provisioner.reconcile(context.Background(), triggerTick)

	f.supervisord.AssertNumberOfCalls(t, "RestartSupervisedService", 1)
	assert.Zero(t, errorCount(t, f.provisioner, stageApply), "a deferral is still not a failure")
	assert.False(t, f.provisioner.applyPending)
}

// TestProvisionerRollsBackAFailedRestart proves the promise the whole design rests on: PMM must
// never leave Grafana unable to start.
func TestProvisionerRollsBackAFailedRestart(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)

	// A first pass leaves a known good file in place.
	f.expectSettings(1, true)
	f.supervisord.On("ProgramState", mock.Anything, grafanaProgramName).Return(nil).Once()
	f.grafana.On("IsReady", mock.Anything).Return(errors.New("connection refused")).Once()
	f.provisioner.reconcile(context.Background(), triggerStartup)
	good := f.fileContent(t)

	// Then a real change - Percona Alerting switched off empties the file - which Grafana refuses
	// to come back from. Only a change reaches the apply step: an identical render is a no-op.
	f.expectSettings(1, false)
	f.leader.On("IsLeader").Return(true)
	f.supervisord.On("ProgramState", mock.Anything, grafanaProgramName).Return(new(true))
	f.supervisord.On("RestartSupervisedService", mock.Anything, grafanaProgramName).Return(nil)
	f.grafana.On("IsReady", mock.Anything).Return(errors.New("connection refused"))

	f.provisioner.reconcile(context.Background(), triggerTick)

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

// TestProvisionerAtStartupDoesNotWait is the review finding on setup(): the first call runs before
// PMM's API servers start, and it used to restart Grafana and wait up to readyTimeout for it right
// there. Asking must touch nothing - every mock here is strict, so any call to Grafana, supervisord
// or the database fails the test - and being asked again by every setup() retry must not panic on
// a second close.
func TestProvisionerAtStartupDoesNotWait(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)

	for range 3 {
		f.provisioner.ProvisionAtStartup()
	}

	_, err := os.Stat(filepath.Join(f.dir, provisioningFileName))
	require.ErrorIs(t, err, fs.ErrNotExist, "nothing may be written until Run does the work")
	require.NoError(t, f.dbMock.ExpectationsWereMet())
}

// TestProvisionerRunReconcilesOnceForStartup checks the other half: Run does the startup reconcile it
// was asked for, straight away rather than at the first tick, and once however often setup() asked.
// A second startup reconcile would find no settings row queued and count a render error.
func TestProvisionerRunReconcilesOnceForStartup(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)
	f.expectSettings(1, true)
	f.supervisord.On("ProgramState", mock.Anything, grafanaProgramName).Return(nil).Once()
	f.grafana.On("IsReady", mock.Anything).Return(errors.New("connection refused")).Once()

	for range 3 {
		f.provisioner.ProvisionAtStartup()
	}

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		f.provisioner.Run(ctx)
		close(done)
	}()

	assert.Eventually(t, func() bool {
		_, err := os.Stat(filepath.Join(f.dir, provisioningFileName))
		return err == nil
	}, 5*time.Second, 10*time.Millisecond, "the startup reconcile should not wait for a tick")

	// Give a second startup reconcile, if there were one, time to run before stopping Run.
	time.Sleep(100 * time.Millisecond)
	cancel()
	<-done

	assert.Zero(t, errorCount(t, f.provisioner, stageRender), "the startup reconcile must run exactly once")
	assert.Zero(t, f.provisioner.retryBackoff)
	require.NoError(t, f.dbMock.ExpectationsWereMet())
}

// TestProvisionerStartsGrafanaWhenSupervisordWillNot covers the case that used to leave a server
// with its interface down: Grafana died on a file PMM has since repaired, supervisord gave up
// retrying, and nothing was left to start it.
func TestProvisionerStartsGrafanaWhenSupervisordWillNot(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)
	f.expectSettings(1, true)

	// false is what parseStatus returns for FATAL and STOPPED - "will not be restarted".
	f.supervisord.On("ProgramState", mock.Anything, grafanaProgramName).Return(new(false))
	f.supervisord.On("StartSupervisedService", grafanaProgramName).Return(nil)
	f.grafana.On("IsReady", mock.Anything).Return(nil)

	f.provisioner.reconcile(context.Background(), triggerStartup)

	// Deliberately no leader stub: starting a dead Grafana is a repair, not a rollout action, so it
	// is not leader-gated. A mockLeaderService call would fail the test.
	f.supervisord.AssertCalled(t, "StartSupervisedService", grafanaProgramName)
	f.supervisord.AssertNotCalled(t, "RestartSupervisedService", mock.Anything, grafanaProgramName)
}

// TestProvisionerLeavesGrafanaAloneWhenStateIsUnknown pins supervisord's documented contract: a nil
// status means "leave it alone". The usual cause is a first boot where Grafana is not configured
// yet, and supervisord starts it moments later with this file already in place.
func TestProvisionerLeavesGrafanaAloneWhenStateIsUnknown(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)
	f.expectSettings(1, true)
	f.supervisord.On("ProgramState", mock.Anything, grafanaProgramName).Return(nil)
	f.grafana.On("IsReady", mock.Anything).Return(errors.New("connection refused"))

	f.provisioner.reconcile(context.Background(), triggerStartup)

	require.NotEmpty(t, f.fileContent(t), "the file is still written")
	f.supervisord.AssertNotCalled(t, "StartSupervisedService", grafanaProgramName)
	f.supervisord.AssertNotCalled(t, "RestartSupervisedService", mock.Anything, grafanaProgramName)
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
	f.supervisord.On("ProgramState", mock.Anything, grafanaProgramName).Return(nil)
	f.grafana.On("IsReady", mock.Anything).Return(errors.New("connection refused"))

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
	f.supervisord.On("ProgramState", mock.Anything, grafanaProgramName).Return(nil)
	f.grafana.On("IsReady", mock.Anything).Return(errors.New("connection refused"))

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

	f.supervisord.On("ProgramState", mock.Anything, grafanaProgramName).Return(new(true))
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

	// Down for the first apply, then a status that cannot be determined after the failed start,
	// which is what a command that would not run looks like, then down again for the retry.
	f.expectProgramStates(new(false), nil, new(false))
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

// TestProvisionerApplyRetryBacksOff checks that a failing apply is retried on the same growing
// backoff as a failing render. The backoff used to be cleared right after the write, before the
// apply, so a start that failed straight away was retried every two seconds for good.
func TestProvisionerApplyRetryBacksOff(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)
	f.expectSettings(3, true)

	f.expectProgramStates(new(false), nil, new(false), nil, new(false))
	f.supervisord.On("StartSupervisedService", grafanaProgramName).
		Return(errors.New("boom")).Times(2)
	f.supervisord.On("StartSupervisedService", grafanaProgramName).Return(nil).Once()
	f.grafana.On("IsReady", mock.Anything).Return(nil).Once()

	f.provisioner.reconcile(context.Background(), triggerStartup)
	assert.Equal(t, datasourceRetryInitial, f.provisioner.retryBackoff)

	f.provisioner.reconcile(context.Background(), triggerRetry)
	assert.Equal(t, datasourceRetryInitial*datasourceRetryFactor, f.provisioner.retryBackoff,
		"a second failure must grow the backoff rather than start over")

	f.provisioner.reconcile(context.Background(), triggerRetry)
	assert.Zero(t, f.provisioner.retryBackoff, "a successful apply clears the backoff")
	assert.False(t, f.provisioner.applyPending)
	assert.InDelta(t, 2, errorCount(t, f.provisioner, stageApply), 0)
	assert.Equal(t, stateWritten, bundleState(t, f.provisioner, haBundleID))
}

// TestProvisionerStillDoesNothingWhenNothingChanged guards the other side of that condition: the
// steady-state no-op must survive. A server that reconciles every five minutes forever must not
// touch Grafana once its file is correct.
func TestProvisionerStillDoesNothingWhenNothingChanged(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)
	f.expectSettings(3, true)

	f.supervisord.On("ProgramState", mock.Anything, grafanaProgramName).Return(new(true)).Once()
	f.leader.On("IsLeader").Return(true).Once()
	f.supervisord.On("RestartSupervisedService", mock.Anything, grafanaProgramName).Return(nil).Once()
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

// TestProvisionerAppliesToAGrafanaThatServesWhileItsStateIsUnknown is the guard against an apply
// discharged by a status command that would not run. ProgramState reports the same nil for output
// it cannot parse as it does for a program supervisord has not been told about, and taking that as
// "leave it alone" settles the apply for good: the file is already on disk, so every later
// reconcile finds it unchanged and never restarts anything. Grafana serving is what tells the two
// apart, because a Grafana that answers is serving rules it read before this file changed.
func TestProvisionerAppliesToAGrafanaThatServesWhileItsStateIsUnknown(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)
	f.expectSettings(1, true)

	f.supervisord.On("ProgramState", mock.Anything, grafanaProgramName).Return(nil)
	f.grafana.On("IsReady", mock.Anything).Return(nil)
	f.supervisord.On("RestartSupervisedService", mock.Anything, grafanaProgramName).Return(nil)

	f.provisioner.reconcile(context.Background(), triggerStartup)

	f.supervisord.AssertNumberOfCalls(t, "RestartSupervisedService", 1)
	assert.False(t, f.provisioner.applyPending, "the apply happened, so nothing is owed")
	assert.False(t, f.provisioner.startupApplyOwed)
	assert.Equal(t, stateWritten, bundleState(t, f.provisioner, haBundleID))
}

// TestProvisionerStopsOfferingARevisionGrafanaRejects is the guard against the restart loop found
// in review. A failed apply leaves applyPending set, which defeats the "nothing changed" early
// return, so without a budget the same content is offered for as long as it keeps being rendered:
// another restart, another wait for a Grafana that never answers, every retry, forever.
func TestProvisionerStopsOfferingARevisionGrafanaRejects(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)

	// A first pass leaves a known good file in place.
	f.expectSettings(1, true)
	f.supervisord.On("ProgramState", mock.Anything, grafanaProgramName).Return(nil).Once()
	f.grafana.On("IsReady", mock.Anything).Return(errors.New("connection refused")).Once()
	f.provisioner.reconcile(context.Background(), triggerStartup)
	good := f.fileContent(t)

	// Then a real change - Percona Alerting switched off empties the file - offered far more times
	// than the budget allows.
	const attempts = 5
	f.expectSettings(attempts, false)
	f.leader.On("IsLeader").Return(true)
	f.supervisord.On("ProgramState", mock.Anything, grafanaProgramName).Return(new(true))
	f.supervisord.On("RestartSupervisedService", mock.Anything, grafanaProgramName).Return(nil)
	f.grafana.On("IsReady", mock.Anything).Return(errors.New("connection refused"))

	for range attempts {
		f.provisioner.reconcile(context.Background(), triggerRetry)
	}

	f.supervisord.AssertNumberOfCalls(t, "RestartSupervisedService", maxApplyAttemptsPerRevision)
	assert.Equal(t, good, f.fileContent(t),
		"the file Grafana last accepted must be the one on disk, so the next start has something to read")
	assert.Zero(t, f.provisioner.retryBackoff, "nothing is owed once a revision is given up on")
	assert.True(t, f.provisioner.applyPending,
		"the rules Grafana holds are still not the rendered ones, and the metric has to say so")
	assert.InDelta(t, maxApplyAttemptsPerRevision, errorCount(t, f.provisioner, stageApply), 0,
		"each attempt is counted once, and the passes that skip the apply add nothing")
}

// TestProvisionerOffersNewContentAfterGivingUpOnARevision is the other half of the budget: it is
// spent per revision, not for good. Anything that renders differently is offered on its own terms.
func TestProvisionerOffersNewContentAfterGivingUpOnARevision(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)

	f.expectSettings(1, true)
	f.supervisord.On("ProgramState", mock.Anything, grafanaProgramName).Return(nil).Once()
	f.grafana.On("IsReady", mock.Anything).Return(errors.New("connection refused")).Once()
	f.provisioner.reconcile(context.Background(), triggerStartup)

	f.expectSettings(maxApplyAttemptsPerRevision, false)
	f.leader.On("IsLeader").Return(true).Times(maxApplyAttemptsPerRevision)
	f.supervisord.On("ProgramState", mock.Anything, grafanaProgramName).Return(new(true))
	f.supervisord.On("RestartSupervisedService", mock.Anything, grafanaProgramName).Return(nil)
	f.grafana.On("IsReady", mock.Anything).Return(errors.New("connection refused"))
	for range maxApplyAttemptsPerRevision {
		f.provisioner.reconcile(context.Background(), triggerRetry)
	}
	require.NotEmpty(t, f.provisioner.rejectedHash, "the revision must have been given up on first")

	// Switching Percona Alerting back on renders something else. This node is not the leader, so
	// the proof that the content is being offered again is that it reaches the leader gate at all.
	f.expectSettings(1, true)
	f.leader.On("IsLeader").Return(false).Once()
	f.provisioner.reconcile(context.Background(), triggerTick)

	assert.Empty(t, f.provisioner.rejectedHash, "one revision is never held against another")
	assert.Contains(t, f.fileContent(t), "PMM High Availability")
	f.supervisord.AssertNumberOfCalls(t, "RestartSupervisedService", maxApplyAttemptsPerRevision)
}

// TestProvisionerStartsAGrafanaLeftDownByARevisionItGaveUpOn covers what is still owed after giving
// up. The file has been rolled back to one Grafana accepted, but Grafana itself may have been left
// dead by the revision that failed, and nothing else is coming for it. Giving up also cancels the
// fast retry, so this has to happen in the same pass: the next one is a whole tick away.
func TestProvisionerStartsAGrafanaLeftDownByARevisionItGaveUpOn(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)

	f.expectSettings(1, true)
	f.supervisord.On("ProgramState", mock.Anything, grafanaProgramName).Return(nil).Once()
	f.grafana.On("IsReady", mock.Anything).Return(errors.New("connection refused")).Once()
	f.provisioner.reconcile(context.Background(), triggerStartup)
	good := f.fileContent(t)

	f.expectSettings(maxApplyAttemptsPerRevision, false)
	f.leader.On("IsLeader").Return(true)
	f.supervisord.On("ProgramState", mock.Anything, grafanaProgramName).
		Return(new(true)).Times(maxApplyAttemptsPerRevision)
	f.supervisord.On("RestartSupervisedService", mock.Anything, grafanaProgramName).Return(nil)
	f.grafana.On("IsReady", mock.Anything).Return(errors.New("connection refused"))

	// After the last attempt Grafana is FATAL, which supervisord documents as "will not be
	// restarted".
	f.supervisord.On("ProgramState", mock.Anything, grafanaProgramName).Return(new(false))
	f.supervisord.On("StartSupervisedService", grafanaProgramName).Return(nil)

	for range maxApplyAttemptsPerRevision {
		f.provisioner.reconcile(context.Background(), triggerRetry)
	}

	f.supervisord.AssertNumberOfCalls(t, "StartSupervisedService", 1)
	assert.Zero(t, f.provisioner.retryBackoff, "the revision was given up on, so no retry is coming")
	assert.Equal(t, good, f.fileContent(t), "it must be started on the file it last accepted, not the one it refused")
}

// TestProvisionerChargesOnlyGrafanaFailuresToTheRevision keeps the budget aimed at what it is for.
// A supervisord command that would not run says nothing about the content and leaves Grafana where
// it was, so it must be retried indefinitely rather than counted against the rules. It is told
// apart by the status check after it: a supervisorctl that is not there cannot report a state
// either.
func TestProvisionerChargesOnlyGrafanaFailuresToTheRevision(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)
	f.expectSettings(3, true)

	for range 3 {
		f.expectProgramStates(new(false), nil)
	}
	f.supervisord.On("StartSupervisedService", grafanaProgramName).Return(errors.New("supervisorctl is not there"))

	for range 3 {
		f.provisioner.reconcile(context.Background(), triggerRetry)
	}

	f.supervisord.AssertNumberOfCalls(t, "StartSupervisedService", 3)
	assert.Zero(t, f.provisioner.rejectedApplies)
	assert.Equal(t, 4*datasourceRetryInitial, f.provisioner.retryBackoff,
		"the backoff must keep growing, because this is still worth retrying")
}

// TestProvisionerRollsBackAStartGrafanaDoesNotSurvive is the rollback on the other apply path. A
// Grafana supervisord has given up on is started by PMM, and content it cannot start on has to be
// taken back off disk just as a failed restart's is - otherwise it can never come up again.
func TestProvisionerRollsBackAStartGrafanaDoesNotSurvive(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)

	f.expectSettings(1, true)
	f.supervisord.On("ProgramState", mock.Anything, grafanaProgramName).Return(nil).Once()
	f.grafana.On("IsReady", mock.Anything).Return(errors.New("connection refused")).Once()
	f.provisioner.reconcile(context.Background(), triggerStartup)
	good := f.fileContent(t)

	f.expectSettings(1, false)
	f.supervisord.On("ProgramState", mock.Anything, grafanaProgramName).Return(new(false))
	f.supervisord.On("StartSupervisedService", grafanaProgramName).Return(nil)
	f.grafana.On("IsReady", mock.Anything).Return(errors.New("connection refused"))

	f.provisioner.reconcile(context.Background(), triggerTick)

	assert.Equal(t, good, f.fileContent(t), "the file Grafana last started from must be restored")
	assert.Equal(t, 1, f.provisioner.rejectedApplies, "a Grafana that did not come back counts against the content")
}

// TestProvisionerChargesARestartGrafanaDiesDuring is the review finding on the restart path. Grafana
// runs with startsecs = 1, so content it cannot start from makes supervisorctl restart itself exit
// non-zero ("abnormal termination"), before any readiness wait. That failure used to be taken for a
// command that would not run: the bad file stayed on disk and the revision was never charged.
func TestProvisionerChargesARestartGrafanaDiesDuring(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)

	f.expectSettings(1, true)
	f.expectProgramStates(nil)
	f.grafana.On("IsReady", mock.Anything).Return(errors.New("connection refused")).Once()
	f.provisioner.reconcile(context.Background(), triggerStartup)
	good := f.fileContent(t)

	f.expectSettings(1, false)
	f.leader.On("IsLeader").Return(true)
	// Running when the apply looks, FATAL once the restart has failed.
	f.expectProgramStates(new(true), new(false))
	f.supervisord.On("RestartSupervisedService", mock.Anything, grafanaProgramName).
		Return(errors.New("grafana: ERROR (abnormal termination)")).Once()

	f.provisioner.reconcile(context.Background(), triggerTick)

	assert.Equal(t, good, f.fileContent(t), "the file Grafana last started from must be restored")
	assert.Equal(t, 1, f.provisioner.rejectedApplies, "a Grafana that died on the content counts against it")
	assert.True(t, f.provisioner.applyPending)
}

// TestProvisionerGivesUpOnAStartGrafanaDiesDuring is the review finding on the start path, where the
// uncharged failure had no way out: the rollback left Grafana FATAL, the next retry wrote the same
// content again and started Grafana on it, and the budget that ends that never ran out. Charged, it
// runs out after maxApplyAttemptsPerRevision, and Grafana is brought back on the restored file.
func TestProvisionerGivesUpOnAStartGrafanaDiesDuring(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)

	f.expectSettings(1, true)
	f.expectProgramStates(nil)
	f.grafana.On("IsReady", mock.Anything).Return(errors.New("connection refused")).Once()
	f.provisioner.reconcile(context.Background(), triggerStartup)
	good := f.fileContent(t)

	f.expectSettings(maxApplyAttemptsPerRevision+1, false)
	// FATAL for each attempt and after each failed start, and for the recovery that follows giving
	// up. Running from then on.
	for range maxApplyAttemptsPerRevision {
		f.expectProgramStates(new(false), new(false))
	}
	f.expectProgramStates(new(false))
	f.supervisord.On("ProgramState", mock.Anything, grafanaProgramName).Return(new(true))
	f.supervisord.On("StartSupervisedService", grafanaProgramName).
		Return(errors.New("grafana: ERROR (abnormal termination)")).Times(maxApplyAttemptsPerRevision)
	f.supervisord.On("StartSupervisedService", grafanaProgramName).Return(nil).Once()
	f.grafana.On("IsReady", mock.Anything).Return(nil).Once()

	for range maxApplyAttemptsPerRevision + 1 {
		f.provisioner.reconcile(context.Background(), triggerRetry)
	}

	f.supervisord.AssertNumberOfCalls(t, "StartSupervisedService", maxApplyAttemptsPerRevision+1)
	assert.Equal(t, maxApplyAttemptsPerRevision, f.provisioner.rejectedApplies)
	assert.Zero(t, f.provisioner.retryBackoff, "the revision was given up on, so no retry is coming")
	assert.Equal(t, good, f.fileContent(t), "Grafana must be started on the file it last accepted")
}

// TestProvisionerWaitsForAGrafanaSupervisordIsStillRetrying covers the failed command that is not
// the end of it. The first abnormal termination is what supervisorctl reports, but supervisord
// keeps retrying up to startretries, and a Grafana that then comes up has applied the content.
func TestProvisionerWaitsForAGrafanaSupervisordIsStillRetrying(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)
	f.expectSettings(1, true)

	f.leader.On("IsLeader").Return(true)
	// Running when the apply looks, backing off (reported as running) after the failed restart.
	f.expectProgramStates(new(true), new(true))
	f.supervisord.On("RestartSupervisedService", mock.Anything, grafanaProgramName).
		Return(errors.New("grafana: ERROR (abnormal termination)")).Once()
	f.grafana.On("IsReady", mock.Anything).Return(nil).Once()

	f.provisioner.reconcile(context.Background(), triggerTick)

	assert.False(t, f.provisioner.applyPending, "Grafana came back on the new content")
	assert.Zero(t, f.provisioner.rejectedApplies)
	assert.Zero(t, errorCount(t, f.provisioner, stageApply))
}

// TestProvisionerBootRestartLetsGrafanaFinishStarting is the review finding on the first boot. From
// its first second Grafana is reported running by supervisord, while a cold start can still be in
// database migrations, and the boot restart used to cut that short. It now waits for Grafana to
// answer first, and only then restarts it.
func TestProvisionerBootRestartLetsGrafanaFinishStarting(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)
	f.expectSettings(1, true)

	var calls []string
	f.supervisord.On("ProgramState", mock.Anything, grafanaProgramName).Return(new(true))
	f.grafana.On("IsReady", mock.Anything).Return(errors.New("connection refused")).Times(2).
		Run(func(mock.Arguments) { calls = append(calls, "not ready") })
	f.grafana.On("IsReady", mock.Anything).Return(nil).
		Run(func(mock.Arguments) { calls = append(calls, "ready") })
	f.supervisord.On("RestartSupervisedService", mock.Anything, grafanaProgramName).Return(nil).Once().
		Run(func(mock.Arguments) { calls = append(calls, "restart") })

	f.provisioner.reconcile(context.Background(), triggerStartup)

	assert.Equal(t, []string{"not ready", "not ready", "ready", "restart", "ready"}, calls)
	assert.False(t, f.provisioner.startupApplyOwed)
	assert.False(t, f.provisioner.applyPending)
}

// TestProvisionerBootLeavesAGrafanaStillStartingAlone covers a Grafana slower than readyTimeout. It
// is not restarted and nothing is charged, because a slow start says nothing about the content, and
// charging it would spend the budget on every slow boot and leave the server without its rules. The
// boot restart stays owed and is retried on the backoff.
func TestProvisionerBootLeavesAGrafanaStillStartingAlone(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)
	f.expectSettings(2, true)

	f.supervisord.On("ProgramState", mock.Anything, grafanaProgramName).Return(new(true))
	f.grafana.On("IsReady", mock.Anything).Return(errors.New("connection refused"))

	for _, trigger := range []provisioningTrigger{triggerStartup, triggerRetry} {
		f.provisioner.reconcile(context.Background(), trigger)
	}

	f.supervisord.AssertNotCalled(t, "RestartSupervisedService", mock.Anything, grafanaProgramName)
	assert.Zero(t, f.provisioner.rejectedApplies, "a slow start must not count against the content")
	assert.True(t, f.provisioner.applyPending)
	assert.True(t, f.provisioner.startupApplyOwed, "the boot restart is still owed")
	assert.Equal(t, datasourceRetryInitial*datasourceRetryFactor, f.provisioner.retryBackoff)
}

// TestProvisionerAppliesADeferralAfterBecomingLeader is the review finding on leadership moving
// between a deferral and the leader's apply. Node B writes as a follower and defers, then becomes
// the leader before the old leader ticked. The old leader, now a follower, defers too, so unless B
// remembers its deferral nobody applies and the shared database keeps the old rules.
func TestProvisionerAppliesADeferralAfterBecomingLeader(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)
	f.expectSettings(2, true)

	f.supervisord.On("ProgramState", mock.Anything, grafanaProgramName).Return(new(true))
	f.leader.On("IsLeader").Return(false).Once()
	f.provisioner.reconcile(context.Background(), triggerTick)
	require.False(t, f.provisioner.deferredAt.IsZero(), "the deferral must be remembered")
	assert.Equal(t, stateWritten, bundleState(t, f.provisioner, haBundleID), "a deferring follower is not pending")

	// Leadership moves to this node. The file is unchanged, so only the deferral can make it apply.
	f.leader.On("IsLeader").Return(true)
	f.supervisord.On("RestartSupervisedService", mock.Anything, grafanaProgramName).Return(nil).Once()
	f.grafana.On("IsReady", mock.Anything).Return(nil).Once()
	f.provisioner.reconcile(context.Background(), triggerTick)

	f.supervisord.AssertNumberOfCalls(t, "RestartSupervisedService", 1)
	assert.True(t, f.provisioner.deferredAt.IsZero(), "applying settles the deferral")
	assert.False(t, f.provisioner.applyPending)
}

// TestProvisionerForgetsADeferralOnceTheLeaderHadItsTurn is the bound on that. Past deferralWindow
// the old leader has ticked and applied, so a node that becomes leader later must not restart its
// Grafana for a change the cluster already has.
func TestProvisionerForgetsADeferralOnceTheLeaderHadItsTurn(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)
	f.expectSettings(2, true)

	f.supervisord.On("ProgramState", mock.Anything, grafanaProgramName).Return(new(true))
	f.leader.On("IsLeader").Return(false).Once()
	f.provisioner.reconcile(context.Background(), triggerTick)
	require.False(t, f.provisioner.deferredAt.IsZero())

	f.provisioner.deferredAt = time.Now().Add(-deferralWindow - time.Second)
	f.provisioner.reconcile(context.Background(), triggerTick)

	f.supervisord.AssertNotCalled(t, "RestartSupervisedService", mock.Anything, grafanaProgramName)
	f.leader.AssertNumberOfCalls(t, "IsLeader", 1)
	assert.True(t, f.provisioner.deferredAt.IsZero(), "an expired deferral is dropped")
}

// TestProvisionerFollowerKeepsLeavingADeferralToTheLeader guards the steady state on a follower: a
// remembered deferral must not make a node that is still a follower restart anything.
func TestProvisionerFollowerKeepsLeavingADeferralToTheLeader(t *testing.T) {
	t.Parallel()

	f := newProvisionerFixture(t, true)
	f.expectSettings(3, true)

	f.supervisord.On("ProgramState", mock.Anything, grafanaProgramName).Return(new(true))
	f.leader.On("IsLeader").Return(false)

	for range 3 {
		f.provisioner.reconcile(context.Background(), triggerTick)
	}

	f.supervisord.AssertNotCalled(t, "RestartSupervisedService", mock.Anything, grafanaProgramName)
	assert.False(t, f.provisioner.deferredAt.IsZero())
	assert.Equal(t, stateWritten, bundleState(t, f.provisioner, haBundleID))
}
