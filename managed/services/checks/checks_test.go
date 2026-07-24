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

package checks

import (
	"bytes"
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	_ "github.com/ClickHouse/clickhouse-go/v2"
	metrics "github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/pi/check"
	"github.com/percona/pmm/managed/pi/common"
	"github.com/percona/pmm/managed/services"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/version"
)

const (
	testChecksFile = "../../testdata/checks/good_check_pg.yml"
)

var (
	vmClient     v1.API
	clickhouseDB *sql.DB
)

// loadTestCheck parses the good-check test fixture into a check.Check.
func loadTestCheck(t *testing.T) check.Check {
	t.Helper()

	b, err := os.ReadFile(testChecksFile)
	require.NoError(t, err)

	checks, err := check.ParseChecks(bytes.NewReader(b), &check.ParseParams{
		DisallowUnknownFields: true,
		DisallowInvalidChecks: true,
	})
	require.NoError(t, err)
	require.Len(t, checks, 1)

	return checks[0]
}

// seedUserCheck stores the good-check test fixture as a user-authored check in the DB.
func seedUserCheck(t *testing.T, db *reform.DB) {
	t.Helper()

	c := loadTestCheck(t)
	m, err := userCheckToModel(c)
	require.NoError(t, err)
	_, err = models.CreateAdvisorCheck(db.Querier, m)
	require.NoError(t, err)
}

func TestLoadBuiltinAdvisors(t *testing.T) {
	setupClients(t)
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)

	s := New(db, nil, vmClient, clickhouseDB)

	t.Run("normal", func(t *testing.T) {
		checks, err := s.GetAdvisors()
		require.NoError(t, err)
		assert.Empty(t, checks)
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer cancel()

		err = s.reconcileBuiltinChecks(ctx)
		require.NoError(t, err)

		s.UpdateAdvisorsList(ctx)

		checks, err = s.GetAdvisors()
		require.NoError(t, err)
		assert.NotEmpty(t, checks)
	})

	t.Run("advisors are loaded with telemetry disabled", func(t *testing.T) {
		_, err := models.UpdateSettings(db.Querier, &models.ChangeSettingsParams{
			EnableTelemetry: new(false),
		})
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer cancel()

		dChecks, err := s.loadBuiltinChecks(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, dChecks)

		checks, err := s.GetAdvisors()
		require.NoError(t, err)
		assert.NotEmpty(t, checks)
	})
}

func TestUpdateAdvisorsList(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)

	t.Run("collect custom checks", func(t *testing.T) {
		s := New(db, nil, vmClient, clickhouseDB)
		seedUserCheck(t, db)

		s.UpdateAdvisorsList(t.Context())

		advisors, err := s.GetAdvisors()
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(advisors), 1)

		// the user check carries a unique (category, subcategory), so it forms
		// its own advisor group loaded last.
		advisor := advisors[len(advisors)-1]
		require.Equal(t, "Development", advisor.Category)
		require.Equal(t, "Dev", advisor.Subcategory)
		require.Len(t, advisor.Checks, 1)

		checkNames := make([]string, 0, len(advisor.Checks))
		for _, c := range advisor.Checks {
			checkNames = append(checkNames, c.Name)
		}
		assert.ElementsMatch(t, []string{
			"good_check_pg",
		}, checkNames)
	})
}

