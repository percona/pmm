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
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/percona/saas/pkg/check"
	"github.com/percona/saas/pkg/common"
	metrics "github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/version"
)

const (
	testChecksFile = "../../testdata/checks/checks.yml"
)

var (
	vmClient     v1.API
	clickhouseDB *sql.DB
)

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
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		dChecks, err := s.loadBuiltinAdvisors(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, dChecks)

		s.CollectAdvisors(ctx)

		checks, err = s.GetAdvisors()
		require.NoError(t, err)
		assert.NotEmpty(t, checks)
	})

	t.Run("advisors are loaded with telemetry disabled", func(t *testing.T) {
		_, err := models.UpdateSettings(db.Querier, &models.ChangeSettingsParams{
			EnableTelemetry: pointer.ToBool(false),
		})
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		dChecks, err := s.loadBuiltinAdvisors(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, dChecks)

		checks, err := s.GetAdvisors()
		require.NoError(t, err)
		assert.NotEmpty(t, checks)
	})
}

func TestLoadLocalChecks(t *testing.T) {
	s := New(nil, nil, vmClient, clickhouseDB)

	checks, err := s.loadCustomChecks(testChecksFile)
	require.NoError(t, err)
	require.Len(t, checks, 5)

	c1, c2, c3, c4, c5 := checks[0], checks[1], checks[2], checks[3], checks[4]

	assert.Equal(t, check.PostgreSQLSelect, c1.Type)
	assert.Equal(t, "good_check_pg", c1.Name)
	assert.Equal(t, uint32(1), c1.Version)
	assert.Equal(t, "rolpassword FROM pg_authid WHERE rolcanlogin", c1.Query)

	assert.Equal(t, check.MySQLShow, c2.Type)
	assert.Equal(t, "bad_check_mysql", c2.Name)
	assert.Equal(t, uint32(1), c2.Version)
	assert.Equal(t, "VARIABLES LIKE 'version%'", c2.Query)

	assert.Equal(t, check.MongoDBBuildInfo, c3.Type)
	assert.Equal(t, "good_check_mongo", c3.Name)
	assert.Equal(t, uint32(1), c3.Version)
	assert.Empty(t, c3.Query)

	assert.Equal(t, check.MongoDBReplSetGetStatus, c4.Type)
	assert.Equal(t, "check_mongo_replSetGetStatus", c4.Name)
	assert.Equal(t, uint32(1), c4.Version)
	assert.Empty(t, c4.Query)

	assert.Equal(t, check.MongoDBGetDiagnosticData, c5.Type)
	assert.Equal(t, "check_mongo_getDiagnosticData", c5.Name)
	assert.Equal(t, uint32(1), c5.Version)
	assert.Empty(t, c5.Query)
}

