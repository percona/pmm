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

package services

import (
	"github.com/percona-platform/saas/pkg/check"
	"github.com/sirupsen/logrus"
)

// GroupChecksByDB splits provided checks by database and returns three
// slices: for MySQL, for PostgreSQL and for MongoDB.
func GroupChecksByDB(
	l *logrus.Entry,
	checks map[string]check.Check,
) (map[string]check.Check, map[string]check.Check, map[string]check.Check) {
	mySQLChecks := make(map[string]check.Check)
	postgreSQLChecks := make(map[string]check.Check)
	mongoDBChecks := make(map[string]check.Check)
	for _, c := range checks {
		switch c.Version {
		case 1:
			switch c.Type {
			case check.MySQLSelect:
				fallthrough
			case check.MySQLShow:
				mySQLChecks[c.Name] = c

			case check.PostgreSQLSelect:
				fallthrough
			case check.PostgreSQLShow:
				postgreSQLChecks[c.Name] = c

			case check.MongoDBGetParameter:
				fallthrough
			case check.MongoDBBuildInfo:
				fallthrough
			case check.MongoDBGetCmdLineOpts:
				fallthrough
			case check.MongoDBReplSetGetStatus:
				fallthrough
			case check.MongoDBGetDiagnosticData:
				mongoDBChecks[c.Name] = c

			default:
				l.Warnf("Unknown check type %s, skip it.", c.Type)
			}
		case 2:
			switch c.Family {
			case check.MySQL:
				mySQLChecks[c.Name] = c
			case check.PostgreSQL:
				postgreSQLChecks[c.Name] = c
			case check.MongoDB:
				mongoDBChecks[c.Name] = c
			default:
				l.Warnf("Unknown check family %s, skip it.", c.Family)
			}
		}
	}

	return mySQLChecks, postgreSQLChecks, mongoDBChecks
}