func TestDisableChecks(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		sqlDB := testdb.Open(t, models.SkipFixtures, nil)
		t.Cleanup(func() {
			require.NoError(t, sqlDB.Close())
		})

		db := reform.NewDB(sqlDB, postgresql.Dialect, nil)

		s := New(db, nil, vmClient, clickhouseDB)
		seedUserCheck(t, db)

		s.UpdateAdvisorsList(t.Context())

		checks, err := s.GetChecks()
		require.NoError(t, err)
		assert.NotEmpty(t, checks)

		disChecks, err := s.GetDisabledChecks(t.Context())
		require.NoError(t, err)
		assert.Empty(t, disChecks)

		err = s.DisableChecks(t.Context(), []string{checks["good_check_pg"].Name})
		require.NoError(t, err)

		disChecks, err = s.GetDisabledChecks(t.Context())
		require.NoError(t, err)
		assert.Len(t, disChecks, 1)
	})

	t.Run("disable same check twice", func(t *testing.T) {
		sqlDB := testdb.Open(t, models.SkipFixtures, nil)
		t.Cleanup(func() {
			require.NoError(t, sqlDB.Close())
		})

		db := reform.NewDB(sqlDB, postgresql.Dialect, nil)

		s := New(db, nil, vmClient, clickhouseDB)
		seedUserCheck(t, db)

		s.UpdateAdvisorsList(t.Context())

		checks, err := s.GetChecks()
		require.NoError(t, err)
		assert.NotEmpty(t, checks)

		disChecks, err := s.GetDisabledChecks(t.Context())
		require.NoError(t, err)
		assert.Empty(t, disChecks)

		err = s.DisableChecks(t.Context(), []string{checks["good_check_pg"].Name})
		require.NoError(t, err)

		err = s.DisableChecks(t.Context(), []string{checks["good_check_pg"].Name})
		require.NoError(t, err)

		disChecks, err = s.GetDisabledChecks(t.Context())
		require.NoError(t, err)
		assert.Len(t, disChecks, 1)
	})

	t.Run("disable unknown check", func(t *testing.T) {
		sqlDB := testdb.Open(t, models.SkipFixtures, nil)
		t.Cleanup(func() {
			require.NoError(t, sqlDB.Close())
		})

		db := reform.NewDB(sqlDB, postgresql.Dialect, nil)

		s := New(db, nil, vmClient, clickhouseDB)
		seedUserCheck(t, db)

		s.UpdateAdvisorsList(t.Context())

		err := s.DisableChecks(t.Context(), []string{"unknown_check"})
		require.Error(t, err)

		disChecks, err := s.GetDisabledChecks(t.Context())
		require.NoError(t, err)
		assert.Empty(t, disChecks)
	})
}

func TestEnableChecks(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		sqlDB := testdb.Open(t, models.SkipFixtures, nil)
		t.Cleanup(func() {
			require.NoError(t, sqlDB.Close())
		})

		db := reform.NewDB(sqlDB, postgresql.Dialect, nil)

		s := New(db, nil, vmClient, clickhouseDB)
		seedUserCheck(t, db)

		s.UpdateAdvisorsList(t.Context())

		checks, err := s.GetChecks()
		require.NoError(t, err)
		assert.NotEmpty(t, checks, 1)

		originalLength := len(checks)
		err = s.DisableChecks(t.Context(), []string{checks["good_check_pg"].Name})
		require.NoError(t, err)

		disChecks, err := s.GetDisabledChecks(t.Context())
		require.NoError(t, err)
		assert.Equal(t, []string{checks["good_check_pg"].Name}, disChecks)

		enabledChecksCount := len(checks) - len(disChecks)
		assert.Equal(t, originalLength-1, enabledChecksCount)
	})
}

func TestChangeInterval(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		sqlDB := testdb.Open(t, models.SkipFixtures, nil)
		t.Cleanup(func() {
			require.NoError(t, sqlDB.Close())
		})

		db := reform.NewDB(sqlDB, postgresql.Dialect, nil)

		s := New(db, nil, vmClient, clickhouseDB)
		seedUserCheck(t, db)

		s.UpdateAdvisorsList(t.Context())

		checks, err := s.GetChecks()
		require.NoError(t, err)
		assert.NotEmpty(t, checks)

		// change all check intervals from standard to rare
		params := make(map[string]check.Interval)
		for _, c := range checks {
			params[c.Name] = check.Rare
		}
		err = s.ChangeInterval(t.Context(), params)
		require.NoError(t, err)

		updatedChecks, err := s.GetChecks()
		require.NoError(t, err)
		for _, c := range updatedChecks {
			assert.Equal(t, check.Rare, c.Interval)
		}

		t.Run("preserve intervals on restarts", func(t *testing.T) {
			err = s.runChecksGroup(t.Context(), "")
			require.NoError(t, err)

			checks, err := s.GetChecks()
			require.NoError(t, err)
			for _, c := range checks {
				assert.Equal(t, check.Rare, c.Interval)
			}
		})
	})
}

