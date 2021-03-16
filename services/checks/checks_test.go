// pmm-managed
// Copyright (C) 2017 Percona LLC
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
	"strings"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	api "github.com/percona-platform/saas/gen/check/retrieval"
	"github.com/percona-platform/saas/pkg/check"
	"github.com/percona/pmm/version"
	promtest "github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/services"
	"github.com/percona/pmm-managed/utils/testdb"
)

const (
	devChecksHost      = "check-dev.percona.com:443"
	devChecksPublicKey = "RWTg+ZmCCjt7O8eWeAmTLAqW+1ozUbpRSKSwNTmO+exlS5KEIPYWuYdX"
	testChecksFile     = "../../testdata/checks/checks.yml"
)

func TestDownloadChecks(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)

	s, err := New(nil, nil, db)
	require.NoError(t, err)
	s.host = devChecksHost
	s.publicKeys = []string{devChecksPublicKey}

	assert.Empty(t, s.GetAllChecks())
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	checks, err := s.downloadChecks(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, checks)
}

func TestLoadLocalChecks(t *testing.T) {
	s, err := New(nil, nil, nil)
	require.NoError(t, err)

	checks, err := s.loadLocalChecks(testChecksFile)
	require.NoError(t, err)
	require.Len(t, checks, 3)

	c1, c2, c3 := checks[0], checks[1], checks[2]

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
}

func TestCollectChecks(t *testing.T) {
	t.Run("collect local checks", func(t *testing.T) {
		sqlDB := testdb.Open(t, models.SkipFixtures, nil)
		db := reform.NewDB(sqlDB, postgresql.Dialect, nil)
		s, err := New(nil, nil, db)
		require.NoError(t, err)
		s.localChecksFile = testChecksFile

		s.collectChecks(context.Background())

		mySQLChecks := s.getMySQLChecks()
		postgreSQLChecks := s.getPostgreSQLChecks()
		mongoDBChecks := s.getMongoDBChecks()
		allChecks := s.GetAllChecks()

		require.Len(t, mySQLChecks, 1)
		require.Len(t, postgreSQLChecks, 1)
		require.Len(t, mongoDBChecks, 1)
		require.Len(t, allChecks, 3)

		assert.Equal(t, check.MySQLShow, mySQLChecks[0].Type)
		assert.Equal(t, check.PostgreSQLSelect, postgreSQLChecks[0].Type)
		assert.Equal(t, check.MongoDBBuildInfo, mongoDBChecks[0].Type)
	})

	t.Run("download checks", func(t *testing.T) {
		sqlDB := testdb.Open(t, models.SkipFixtures, nil)
		db := reform.NewDB(sqlDB, postgresql.Dialect, nil)
		s, err := New(nil, nil, db)
		require.NoError(t, err)
		s.localChecksFile = testChecksFile

		s.collectChecks(context.Background())

		assert.NotEmpty(t, s.mySQLChecks)
		assert.NotEmpty(t, s.postgreSQLChecks)
		assert.NotEmpty(t, s.mongoDBChecks)
	})
}

func TestDisableChecks(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		sqlDB := testdb.Open(t, models.SkipFixtures, nil)
		db := reform.NewDB(sqlDB, postgresql.Dialect, nil)
		s, err := New(nil, nil, db)
		require.NoError(t, err)
		s.localChecksFile = testChecksFile

		s.collectChecks(context.Background())

		checks := s.GetAllChecks()
		assert.Len(t, checks, 3)

		disChecks, err := s.GetDisabledChecks()
		require.NoError(t, err)
		assert.Empty(t, disChecks)

		err = s.DisableChecks([]string{checks[0].Name})
		require.NoError(t, err)

		disChecks, err = s.GetDisabledChecks()
		require.NoError(t, err)
		assert.Len(t, disChecks, 1)
	})

	t.Run("disable same check twice", func(t *testing.T) {
		sqlDB := testdb.Open(t, models.SkipFixtures, nil)
		db := reform.NewDB(sqlDB, postgresql.Dialect, nil)
		s, err := New(nil, nil, db)
		require.NoError(t, err)
		s.localChecksFile = testChecksFile

		s.collectChecks(context.Background())

		checks := s.GetAllChecks()
		assert.Len(t, checks, 3)

		disChecks, err := s.GetDisabledChecks()
		require.NoError(t, err)
		assert.Empty(t, disChecks)

		err = s.DisableChecks([]string{checks[0].Name})
		require.NoError(t, err)

		err = s.DisableChecks([]string{checks[0].Name})
		require.NoError(t, err)

		disChecks, err = s.GetDisabledChecks()
		require.NoError(t, err)
		assert.Len(t, disChecks, 1)
	})

	t.Run("disable unknown check", func(t *testing.T) {
		sqlDB := testdb.Open(t, models.SkipFixtures, nil)
		db := reform.NewDB(sqlDB, postgresql.Dialect, nil)
		s, err := New(nil, nil, db)
		require.NoError(t, err)
		s.localChecksFile = testChecksFile

		s.collectChecks(context.Background())

		err = s.DisableChecks([]string{"unknown_check"})
		require.Error(t, err)

		disChecks, err := s.GetDisabledChecks()
		require.NoError(t, err)
		assert.Empty(t, disChecks)
	})
}