func TestCollectAdvisors(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)

	t.Run("collect local checks", func(t *testing.T) {
		s := New(db, nil, vmClient, clickhouseDB)
		s.customCheckFile = testChecksFile

		s.CollectAdvisors(context.Background())

		advisors, err := s.GetAdvisors()
		require.NoError(t, err)
		require.Len(t, advisors, 1)

		advisor := advisors[0]
		require.Equal(t, "dev", advisor.Name)
		require.Equal(t, "Dev Advisor", advisor.Summary)
		require.Equal(t, "Advisor used for developing checks", advisor.Description)
		require.Equal(t, "development", advisor.Category)
		require.Empty(t, advisor.Tiers)
		require.Len(t, advisor.Checks, 5)

		checkNames := make([]string, 0, len(advisor.Checks))
		for _, c := range advisor.Checks {
			checkNames = append(checkNames, c.Name)
		}
		assert.ElementsMatch(t, []string{
			"bad_check_mysql",
			"good_check_pg",
			"good_check_mongo",
			"check_mongo_replSetGetStatus",
			"check_mongo_getDiagnosticData",
		}, checkNames)
	})

	t.Run("download checks", func(t *testing.T) {
		s := New(db, nil, vmClient, clickhouseDB)

		s.CollectAdvisors(context.Background())

		checks, err := s.GetChecks()
		require.NoError(t, err)
		require.NotEmpty(t, checks)

		advisors, err := s.GetAdvisors()
		require.NoError(t, err)
		require.NotEmpty(t, s.advisors)

		checksFromAdvisors := make(map[string]check.Check)
		for _, advisor := range advisors {
			for _, c := range advisor.Checks {
				checksFromAdvisors[c.Name] = c
			}
		}

		assert.Equal(t, checks, checksFromAdvisors)
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
		s.customCheckFile = testChecksFile

		s.CollectAdvisors(context.Background())

		checks, err := s.GetChecks()
		require.NoError(t, err)
		assert.Len(t, checks, 5)

		disChecks, err := s.GetDisabledChecks()
		require.NoError(t, err)
		assert.Empty(t, disChecks)

		err = s.DisableChecks([]string{checks["bad_check_mysql"].Name})
		require.NoError(t, err)

		disChecks, err = s.GetDisabledChecks()
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
		s.customCheckFile = testChecksFile

		s.CollectAdvisors(context.Background())

		checks, err := s.GetChecks()
		require.NoError(t, err)
		assert.Len(t, checks, 5)

		disChecks, err := s.GetDisabledChecks()
		require.NoError(t, err)
		assert.Empty(t, disChecks)

		err = s.DisableChecks([]string{checks["bad_check_mysql"].Name})
		require.NoError(t, err)

		err = s.DisableChecks([]string{checks["bad_check_mysql"].Name})
		require.NoError(t, err)

		disChecks, err = s.GetDisabledChecks()
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
		s.customCheckFile = testChecksFile

		s.CollectAdvisors(context.Background())

		err := s.DisableChecks([]string{"unknown_check"})
		require.Error(t, err)

		disChecks, err := s.GetDisabledChecks()
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
		s.customCheckFile = testChecksFile

		s.CollectAdvisors(context.Background())

		checks, err := s.GetChecks()
		require.NoError(t, err)
		assert.Len(t, checks, 5)

		err = s.DisableChecks([]string{checks["bad_check_mysql"].Name, checks["good_check_pg"].Name, checks["good_check_mongo"].Name})
		require.NoError(t, err)

		err = s.EnableChecks([]string{checks["good_check_pg"].Name, checks["good_check_mongo"].Name})
		require.NoError(t, err)

		disChecks, err := s.GetDisabledChecks()
		require.NoError(t, err)
		assert.Equal(t, []string{checks["bad_check_mysql"].Name}, disChecks)

		enabledChecksCount := len(checks) - len(disChecks)
		assert.Equal(t, 4, enabledChecksCount)
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
		s.customCheckFile = testChecksFile

		s.CollectAdvisors(context.Background())

		checks, err := s.GetChecks()
		require.NoError(t, err)
		assert.Len(t, checks, 5)

		// change all check intervals from standard to rare
		params := make(map[string]check.Interval)
		for _, c := range checks {
			params[c.Name] = check.Rare
		}
		err = s.ChangeInterval(params)
		require.NoError(t, err)

		updatedChecks, err := s.GetChecks()
		require.NoError(t, err)
		for _, c := range updatedChecks {
			assert.Equal(t, check.Rare, c.Interval)
		}

		t.Run("preserve intervals on restarts", func(t *testing.T) {
			err = s.runChecksGroup(context.Background(), "")
			require.NoError(t, err)

			checks, err := s.GetChecks()
			require.NoError(t, err)
			for _, c := range checks {
				assert.Equal(t, check.Rare, c.Interval)
			}
		})
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
		s.customCheckFile = testChecksFile

		err := s.runChecksGroup(context.Background(), "unknown")
		assert.EqualError(t, err, "unknown check interval: unknown")
	})

	t.Run("advisors enabled", func(t *testing.T) {
		s := New(db, nil, vmClient, clickhouseDB)

		s.customCheckFile = testChecksFile
		s.CollectAdvisors(context.Background())
		assert.NotEmpty(t, s.advisors)
		assert.NotEmpty(t, s.checks)

		err := s.runChecksGroup(context.Background(), "")
		require.NoError(t, err)
	})

	t.Run("advisors disabled", func(t *testing.T) {
		s := New(db, nil, vmClient, clickhouseDB)

		settings, err := models.GetSettings(db)
		require.NoError(t, err)

		settings.SaaS.Enabled = pointer.ToBool(false)
		err = models.SaveSettings(db, settings)
		require.NoError(t, err)

		err = s.runChecksGroup(context.Background(), "")
		assert.ErrorIs(t, err, services.ErrAdvisorsDisabled)
	})
}