func TestChecksForServices(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)
	ctx := t.Context()

	s := New(db, nil, vmClient, clickhouseDB)
	seedUserCheck(t, db)
	s.UpdateAdvisorsList(ctx)

	node, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{
		NodeName: "test-node",
	})
	require.NoError(t, err)

	serviceIDs := make([]string, 0, 2)
	for _, name := range []string{"mysql1", "mysql2"} {
		svc, err := models.AddNewService(db.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
			ServiceName: name,
			NodeID:      node.NodeID,
			Address:     new("127.0.0.1"),
			Port:        new(uint16(3306)),
		})
		require.NoError(t, err)
		serviceIDs = append(serviceIDs, svc.ServiceID)
	}

	t.Run("disable and dedup", func(t *testing.T) {
		err := s.DisableChecksForServices(ctx, "good_check_pg", []string{serviceIDs[0]})
		require.NoError(t, err)

		// disabling again including an already-disabled service must not duplicate it
		err = s.DisableChecksForServices(ctx, "good_check_pg", serviceIDs)
		require.NoError(t, err)

		m, err := s.GetDisabledServicesForChecks(ctx)
		require.NoError(t, err)
		assert.ElementsMatch(t, serviceIDs, m["good_check_pg"])
	})

	t.Run("unknown check rejected", func(t *testing.T) {
		err := s.DisableChecksForServices(ctx, "no_such_check", []string{serviceIDs[0]})
		require.Error(t, err)
		assert.Equal(t, codes.NotFound, status.Code(err))
	})

	t.Run("unknown service rejected", func(t *testing.T) {
		err := s.DisableChecksForServices(ctx, "good_check_pg", []string{"no-such-service"})
		require.Error(t, err)
		assert.Equal(t, codes.NotFound, status.Code(err))
	})

	t.Run("globally disabled check rejects per-service changes but keeps them", func(t *testing.T) {
		err := s.DisableChecks(ctx, []string{"good_check_pg"})
		require.NoError(t, err)

		err = s.DisableChecksForServices(ctx, "good_check_pg", []string{serviceIDs[0]})
		require.Error(t, err)
		assert.Equal(t, codes.FailedPrecondition, status.Code(err))

		// existing per-service settings survive the global disable...
		m, err := s.GetDisabledServicesForChecks(ctx)
		require.NoError(t, err)
		assert.ElementsMatch(t, serviceIDs, m["good_check_pg"])

		// ...and still apply after the check is re-enabled globally
		err = s.EnableChecks(ctx, []string{"good_check_pg"})
		require.NoError(t, err)

		m, err = s.GetDisabledServicesForChecks(ctx)
		require.NoError(t, err)
		assert.ElementsMatch(t, serviceIDs, m["good_check_pg"])
	})

	t.Run("enable removes only given services", func(t *testing.T) {
		err := s.EnableChecksForServices(ctx, "good_check_pg", []string{serviceIDs[0]})
		require.NoError(t, err)

		m, err := s.GetDisabledServicesForChecks(ctx)
		require.NoError(t, err)
		assert.Equal(t, []string{serviceIDs[1]}, m["good_check_pg"])

		// IDs of unknown (e.g. already removed) services are accepted
		err = s.EnableChecksForServices(ctx, "good_check_pg", []string{"no-such-service", serviceIDs[1]})
		require.NoError(t, err)

		m, err = s.GetDisabledServicesForChecks(ctx)
		require.NoError(t, err)
		assert.Empty(t, m)
	})
}

func TestStartChecks(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)
	setupClients(t)

	t.Run("unknown interval", func(t *testing.T) {
		s := New(db, nil, vmClient, clickhouseDB)

		err := s.runChecksGroup(t.Context(), "unknown")
		require.EqualError(t, err, "unknown check interval: unknown")
	})

	t.Run("advisors enabled", func(t *testing.T) {
		s := New(db, nil, vmClient, clickhouseDB)

		seedUserCheck(t, db)
		s.UpdateAdvisorsList(t.Context())
		assert.NotEmpty(t, s.advisors)
		assert.NotEmpty(t, s.checks)

		err := s.runChecksGroup(t.Context(), "")
		require.NoError(t, err)
	})

	t.Run("advisors disabled", func(t *testing.T) {
		s := New(db, nil, vmClient, clickhouseDB)

		settings, err := models.GetSettings(db)
		require.NoError(t, err)

		settings.SaaS.Enabled = new(false)
		err = models.SaveSettings(db, settings)
		require.NoError(t, err)

		err = s.runChecksGroup(t.Context(), "")
		require.ErrorIs(t, err, services.ErrAdvisorsDisabled)
	})
}

