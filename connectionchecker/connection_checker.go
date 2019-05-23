// pmm-agent
// Copyright (C) 2018 Percona LLC
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

// Package connectionchecker provides database connection checkers.
package connectionchecker

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql" // register SQL driver
	_ "github.com/lib/pq"              // register SQL driver
	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
	"gopkg.in/mgo.v2"
)

// ConnectionChecker is a struct to check connection to services.
type ConnectionChecker struct {
}

// New creates new ConnectionChecker.
func New() *ConnectionChecker {
	return &ConnectionChecker{}
}

// Check checks connection to a service.
func (c *ConnectionChecker) Check(msg *agentpb.CheckConnectionRequest) error {
	switch msg.Type {
	case inventorypb.ServiceType_MYSQL_SERVICE:
		return c.checkSQLConnection("mysql", msg.Dsn)
	case inventorypb.ServiceType_POSTGRESQL_SERVICE:
		return c.checkSQLConnection("postgres", msg.Dsn)
	case inventorypb.ServiceType_MONGODB_SERVICE:
		return c.checkMongoDBConnection(msg.Dsn)
	default:
		panic(fmt.Sprintf("unhandled service type: %v", msg.Type))
	}
}

func (c *ConnectionChecker) checkMongoDBConnection(dsn string) error {
	session, err := mgo.DialWithTimeout(dsn, time.Second) // TODO make timeout configurable
	if err != nil {
		return err
	}
	defer session.Close()

	return session.Ping()
}

func (c *ConnectionChecker) checkSQLConnection(driver string, dsn string) error {
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return err
	}
	defer db.Close() //nolint:errcheck

	var res string
	return db.QueryRow(`SELECT 'pmm-agent'`).Scan(&res)
}
