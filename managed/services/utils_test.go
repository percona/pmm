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

package services

import (
	"testing"

	"github.com/percona-platform/saas/pkg/check"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	mySQLChecks, postgreSQLChecks, mongoDBChecks := GroupChecksByDB(l, checks)

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
	assert.Equal(t, check.PostgreSQL, postgreSQLChecks["PostrgeSQL family V2"].Family)
	assert.Equal(t, check.MongoDB, mongoDBChecks["MongoDB family V2"].Family)
}