func TestUserAdvisorChecks(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)
	ctx := t.Context()

	s := New(db, nil, vmClient, clickhouseDB)

	// author a valid user check from a known-good template
	c := loadTestCheck(t)
	c.Name = "custom_test_user_check_crud"

	err := s.CreateAdvisorCheck(ctx, c)
	require.NoError(t, err)

	checks, err := s.GetChecks()
	require.NoError(t, err)
	created, ok := checks[c.Name]
	require.True(t, ok)
	assert.True(t, created.UserDefined)
	assert.Equal(t, c.Summary, created.Summary)

	t.Run("duplicate name rejected", func(t *testing.T) {
		err := s.CreateAdvisorCheck(ctx, c)
		require.Error(t, err)
		assert.Equal(t, codes.AlreadyExists, status.Code(err))
	})

	t.Run("name without the reserved prefix rejected", func(t *testing.T) {
		unprefixed := c
		unprefixed.Name = "test_user_check_without_prefix"
		err := s.CreateAdvisorCheck(ctx, unprefixed)
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("update", func(t *testing.T) {
		updated := c
		updated.Summary = "updated summary"
		err := s.UpdateAdvisorCheck(ctx, updated)
		require.NoError(t, err)

		checks, err := s.GetChecks()
		require.NoError(t, err)
		require.Contains(t, checks, c.Name)
		assert.Equal(t, "updated summary", checks[c.Name].Summary)
	})

	t.Run("update unknown rejected", func(t *testing.T) {
		unknown := c
		unknown.Name = "no_such_check"
		err := s.UpdateAdvisorCheck(ctx, unknown)
		require.Error(t, err)
		assert.Equal(t, codes.NotFound, status.Code(err))
	})

	t.Run("delete", func(t *testing.T) {
		err := s.DeleteAdvisorCheck(ctx, c.Name)
		require.NoError(t, err)

		checks, err := s.GetChecks()
		require.NoError(t, err)
		assert.NotContains(t, checks, c.Name)
	})

	t.Run("delete unknown rejected", func(t *testing.T) {
		err := s.DeleteAdvisorCheck(ctx, "no_such_check")
		require.Error(t, err)
		assert.Equal(t, codes.NotFound, status.Code(err))
	})
}

func TestTestAdvisorCheck(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)
	ctx := t.Context()

	s := New(db, nil, vmClient, clickhouseDB)

	c := loadTestCheck(t)
	c.Name = "custom_test_dry_run"

	t.Run("invalid check rejected", func(t *testing.T) {
		invalid := c
		invalid.Script = ""

		res, output, err := s.TestAdvisorCheck(ctx, invalid, "svc-1")
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Nil(t, res)
		assert.Empty(t, output)
	})

	t.Run("unknown check family rejected", func(t *testing.T) {
		unknown := c
		unknown.Family = "unknown"

		res, output, err := s.TestAdvisorCheck(ctx, unknown, "svc-1")
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Nil(t, res)
		assert.Empty(t, output)
	})

	t.Run("unknown service rejected", func(t *testing.T) {
		res, output, err := s.TestAdvisorCheck(ctx, c, "no-such-service")
		require.Error(t, err)
		assert.Equal(t, codes.NotFound, status.Code(err))
		assert.Nil(t, res)
		assert.Empty(t, output)
	})

	// keep last: it flips the shared test DB settings
	t.Run("advisors disabled", func(t *testing.T) {
		settings, err := models.GetSettings(db)
		require.NoError(t, err)

		settings.SaaS.Enabled = new(false)
		err = models.SaveSettings(db, settings)
		require.NoError(t, err)

		res, output, err := s.TestAdvisorCheck(ctx, c, "svc-1")
		require.ErrorIs(t, err, services.ErrAdvisorsDisabled)
		assert.Nil(t, res)
		assert.Empty(t, output)
	})
}

func TestNewInitializesStartCheckChannel(t *testing.T) {
	t.Parallel()
	// New must initialize the on-demand channel so StartChecks can enqueue a
	// run before Run starts draining it.
	s := New(nil, nil, nil, nil)
	require.NotNil(t, s.startCheckCh)
}

func TestUpdateIntervalsBeforeRun(t *testing.T) {
	t.Parallel()
	// UpdateIntervals must not panic when Run has not created the tickers yet
	// (e.g. a settings change on a node that is not the leader).
	s := New(nil, nil, nil, nil)
	assert.NotPanics(t, func() {
		s.UpdateIntervals(time.Hour, time.Minute, time.Second)
	})
}

