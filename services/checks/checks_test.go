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
	"os"
	"strings"
	"testing"

	api "github.com/percona-platform/saas/gen/check/retrieval"
	"github.com/percona-platform/saas/pkg/check"
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
)

func TestDownloadChecks(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		s := New(nil, nil, nil, "2.5.0")
		s.host = devChecksHost
		s.publicKeys = []string{devChecksPublicKey}

		assert.Empty(t, s.getMySQLChecks())
		assert.Empty(t, s.getPostgreSQLChecks())
		assert.Empty(t, s.getMongoDBChecks())
		ctx, cancel := context.WithTimeout(context.Background(), downloadTimeout)
		defer cancel()

		checks, err := s.downloadChecks(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, checks)
	})
}

func TestLoadLocalChecks(t *testing.T) {
	s := New(nil, nil, nil, "2.5.0")

	checks, err := s.loadLocalChecks("../../testdata/checks/checks.yml")
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
		err := os.Setenv("PERCONA_TEST_CHECKS_FILE", "../../testdata/checks/checks.yml")
		require.NoError(t, err)
		defer os.Unsetenv("PERCONA_TEST_CHECKS_FILE") //nolint:errcheck

		s := New(nil, nil, nil, "2.5.0")
		s.collectChecks(context.Background())

		mySQLChecks := s.getMySQLChecks()
		postgreSQLChecks := s.getPostgreSQLChecks()
		mongoDBChecks := s.getMongoDBChecks()

		require.Len(t, mySQLChecks, 1)
		require.Len(t, postgreSQLChecks, 1)
		require.Len(t, mongoDBChecks, 1)

		assert.Equal(t, check.MySQLShow, mySQLChecks[0].Type)
		assert.Equal(t, check.PostgreSQLSelect, postgreSQLChecks[0].Type)
		assert.Equal(t, check.MongoDBBuildInfo, mongoDBChecks[0].Type)
	})

	t.Run("download checks", func(t *testing.T) {
		s := New(nil, nil, nil, "2.5.0")
		s.collectChecks(context.Background())

		assert.NotEmpty(t, s.mySQLChecks)
		assert.NotEmpty(t, s.postgreSQLChecks)
		assert.NotEmpty(t, s.mongoDBChecks)
	})
}

func TestVerifySignatures(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		s := New(nil, nil, nil, "2.5.0")
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

		err := s.verifySignatures(&resp)
		assert.NoError(t, err)
	})

	t.Run("empty signatures", func(t *testing.T) {
		s := New(nil, nil, nil, "2.5.0")
		s.host = devChecksHost
		s.publicKeys = []string{"RWSdGihBPffV2c4IysqHAIxc5c5PLfmQStbRPkuLXDr3igJOqFWt7aml"}

		resp := api.GetAllChecksResponse{
			File:       "random data",
			Signatures: []string{},
		}

		err := s.verifySignatures(&resp)
		assert.EqualError(t, err, "zero signatures received")
	})
}

func TestStartChecks(t *testing.T) {
	t.Run("stt disabled", func(t *testing.T) {
		sqlDB := testdb.Open(t, models.SkipFixtures, nil)
		db := reform.NewDB(sqlDB, postgresql.Dialect, nil)

		defer func() {
			require.NoError(t, sqlDB.Close())
		}()

		s := New(nil, nil, db, "2.5.0")
		err := s.StartChecks(context.Background())
		assert.EqualError(t, err, services.ErrSTTDisabled.Error())
	})

	t.Run("stt enabled", func(t *testing.T) {
		sqlDB := testdb.Open(t, models.SkipFixtures, nil)
		db := reform.NewDB(sqlDB, postgresql.Dialect, nil)

		defer func() {
			require.NoError(t, sqlDB.Close())
		}()

		var ar mockAlertRegistry
		ar.On("RemovePrefix", mock.Anything, mock.Anything).Return()

		s := New(nil, &ar, db, "2.5.0")
		settings, err := models.GetSettings(db)
		require.NoError(t, err)

		settings.SaaS.STTEnabled = true
		err = models.SaveSettings(db, settings)
		require.NoError(t, err)

		err = s.StartChecks(context.Background())
		require.NoError(t, err)
	})
}

func TestFilterChecks(t *testing.T) {
	valid := []check.Check{
		{Name: "MySQLShow", Version: 1, Type: check.MySQLShow},
		{Name: "MySQLSelect", Version: 1, Type: check.MySQLSelect},
		{Name: "PostgreSQLShow", Version: 1, Type: check.PostgreSQLShow},
		{Name: "PostgreSQLSelect", Version: 1, Type: check.PostgreSQLSelect},
		{Name: "MongoDBGetParameter", Version: 1, Type: check.MongoDBGetParameter},
		{Name: "MongoDBBuildInfo", Version: 1, Type: check.MongoDBBuildInfo},
	}

	invalid := []check.Check{
		{Name: "unsupported version", Version: maxSupportedVersion + 1, Type: check.MySQLShow},
		{Name: "unsupported type", Version: 1, Type: check.Type("RedisInfo")},
		{Name: "missing type", Version: 1},
	}

	checks := append(valid, invalid...)

	s := New(nil, nil, nil, "2.5.0")

	actual := s.filterSupportedChecks(checks)

	assert.ElementsMatch(t, valid, actual)
}

func TestGroupChecksByDB(t *testing.T) {
	checks := []check.Check{
		{Name: "MySQLShow", Version: 1, Type: check.MySQLShow},
		{Name: "MySQLSelect", Version: 1, Type: check.MySQLSelect},
		{Name: "PostgreSQLShow", Version: 1, Type: check.PostgreSQLShow},
		{Name: "PostgreSQLSelect", Version: 1, Type: check.PostgreSQLSelect},
		{Name: "MongoDBGetParameter", Version: 1, Type: check.MongoDBGetParameter},
		{Name: "MongoDBBuildInfo", Version: 1, Type: check.MongoDBBuildInfo},
		{Name: "unsupported type", Version: 1, Type: check.Type("RedisInfo")},
		{Name: "missing type", Version: 1},
	}

	s := New(nil, nil, nil, "2.5.0")
	mySQLChecks, postgreSQLChecks, mongoDBChecks := s.groupChecksByDB(checks)

	require.Len(t, mySQLChecks, 2)
	require.Len(t, postgreSQLChecks, 2)
	require.Len(t, mongoDBChecks, 2)

	assert.Equal(t, check.MySQLShow, mySQLChecks[0].Type)
	assert.Equal(t, check.MySQLSelect, mySQLChecks[1].Type)

	assert.Equal(t, check.PostgreSQLShow, postgreSQLChecks[0].Type)
	assert.Equal(t, check.PostgreSQLSelect, postgreSQLChecks[1].Type)

	assert.Equal(t, check.MongoDBGetParameter, mongoDBChecks[0].Type)
	assert.Equal(t, check.MongoDBBuildInfo, mongoDBChecks[1].Type)
}

func TestFindTargets(t *testing.T) {
	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	defer func() {
		require.NoError(t, sqlDB.Close())
	}()
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	s := New(nil, nil, db, "2.5.0")

	targets, err := s.findTargets(models.PostgreSQLServiceType)
	require.NoError(t, err)
	assert.Len(t, targets, 0)
}