func TestFilterChecks(t *testing.T) {
	t.Parallel()

	valid := []check.Advisor{
		{
			Name:        "mysql_advisor",
			Summary:     "MySQL advisor",
			Description: "Test mySQL advisor",
			Category:    "test",
			Checks: []check.Check{
				{Name: "MySQLShow", Version: 1, Type: check.MySQLShow},
				{Name: "MySQLSelect", Version: 1, Type: check.MySQLSelect},
				{Name: "MySQL check V2", Version: 2, Queries: []check.Query{{Type: check.MySQLShow}, {Type: check.MySQLSelect}}},
			},
		},
		{
			Name:        "postgresql_advisor",
			Summary:     "PostgreSQL advisor",
			Description: "Test postgreSQL advisor",
			Category:    "test",
			Checks: []check.Check{
				{Name: "PostgreSQLShow", Version: 1, Type: check.PostgreSQLShow},
				{Name: "PostgreSQLSelect", Version: 1, Type: check.PostgreSQLSelect},
				{Name: "PostgreSQL check V2", Version: 2, Queries: []check.Query{{Type: check.PostgreSQLShow}, {Type: check.PostgreSQLSelect}}},
			},
		},
		{
			Name:        "mongodb_advisor",
			Summary:     "MongoDB advisor",
			Description: "Test mongoDB advisor",
			Category:    "test",
			Checks: []check.Check{
				{Name: "MongoDBGetParameter", Version: 1, Type: check.MongoDBGetParameter},
				{Name: "MongoDBBuildInfo", Version: 1, Type: check.MongoDBBuildInfo},
				{Name: "MongoDBGetCmdLineOpts", Version: 1, Type: check.MongoDBGetCmdLineOpts},
				{Name: "MongoDBReplSetGetStatus", Version: 1, Type: check.MongoDBReplSetGetStatus},
				{Name: "MongoDBGetDiagnosticData", Version: 1, Type: check.MongoDBGetDiagnosticData},
				{Name: "MongoDB check V2", Version: 2, Queries: []check.Query{{Type: check.MongoDBBuildInfo}, {Type: check.MongoDBGetParameter}, {Type: check.MongoDBGetCmdLineOpts}}},
			},
		},
	}

	invalid := []check.Advisor{
		{
			Name:        "completely_invalid_advisor",
			Summary:     "Completely invalid advisor",
			Description: "Test advisor that contains only unsupported checks",
			Category:    "test",
			Checks: []check.Check{
				{Name: "unsupported version", Version: maxSupportedVersion + 1, Type: check.MySQLShow},
				{Name: "unsupported type", Version: 1, Type: check.Type("RedisInfo")},
			},
		},
		{
			Name:        "partially_invalid_advisor",
			Summary:     "Partially invalid advisor",
			Description: "Test advisor that contains some unsupported checks",
			Category:    "test",
			Checks: []check.Check{
				{Name: "MySQLShow", Version: 1, Type: check.MySQLShow},
				{Name: "missing type", Version: 1},
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
		{name: "MySQLShow", minVersion: pmmAgent2_6_0, check: check.Check{Version: 1, Type: check.MySQLShow}},
		{name: "MySQLSelect", minVersion: pmmAgent2_6_0, check: check.Check{Version: 1, Type: check.MySQLSelect}},
		{name: "PostgreSQLShow", minVersion: pmmAgent2_6_0, check: check.Check{Version: 1, Type: check.PostgreSQLShow}},
		{name: "PostgreSQLSelect", minVersion: pmmAgent2_6_0, check: check.Check{Version: 1, Type: check.PostgreSQLSelect}},
		{name: "MongoDBGetParameter", minVersion: pmmAgent2_6_0, check: check.Check{Version: 1, Type: check.MongoDBGetParameter}},
		{name: "MongoDBBuildInfo", minVersion: pmmAgent2_6_0, check: check.Check{Version: 1, Type: check.MongoDBBuildInfo}},
		{name: "MongoDBGetCmdLineOpts", minVersion: pmmAgent2_7_0, check: check.Check{Version: 1, Type: check.MongoDBGetCmdLineOpts}},
		{name: "MySQL Family", minVersion: pmmAgent2_6_0, check: check.Check{Version: 2, Queries: []check.Query{{Type: check.MySQLShow}, {Type: check.MySQLSelect}}}},
		{name: "MongoDB Family", minVersion: pmmAgent2_7_0, check: check.Check{Version: 2, Queries: []check.Query{{Type: check.MongoDBBuildInfo}, {Type: check.MongoDBGetParameter}, {Type: check.MongoDBGetCmdLineOpts}}}},
		{name: "PostgreSQL Family", minVersion: pmmAgent2_6_0, check: check.Check{Version: 2, Queries: []check.Query{{Type: check.PostgreSQLShow}, {Type: check.PostgreSQLSelect}}}},
	}

	s := New(nil, nil, vmClient, clickhouseDB)

	for _, test := range tests {
		test := test

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
		Address:     pointer.ToString("127.0.0.1"),
		Port:        pointer.ToUint16(3306),
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

		targets, err := s.findTargets(models.PostgreSQLServiceType, nil)
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
			test := test

			t.Run(test.name, func(t *testing.T) {
				t.Parallel()

				targets, err := s.findTargets(models.MySQLServiceType, test.minRequiredVersion)
				require.NoError(t, err)
				assert.Len(t, targets, test.count)
			})
		}
	})
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

		results, err := s.GetChecksResults(context.Background(), "test_svc")
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

		response, err := s.GetChecksResults(context.Background(), "")
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

		response, err := s.GetChecksResults(context.Background(), "test_svc1")
		require.NoError(t, err)
		require.Len(t, response, 1)
		assert.Equal(t, checkResults[0], response[0])
	})

	t.Run("Advisors disabled", func(t *testing.T) {
		s := New(db, nil, vmClient, clickhouseDB)

		settings, err := models.GetSettings(db)
		require.NoError(t, err)

		settings.SaaS.Enabled = pointer.ToBool(false)
		err = models.SaveSettings(db, settings)
		require.NoError(t, err)

		results, err := s.GetChecksResults(context.Background(), "test_svc")
		assert.Nil(t, results)
		assert.ErrorIs(t, err, services.ErrAdvisorsDisabled)
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual, err := fillQueryPlaceholders(tt.query, tt.placeholders)
			if tt.errString == "" {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, actual)
			} else {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.errString)
			}
		})
	}
}