func TestFilterChecks(t *testing.T) {
	t.Parallel()

	valid := []check.Advisor{
		{
			Category:    "Test",
			Subcategory: "MySQL",
			Checks: []check.Check{
				{Name: "MySQL check V2", Version: 2, Queries: []check.Query{{Type: check.MySQLShow}, {Type: check.MySQLSelect}}},
			},
		},
		{
			Category:    "Test",
			Subcategory: "PostgreSQL",
			Checks: []check.Check{
				{Name: "PostgreSQL check V2", Version: 2, Queries: []check.Query{{Type: check.PostgreSQLShow}, {Type: check.PostgreSQLSelect}}},
			},
		},
		{
			Category:    "Test",
			Subcategory: "MongoDB",
			Checks: []check.Check{
				{Name: "MongoDB check V2", Version: 2, Queries: []check.Query{{Type: check.MongoDBBuildInfo}, {Type: check.MongoDBGetParameter}, {Type: check.MongoDBGetCmdLineOpts}}},
			},
		},
	}

	invalid := []check.Advisor{
		{
			Category:    "Test",
			Subcategory: "CompletelyInvalid",
			Checks: []check.Check{
				{Name: "unsupported version", Version: check.MaxSupportedVersion + 1, Queries: []check.Query{{Type: check.MySQLShow}}},
				{Name: "unsupported type", Version: 2, Queries: []check.Query{{Type: check.Type("RedisInfo")}}},
			},
		},
		{
			Category:    "Test",
			Subcategory: "PartiallyInvalid",
			Checks: []check.Check{
				{Name: "MySQLShow", Version: 2, Queries: []check.Query{{Type: check.MySQLShow}}},
				{Name: "unsupported type", Version: 2, Queries: []check.Query{{Type: check.Type("RedisInfo")}}},
			},
		},
	}

	checks := append(valid, invalid...) //nolint:gocritic

	partiallyValidAdvisor := invalid[1]
	partiallyValidAdvisor.Checks = partiallyValidAdvisor.Checks[0:1] // remove invalid check
	expected := append(valid, partiallyValidAdvisor)                 //nolint:gocritic

	s := New(nil, nil, vmClient, clickhouseDB)
	actual := s.filterSupportedChecks(checks)
	assert.ElementsMatch(t, expected, actual)
}

func TestMinPMMAgents(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		check      check.Check
		minVersion *version.Parsed
	}{
		{name: "MySQLShow", minVersion: pmmAgent3_0_0, check: check.Check{Version: 2, Queries: []check.Query{{Type: check.MySQLShow}}}},
		{name: "MySQLSelect", minVersion: pmmAgent3_0_0, check: check.Check{Version: 2, Queries: []check.Query{{Type: check.MySQLSelect}}}},
		{name: "PostgreSQLShow", minVersion: pmmAgent3_0_0, check: check.Check{Version: 2, Queries: []check.Query{{Type: check.PostgreSQLShow}}}},
		{name: "PostgreSQLSelect", minVersion: pmmAgent3_0_0, check: check.Check{Version: 2, Queries: []check.Query{{Type: check.PostgreSQLSelect}}}},
		{name: "MongoDBGetParameter", minVersion: pmmAgent3_0_0, check: check.Check{Version: 2, Queries: []check.Query{{Type: check.MongoDBGetParameter}}}},
		{name: "MongoDBBuildInfo", minVersion: pmmAgent3_0_0, check: check.Check{Version: 2, Queries: []check.Query{{Type: check.MongoDBBuildInfo}}}},
		{name: "MongoDBGetCmdLineOpts", minVersion: pmmAgent3_0_0, check: check.Check{Version: 2, Queries: []check.Query{{Type: check.MongoDBGetCmdLineOpts}}}},
		{name: "MySQL Family", minVersion: pmmAgent3_0_0, check: check.Check{Version: 2, Queries: []check.Query{{Type: check.MySQLShow}, {Type: check.MySQLSelect}}}},
		{name: "MongoDB Family", minVersion: pmmAgent3_0_0, check: check.Check{Version: 2, Queries: []check.Query{{Type: check.MongoDBBuildInfo}, {Type: check.MongoDBGetParameter}, {Type: check.MongoDBGetCmdLineOpts}}}},
		{name: "PostgreSQL Family", minVersion: pmmAgent3_0_0, check: check.Check{Version: 2, Queries: []check.Query{{Type: check.PostgreSQLShow}, {Type: check.PostgreSQLSelect}}}},
	}

	s := New(nil, nil, vmClient, clickhouseDB)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.minVersion, s.minPMMAgentVersion(test.check))
		})
	}
}