func TestEnableChecks(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		sqlDB := testdb.Open(t, models.SkipFixtures, nil)
		db := reform.NewDB(sqlDB, postgresql.Dialect, nil)
		s, err := New(nil, nil, db)
		require.NoError(t, err)
		s.localChecksFile = testChecksFile

		s.collectChecks(context.Background())

		checks := s.GetAllChecks()
		assert.Len(t, checks, 3)

		err = s.DisableChecks([]string{checks[0].Name, checks[1].Name, checks[2].Name})
		require.NoError(t, err)

		err = s.EnableChecks([]string{checks[0].Name, checks[2].Name})
		require.NoError(t, err)

		disChecks, err := s.GetDisabledChecks()
		require.NoError(t, err)
		assert.Equal(t, []string{checks[1].Name}, disChecks)
	})
}

// A proper unit test could not be written due
// to problems with the code responsible for locating agents
// Once it is fixed rewrite this test to actually run `executeChecks`
// method and test for recorded metrics.
func TestSTTMetrics(t *testing.T) {
	t.Run("check for recorded metrics", func(t *testing.T) {
		s, err := New(nil, nil, nil)
		require.NoError(t, err)
		expected := strings.NewReader(`
		    # HELP pmm_managed_checks_alerts_generated_total Counter of alerts generated per service type per check type
		    # TYPE pmm_managed_checks_alerts_generated_total counter
		    pmm_managed_checks_alerts_generated_total{check_type="MONGODB_BUILDINFO",service_type="mongodb"} 0
		    pmm_managed_checks_alerts_generated_total{check_type="MONGODB_GETCMDLINEOPTS",service_type="mongodb"} 0
		    pmm_managed_checks_alerts_generated_total{check_type="MONGODB_GETPARAMETER",service_type="mongodb"} 0
		    pmm_managed_checks_alerts_generated_total{check_type="MYSQL_SELECT",service_type="mysql"} 0
		    pmm_managed_checks_alerts_generated_total{check_type="MYSQL_SHOW",service_type="mysql"} 0
		    pmm_managed_checks_alerts_generated_total{check_type="POSTGRESQL_SELECT",service_type="postgresql"} 0
		    pmm_managed_checks_alerts_generated_total{check_type="POSTGRESQL_SHOW",service_type="postgresql"} 0
		    # HELP pmm_managed_checks_scripts_executed_total Counter of check scripts executed per service type
		    # TYPE pmm_managed_checks_scripts_executed_total counter
		    pmm_managed_checks_scripts_executed_total{service_type="mongodb"} 0
		    pmm_managed_checks_scripts_executed_total{service_type="mysql"} 0
		    pmm_managed_checks_scripts_executed_total{service_type="postgresql"} 0
		`)
		assert.NoError(t, promtest.CollectAndCompare(s, expected))
	})
}