func TestGroupChecksByDB(t *testing.T) {
	t.Parallel()

	checks := map[string]check.Check{
		"MySQLShow":                {Name: "MySQLShow", Version: 1, Type: check.MySQLShow},
		"MySQLSelect":              {Name: "MySQLSelect", Version: 1, Type: check.MySQLSelect},
		"PostgreSQLShow":           {Name: "PostgreSQLShow", Version: 1, Type: check.PostgreSQLShow},
		"PostgreSQLSelect":         {Name: "PostgreSQLSelect", Version: 1, Type: check.PostgreSQLSelect},
		"MongoDBGetParameter":      {Name: "MongoDBGetParameter", Version: 1, Type: check.MongoDBGetParameter},
		"MongoDBBuildInfo":         {Name: "MongoDBBuildInfo", Version: 1, Type: check.MongoDBBuildInfo},
		"MongoDBGetCmdLineOpts":    {Name: "MongoDBGetCmdLineOpts", Version: 1, Type: check.MongoDBGetCmdLineOpts},
		"MongoDBReplSetGetStatus":  {Name: "MongoDBReplSetGetStatus", Version: 1, Type: check.MongoDBReplSetGetStatus},
		"MongoDBGetDiagnosticData": {Name: "MongoDBGetDiagnosticData", Version: 1, Type: check.MongoDBGetDiagnosticData},
		"unsupported type":         {Name: "unsupported type", Version: 1, Type: check.Type("RedisInfo")},
		"missing type":             {Name: "missing type", Version: 1},
		"MySQL family V2":          {Name: "MySQL family V2", Version: 2, Family: check.MySQL},
		"PostgreSQL family V2":     {Name: "PostgreSQL family V2", Version: 2, Family: check.PostgreSQL},
		"MongoDB family V2":        {Name: "MongoDB family V2", Version: 2, Family: check.MongoDB},
		"missing family":           {Name: "missing family", Version: 2},
	}

	l := logrus.WithField("component", "tests")
	mySQLChecks, postgreSQLChecks, mongoDBChecks := groupChecksByDB(l, checks)

	require.Len(t, mySQLChecks, 3)
	require.Len(t, postgreSQLChecks, 3)
	require.Len(t, mongoDBChecks, 6)

	// V1 checks
	assert.Equal(t, check.MySQLShow, mySQLChecks["MySQLShow"].Type)
	assert.Equal(t, check.MySQLSelect, mySQLChecks["MySQLSelect"].Type)

	assert.Equal(t, check.PostgreSQLShow, postgreSQLChecks["PostgreSQLShow"].Type)
	assert.Equal(t, check.PostgreSQLSelect, postgreSQLChecks["PostgreSQLSelect"].Type)

	assert.Equal(t, check.MongoDBGetParameter, mongoDBChecks["MongoDBGetParameter"].Type)
	assert.Equal(t, check.MongoDBBuildInfo, mongoDBChecks["MongoDBBuildInfo"].Type)
	assert.Equal(t, check.MongoDBGetCmdLineOpts, mongoDBChecks["MongoDBGetCmdLineOpts"].Type)
	assert.Equal(t, check.MongoDBReplSetGetStatus, mongoDBChecks["MongoDBReplSetGetStatus"].Type)
	assert.Equal(t, check.MongoDBGetDiagnosticData, mongoDBChecks["MongoDBGetDiagnosticData"].Type)

	// V2 checks
	assert.Equal(t, check.MySQL, mySQLChecks["MySQL family V2"].Family)
	assert.Equal(t, check.PostgreSQL, postgreSQLChecks["PostgreSQL family V2"].Family)
	assert.Equal(t, check.MongoDB, mongoDBChecks["MongoDB family V2"].Family)
}