func setup(t *testing.T, db *reform.DB, serviceName, nodeID, pmmAgentVersion string) {
	t.Helper()
	pmmAgent, err := models.CreatePMMAgent(db.Querier, nodeID, nil)
	require.NoError(t, err)

	pmmAgent.Version = pointer.ToStringOrNil(pmmAgentVersion)
	err = db.Update(pmmAgent)
	require.NoError(t, err)

	mysql, err := models.AddNewService(db.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
		ServiceName: serviceName,
		NodeID:      nodeID,
		Address:     new("127.0.0.1"),
		Port:        new(uint16(3306)),
	})
	require.NoError(t, err)

	_, err = models.CreateAgent(db.Querier, models.MySQLdExporterType, &models.CreateAgentParams{
		PMMAgentID: pmmAgent.AgentID,
		ServiceID:  mysql.ServiceID,
	})
	require.NoError(t, err)
}

// setupClients configures actual vm and clickhouse clients for tests that need them.
func setupClients(t *testing.T) {
	t.Helper()
	vmAddr := "http://127.0.0.1:9090/prometheus/"
	clickhouseDSN := "tcp://127.0.0.1:9000/pmm"

	client, err := metrics.NewClient(metrics.Config{Address: vmAddr})
	require.NoError(t, err)
	vmClient = v1.NewAPI(client)

	clickhouseDB, err = sql.Open("clickhouse", clickhouseDSN)
	require.NoError(t, err)

	clickhouseDB.SetConnMaxLifetime(0)
}

func TestFindTargets(t *testing.T) {
	t.Parallel()
	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	s := New(db, nil, vmClient, clickhouseDB)

	t.Run("unknown service", func(t *testing.T) {
		t.Parallel()

		targets, err := s.findTargets(t.Context(), models.PostgreSQLServiceType, nil)
		require.NoError(t, err)
		assert.Empty(t, targets)
	})

	t.Run("different pmm agent versions", func(t *testing.T) {
		t.Parallel()

		node, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{
			NodeName: "test-node",
		})
		require.NoError(t, err)

		setup(t, db, "mysql1", node.NodeID, "")
		setup(t, db, "mysql2", node.NodeID, "2.5.0")
		setup(t, db, "mysql3", node.NodeID, "2.6.0")
		setup(t, db, "mysql4", node.NodeID, "2.6.1")
		setup(t, db, "mysql5", node.NodeID, "2.7.0")

		tests := []struct {
			name               string
			minRequiredVersion *version.Parsed
			count              int
		}{
			{"without version", nil, 5},
			{"version 2.5.0", version.MustParse("2.5.0"), 4},
			{"version 2.6.0", version.MustParse("2.6.0"), 3},
			{"version 2.6.1", version.MustParse("2.6.1"), 2},
			{"version 2.7.0", version.MustParse("2.7.0"), 1},
			{"version 2.9.0", version.MustParse("2.9.0"), 0},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				t.Parallel()

				targets, err := s.findTargets(t.Context(), models.MySQLServiceType, test.minRequiredVersion)
				require.NoError(t, err)
				assert.Len(t, targets, test.count)
			})
		}
	})
}

func TestFindTargetsSkipsOnlyInternalPostgreSQL(t *testing.T) {
	// NOTE: no t.Parallel() - testdb.Open recreates a single shared database, so concurrent
	// testdb tests collide.
	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	s := New(db, nil, vmClient, clickhouseDB)

	// A user service registered on the PMM Server node must still be a valid target.
	setup(t, db, "mysql-on-pmm-node", models.PMMServerNodeID, "")

	mysqlTargets, err := s.findTargets(t.Context(), models.MySQLServiceType, nil)
	require.NoError(t, err)
	require.Len(t, mysqlTargets, 1)
	assert.Equal(t, "mysql-on-pmm-node", mysqlTargets[0].ServiceName)

	// PMM Server's internal PostgreSQL must be skipped, leaving no PostgreSQL targets.
	pgTargets, err := s.findTargets(t.Context(), models.PostgreSQLServiceType, nil)
	require.NoError(t, err)
	assert.Empty(t, pgTargets)
}

