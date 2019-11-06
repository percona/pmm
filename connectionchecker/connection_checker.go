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
	"math"

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
func (c *ConnectionChecker) Check(msg *agentpb.CheckConnectionRequest) *agentpb.CheckConnectionResponse {
	ctx := c.ctx
	timeout, _ := ptypes.Duration(msg.Timeout)
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	switch msg.Type {
	case inventorypb.ServiceType_MYSQL_SERVICE:
		return checkMySQLConnection(ctx, msg.Dsn)
	case inventorypb.ServiceType_MONGODB_SERVICE:
		return checkMongoDBConnection(ctx, msg.Dsn)
	case inventorypb.ServiceType_POSTGRESQL_SERVICE:
		return checkPostgreSQLConnection(ctx, msg.Dsn)
	case inventorypb.ServiceType_PROXYSQL_SERVICE:
		return checkProxySQLConnection(ctx, msg.Dsn)
	default:
		panic(fmt.Sprintf("unhandled service type: %v", msg.Type))
	}
}

func sqlPing(ctx context.Context, db *sql.DB) error {
	// use both query tag and SELECT value to cover both comments and values stripping by the server
	var dest string
	return db.QueryRowContext(ctx, `SELECT /* pmm-agent:connectionchecker */ 'pmm-agent'`).Scan(&dest)
}

func checkMySQLConnection(ctx context.Context, dsn string) *agentpb.CheckConnectionResponse {
	var res agentpb.CheckConnectionResponse

	// TODO Use sql.OpenDB with ctx when https://github.com/go-sql-driver/mysql/issues/671 is released
	// (likely in version 1.5.0).
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		res.Error = err.Error()
		return &res
	}
	defer db.Close() //nolint:errcheck

	if err = sqlPing(ctx, db); err != nil {
		res.Error = err.Error()
		return &res
	}

	var count uint64
	if err = db.QueryRowContext(ctx, "SELECT /* pmm-agent:connectionchecker */ COUNT(*) FROM information_schema.tables").Scan(&count); err != nil {
		res.Error = err.Error()
		return &res
	}

	tableCount := int32(count)
	if count > math.MaxInt32 {
		tableCount = math.MaxInt32
	}

	res.Stats = &agentpb.CheckConnectionResponse_Stats{
		TableCount: tableCount,
	}

	return &res
}

func checkMongoDBConnection(ctx context.Context, dsn string) *agentpb.CheckConnectionResponse {
	var res agentpb.CheckConnectionResponse

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(dsn))
	if err != nil {
		res.Error = err.Error()
		return &res
	}
	defer client.Disconnect(ctx) //nolint:errcheck

	if err = client.Ping(ctx, nil); err != nil {
		res.Error = err.Error()
	}

	return &res
}

func checkPostgreSQLConnection(ctx context.Context, dsn string) *agentpb.CheckConnectionResponse {
	var res agentpb.CheckConnectionResponse

	c, err := pq.NewConnector(dsn)
	if err != nil {
		res.Error = err.Error()
		return &res
	}
	db := sql.OpenDB(c)
	defer db.Close() //nolint:errcheck

	if err = sqlPing(ctx, db); err != nil {
		res.Error = err.Error()
	}

	return &res
}

func checkProxySQLConnection(ctx context.Context, dsn string) *agentpb.CheckConnectionResponse {
	var res agentpb.CheckConnectionResponse

	// TODO Use sql.OpenDB with ctx when https://github.com/go-sql-driver/mysql/issues/671 is released
	// (likely in version 1.5.0).
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		res.Error = err.Error()
		return &res
	}
	defer db.Close() //nolint:errcheck

	if err = sqlPing(ctx, db); err != nil {
		res.Error = err.Error()
	}

	return &res
}
