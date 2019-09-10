// pmm-agent
// Copyright 2019 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package connectionchecker provides database connection checkers.
package connectionchecker

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql" // register SQL driver
	"github.com/golang/protobuf/ptypes"
	"github.com/lib/pq"
	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ConnectionChecker is a struct to check connection to services.
type ConnectionChecker struct {
	ctx context.Context
}

// New creates new ConnectionChecker.
func New(ctx context.Context) *ConnectionChecker {
	return &ConnectionChecker{
		ctx: ctx,
	}
}

// Check checks connection to a service. It returns context cancelation/timeout or driver errors as is.
func (c *ConnectionChecker) Check(msg *agentpb.CheckConnectionRequest) error {
	timeout, _ := ptypes.Duration(msg.Timeout)
	if timeout == 0 {
		timeout = 3 * time.Second
	}

	ctx, cancel := context.WithTimeout(c.ctx, timeout)
	defer cancel()

	switch msg.Type {
	case inventorypb.ServiceType_MYSQL_SERVICE, inventorypb.ServiceType_PROXYSQL_SERVICE:
		// TODO Use sql.OpenDB with ctx when https://github.com/go-sql-driver/mysql/issues/671 is released
		// (likely in version 1.5.0).

		db, err := sql.Open("mysql", msg.Dsn)
		if err != nil {
			return err
		}
		return checkSQLConnection(ctx, db)

	case inventorypb.ServiceType_POSTGRESQL_SERVICE:
		c, err := pq.NewConnector(msg.Dsn)
		if err != nil {
			return err
		}
		db := sql.OpenDB(c)
		return checkSQLConnection(ctx, db)

	case inventorypb.ServiceType_MONGODB_SERVICE:
		return checkMongoDBConnection(ctx, msg.Dsn)

	default:
		panic(fmt.Sprintf("unhandled service type: %v", msg.Type))
	}
}

func checkMongoDBConnection(ctx context.Context, dsn string) error {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(dsn))
	if err != nil {
		return err
	}

	defer client.Disconnect(ctx) //nolint:errcheck

	return client.Ping(ctx, nil)
}

func checkSQLConnection(ctx context.Context, db *sql.DB) error {
	defer db.Close() //nolint:errcheck

	// use both query tag and SELECT value to cover both comments and values stripping by the server
	var res string
	return db.QueryRowContext(ctx, `SELECT /* pmm-agent:connectionchecker */ 'pmm-agent'`).Scan(&res)
}