func TestFilterChecksByInterval(t *testing.T) {
	t.Parallel()
	s := New(nil, nil, vmClient, clickhouseDB)

	rareCheck := check.Check{Name: "rareCheck", Interval: check.Rare}
	standardCheck := check.Check{Name: "standardCheck", Interval: check.Standard}
	frequentCheck := check.Check{Name: "frequentCheck", Interval: check.Frequent}
	emptyCheck := check.Check{Name: "emptyCheck"}

	checks := map[string]check.Check{
		rareCheck.Name:     rareCheck,
		standardCheck.Name: standardCheck,
		frequentCheck.Name: frequentCheck,
		emptyCheck.Name:    emptyCheck,
	}

	rareChecks := s.filterChecks(checks, check.Rare, nil, nil)
	assert.Equal(t, map[string]check.Check{"rareCheck": rareCheck}, rareChecks)

	standardChecks := s.filterChecks(checks, check.Standard, nil, nil)
	assert.Equal(t, map[string]check.Check{"standardCheck": standardCheck, "emptyCheck": emptyCheck}, standardChecks)

	frequentChecks := s.filterChecks(checks, check.Frequent, nil, nil)
	assert.Equal(t, map[string]check.Check{"frequentCheck": frequentCheck}, frequentChecks)
}

func TestGetFailedChecks(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)

	t.Run("no failed check for service", func(t *testing.T) {
		s := New(db, nil, vmClient, clickhouseDB)

		results, err := s.GetChecksResults(t.Context(), "test_svc")
		assert.Empty(t, results)
		require.NoError(t, err)
	})

	t.Run("non empty failed checks", func(t *testing.T) {
		checkResults := []services.CheckResult{
			{
				CheckName: "test_check",
				Interval:  check.Frequent,
				Target: services.Target{
					ServiceName: "test_svc1",
					ServiceID:   "test_svc1",
					Labels: map[string]string{
						"targetLabel": "targetLabelValue",
					},
				},
				Result: check.Result{
					Summary:     "Check summary",
					Description: "Check description",
					ReadMoreURL: "https://www.example.com",
					Severity:    common.Error,
					Labels: map[string]string{
						"resultLabel": "reslutLabelValue",
					},
				},
			},
			{
				CheckName: "test_check2",
				Interval:  check.Frequent,
				Target: services.Target{
					ServiceName: "test_svc2",
					ServiceID:   "test_svc2",
					Labels: map[string]string{
						"targetLabel": "targetLabelValue",
					},
				},
				Result: check.Result{
					Summary:     "Check summary",
					Description: "Check description",
					ReadMoreURL: "https://www.example.com",
					Severity:    common.Error,
					Labels: map[string]string{
						"resultLabel": "reslutLabelValue",
					},
				},
			},
		}

		s := New(db, nil, vmClient, clickhouseDB)
		s.alertsRegistry.set(checkResults)

		response, err := s.GetChecksResults(t.Context(), "")
		require.NoError(t, err)
		assert.ElementsMatch(t, checkResults, response)
	})

	t.Run("non empty failed checks for specific service", func(t *testing.T) {
		checkResults := []services.CheckResult{
			{
				CheckName: "test_check",
				Interval:  check.Frequent,
				Target: services.Target{
					ServiceName: "test_svc1",
					ServiceID:   "test_svc1",
					Labels: map[string]string{
						"targetLabel": "targetLabelValue",
					},
				},
				Result: check.Result{
					Summary:     "Check summary",
					Description: "Check description",
					ReadMoreURL: "https://www.example.com",
					Severity:    common.Error,
					Labels: map[string]string{
						"resultLabel": "reslutLabelValue",
					},
				},
			},
			{
				CheckName: "test_check2",
				Interval:  check.Frequent,
				Target: services.Target{
					ServiceName: "test_svc2",
					ServiceID:   "test_svc2",
					Labels: map[string]string{
						"targetLabel": "targetLabelValue",
					},
				},
				Result: check.Result{
					Summary:     "Check summary",
					Description: "Check description",
					ReadMoreURL: "https://www.example.com",
					Severity:    common.Error,
					Labels: map[string]string{
						"resultLabel": "reslutLabelValue",
					},
				},
			},
		}

		s := New(db, nil, vmClient, clickhouseDB)
		s.alertsRegistry.set(checkResults)

		response, err := s.GetChecksResults(t.Context(), "test_svc1")
		require.NoError(t, err)
		require.Len(t, response, 1)
		assert.Equal(t, checkResults[0], response[0])
	})

	t.Run("Advisors disabled", func(t *testing.T) {
		s := New(db, nil, vmClient, clickhouseDB)

		settings, err := models.GetSettings(db)
		require.NoError(t, err)

		settings.SaaS.Enabled = new(false)
		err = models.SaveSettings(db, settings)
		require.NoError(t, err)

		results, err := s.GetChecksResults(t.Context(), "test_svc")
		assert.Nil(t, results)
		require.ErrorIs(t, err, services.ErrAdvisorsDisabled)
	})
}