func TestVerifySignatures(t *testing.T) {
	t.Parallel()

	t.Run("normal", func(t *testing.T) {
		t.Parallel()

		s, err := New(nil, nil, nil)
		require.NoError(t, err)
		s.host = devChecksHost

		validKey := "RWSdGihBPffV2c4IysqHAIxc5c5PLfmQStbRPkuLXDr3igJOqFWt7aml"
		invalidKey := "RWSdGihBPffV2c4IysqHAIxc5c5PLfmQStbRPkuLXDr3igJO+INVALID"

		s.publicKeys = []string{invalidKey, validKey}

		validSign := strings.TrimSpace(`
untrusted comment: signature from minisign secret key
RWSdGihBPffV2W/zvmIiTLh8UnocoF3OcwmczGdZ+zM13eRnm2Qq9YxfQ9cLzAp1dA5w7C5a3Cp5D7jlYiydu5hqZhJUxJt/ugg=
trusted comment: some comment
uEF33ScMPYpvHvBKv8+yBkJ9k4+DCfV4nDs6kKYwGhalvkkqwWkyfJffO+KW7a1m3y42WHpOnzBxLJeU/AuzDw==
`)

		invalidSign := strings.TrimSpace(`
untrusted comment: signature from minisign secret key
RWSdGihBPffV2W/zvmIiTLh8UnocoF3OcwmczGdZ+zM13eRnm2Qq9YxfQ9cLzAp1dA5w7C5a3Cp5D7jlYiydu5hqZhJ+INVALID=
trusted comment: some comment
uEF33ScMPYpvHvBKv8+yBkJ9k4+DCfV4nDs6kKYwGhalvkkqwWkyfJffO+KW7a1m3y42WHpOnzBxLJ+INVALID==
`)

		resp := api.GetAllChecksResponse{
			File:       "random data",
			Signatures: []string{invalidSign, validSign},
		}

		err = s.verifySignatures(&resp)
		assert.NoError(t, err)
	})

	t.Run("empty signatures", func(t *testing.T) {
		t.Parallel()

		s, err := New(nil, nil, nil)
		require.NoError(t, err)
		s.host = devChecksHost
		s.publicKeys = []string{"RWSdGihBPffV2c4IysqHAIxc5c5PLfmQStbRPkuLXDr3igJOqFWt7aml"}

		resp := api.GetAllChecksResponse{
			File:       "random data",
			Signatures: []string{},
		}

		err = s.verifySignatures(&resp)
		assert.EqualError(t, err, "zero signatures received")
	})
}

func TestGetSecurityCheckResults(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)

	t.Run("STT disabled", func(t *testing.T) {
		s, err := New(nil, nil, db)
		require.NoError(t, err)
		results, err := s.GetSecurityCheckResults()
		assert.Nil(t, results)
		assert.EqualError(t, err, services.ErrSTTDisabled.Error())
	})

	t.Run("STT enabled", func(t *testing.T) {
		s, err := New(nil, nil, db)
		require.NoError(t, err)
		settings, err := models.GetSettings(db)
		require.NoError(t, err)

		settings.SaaS.STTEnabled = true
		err = models.SaveSettings(db, settings)
		require.NoError(t, err)

		results, err := s.GetSecurityCheckResults()
		assert.Empty(t, results)
		require.NoError(t, err)
	})
}

func TestStartChecks(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)

	t.Run("stt disabled", func(t *testing.T) {
		s, err := New(nil, nil, db)
		require.NoError(t, err)
		err = s.StartChecks(context.Background(), "")
		assert.EqualError(t, err, services.ErrSTTDisabled.Error())
	})

	t.Run("unknown interval", func(t *testing.T) {
		s, err := New(nil, nil, db)
		require.NoError(t, err)
		settings, err := models.GetSettings(db)
		require.NoError(t, err)

		settings.SaaS.STTEnabled = true
		err = models.SaveSettings(db, settings)
		require.NoError(t, err)

		err = s.StartChecks(context.Background(), check.Interval("unknown"))
		assert.EqualError(t, err, "unknown check interval: unknown")
	})

	t.Run("stt enabled", func(t *testing.T) {
		var ams mockAlertmanagerService
		ams.On("SendAlerts", mock.Anything, mock.Anything).Return()

		s, err := New(nil, &ams, db)
		require.NoError(t, err)
		settings, err := models.GetSettings(db)
		require.NoError(t, err)

		settings.SaaS.STTEnabled = true
		err = models.SaveSettings(db, settings)
		require.NoError(t, err)

		err = s.StartChecks(context.Background(), "")
		require.NoError(t, err)
	})
}