func TestFillQueryPlaceholders(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name         string
		query        string
		placeholders queryPlaceholders
		expected     string
		errString    string
	}

	target := services.Target{
		ServiceID:   "test_service_id",
		ServiceName: "service_name",
		NodeName:    "node_name",
	}

	cases := []testCase{
		{
			name:     "vm query with placeholders",
			query:    "some query with service={{ .ServiceName }} and node={{ .NodeName }}",
			expected: "some query with service=service_name and node=node_name",
			placeholders: queryPlaceholders{
				ServiceName: target.ServiceName,
				NodeName:    target.NodeName,
			},
		},
		{
			name:     "clickhouse query with placeholders",
			query:    "m_docs_scanned FROM metrics WHERE service_id='{{.ServiceID}}' AND period_start >= subtractHours(now(), 1) AND col1 < 10",
			expected: "m_docs_scanned FROM metrics WHERE service_id='test_service_id' AND period_start >= subtractHours(now(), 1) AND col1 < 10",
			placeholders: queryPlaceholders{
				ServiceID: target.ServiceID,
			},
		},
		{
			name:     "vm query without placeholders",
			query:    "some query",
			expected: "some query",
			placeholders: queryPlaceholders{
				ServiceName: target.ServiceName,
				NodeName:    target.NodeName,
			},
		},
		{
			name:     "clickhouse query without placeholders",
			query:    "fingerprint FROM metrics",
			expected: "fingerprint FROM metrics",
			placeholders: queryPlaceholders{
				ServiceID: target.ServiceID,
			},
		},
		{
			name:  "unknown placeholder in query",
			query: "some query with service={{ .ServiceName }} and os={{ .OS }}",
			placeholders: queryPlaceholders{
				ServiceName: target.ServiceName,
			},
			errString: "failed to fill query placeholders: template: query:1:53: executing \"query\" at <.OS>: can't evaluate field OS in type checks.queryPlaceholders",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual, err := fillQueryPlaceholders(tt.query, tt.placeholders)
			if tt.errString == "" {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, actual)
			} else {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.errString)
			}
		})
	}
}

func TestGroupChecksByDB(t *testing.T) {
	t.Parallel()

	checks := map[string]check.Check{
		"mysql_1":        {Name: "mysql_1", Version: 2, Family: check.MySQL},
		"mysql_2":        {Name: "mysql_2", Version: 2, Family: check.MySQL},
		"mysql_3":        {Name: "mysql_3", Version: 2, Family: check.MySQL},
		"postgresql_1":   {Name: "postgresql_1", Version: 2, Family: check.PostgreSQL},
		"postgresql_2":   {Name: "postgresql_2", Version: 2, Family: check.PostgreSQL},
		"postgresql_3":   {Name: "postgresql_3", Version: 2, Family: check.PostgreSQL},
		"mongodb_1":      {Name: "mongodb_1", Version: 2, Family: check.MongoDB},
		"mongodb_2":      {Name: "mongodb_2", Version: 2, Family: check.MongoDB},
		"mongodb_3":      {Name: "mongodb_3", Version: 2, Family: check.MongoDB},
		"mongodb_4":      {Name: "mongodb_4", Version: 2, Family: check.MongoDB},
		"mongodb_5":      {Name: "mongodb_5", Version: 2, Family: check.MongoDB},
		"mongodb_6":      {Name: "mongodb_6", Version: 2, Family: check.MongoDB},
		"missing family": {Name: "missing family", Version: 2},
		"unknown family": {Name: "unknown family", Version: 2, Family: check.Family("RedisFamily")},
	}

	l := logrus.WithField("component", "tests")
	mySQLChecks, postgreSQLChecks, mongoDBChecks := groupChecksByDB(l, checks)

	// checks with a missing or unknown family are skipped
	require.Len(t, mySQLChecks, 3)
	require.Len(t, postgreSQLChecks, 3)
	require.Len(t, mongoDBChecks, 6)

	assert.Equal(t, check.MySQL, mySQLChecks["mysql_1"].Family)
	assert.Equal(t, check.PostgreSQL, postgreSQLChecks["postgresql_1"].Family)
	assert.Equal(t, check.MongoDB, mongoDBChecks["mongodb_1"].Family)
}