func TestFilterChecks(t *testing.T) {
	t.Parallel()

	valid := []check.Check{
		{Name: "MySQLShow", Version: 1, Type: check.MySQLShow},
		{Name: "MySQLSelect", Version: 1, Type: check.MySQLSelect},
		{Name: "PostgreSQLShow", Version: 1, Type: check.PostgreSQLShow},
		{Name: "PostgreSQLSelect", Version: 1, Type: check.PostgreSQLSelect},
		{Name: "MongoDBGetParameter", Version: 1, Type: check.MongoDBGetParameter},
		{Name: "MongoDBBuildInfo", Version: 1, Type: check.MongoDBBuildInfo},
		{Name: "MongoDBGetCmdLineOpts", Version: 1, Type: check.MongoDBGetCmdLineOpts},
	}

	invalid := []check.Check{
		{Name: "unsupported version", Version: maxSupportedVersion + 1, Type: check.MySQLShow},
		{Name: "unsupported type", Version: 1, Type: check.Type("RedisInfo")},
		{Name: "missing type", Version: 1},
	}

	checks := append(valid, invalid...)

	s, err := New(nil, nil, nil)
	require.NoError(t, err)
	actual := s.filterSupportedChecks(checks)
	assert.ElementsMatch(t, valid, actual)
}

func TestGroupChecksByDB(t *testing.T) {
	t.Parallel()

	checks := []check.Check{
		{Name: "MySQLShow", Version: 1, Type: check.MySQLShow},
		{Name: "MySQLSelect", Version: 1, Type: check.MySQLSelect},
		{Name: "PostgreSQLShow", Version: 1, Type: check.PostgreSQLShow},
		{Name: "PostgreSQLSelect", Version: 1, Type: check.PostgreSQLSelect},
		{Name: "MongoDBGetParameter", Version: 1, Type: check.MongoDBGetParameter},
		{Name: "MongoDBBuildInfo", Version: 1, Type: check.MongoDBBuildInfo},
		{Name: "MongoDBGetCmdLineOpts", Version: 1, Type: check.MongoDBGetCmdLineOpts},
		{Name: "unsupported type", Version: 1, Type: check.Type("RedisInfo")},
		{Name: "missing type", Version: 1},
	}

	s, err := New(nil, nil, nil)
	require.NoError(t, err)
	mySQLChecks, postgreSQLChecks, mongoDBChecks := s.groupChecksByDB(checks)

	require.Len(t, mySQLChecks, 2)
	require.Len(t, postgreSQLChecks, 2)
	require.Len(t, mongoDBChecks, 3)

	assert.Equal(t, check.MySQLShow, mySQLChecks[0].Type)
	assert.Equal(t, check.MySQLSelect, mySQLChecks[1].Type)

	assert.Equal(t, check.PostgreSQLShow, postgreSQLChecks[0].Type)
	assert.Equal(t, check.PostgreSQLSelect, postgreSQLChecks[1].Type)

	assert.Equal(t, check.MongoDBGetParameter, mongoDBChecks[0].Type)
	assert.Equal(t, check.MongoDBBuildInfo, mongoDBChecks[1].Type)
	assert.Equal(t, check.MongoDBGetCmdLineOpts, mongoDBChecks[2].Type)
}

func setup(t *testing.T, db *reform.DB, serviceName, nodeID, pmmAgentVersion string) {
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

func TestFindTargets(t *testing.T) {
	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	s, err := New(nil, nil, db)
	require.NoError(t, err)

	t.Run("unknown service", func(t *testing.T) {
		t.Parallel()

		targets, err := s.findTargets(models.PostgreSQLServiceType, nil)
		require.NoError(t, err)
		assert.Len(t, targets, 0)
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

	rareCheck := check.Check{Name: "rareCheck", Interval: check.Rare}
	standardCheck := check.Check{Name: "standardCheck", Interval: check.Standard}
	frequentCheck := check.Check{Name: "frequentCheck", Interval: check.Frequent}
	emptyCheck := check.Check{Name: "emptyCheck"}

	checks := []check.Check{rareCheck, standardCheck, frequentCheck, emptyCheck}

	rareChecks := filterChecksByInterval(checks, check.Rare)
	assert.ElementsMatch(t, []check.Check{rareCheck}, rareChecks)

	standardChecks := filterChecksByInterval(checks, check.Standard)
	assert.ElementsMatch(t, []check.Check{standardCheck, emptyCheck}, standardChecks)

	frequentChecks := filterChecksByInterval(checks, check.Frequent)
	assert.ElementsMatch(t, []check.Check{frequentCheck}, frequentChecks)
}
