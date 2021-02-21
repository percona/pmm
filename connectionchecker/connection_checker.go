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
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/golang/protobuf/ptypes"
	"github.com/lib/pq"
	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/percona/pmm-agent/config"
	"github.com/percona/pmm-agent/utils/templates"
)

// ConnectionChecker is a struct to check connection to services.
type ConnectionChecker struct {
	l     *logrus.Entry
	paths *config.Paths
}

// New creates new ConnectionChecker.
func New(paths *config.Paths) *ConnectionChecker {
	return &ConnectionChecker{
		l:     logrus.WithField("component", "connectionchecker"),
		paths: paths,
	}
}

// Check checks connection to a service. It returns context cancelation/timeout or driver errors as is.
func (cc *ConnectionChecker) Check(ctx context.Context, msg *agentpb.CheckConnectionRequest, id uint32) *agentpb.CheckConnectionResponse {
	timeout, _ := ptypes.Duration(msg.Timeout)
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	switch msg.Type {
	case inventorypb.ServiceType_MYSQL_SERVICE:
		return cc.checkMySQLConnection(ctx, msg.Dsn)
	case inventorypb.ServiceType_MONGODB_SERVICE:
		return cc.checkMongoDBConnection(ctx, msg.Dsn, msg.TextFiles, id)
	case inventorypb.ServiceType_POSTGRESQL_SERVICE:
		return cc.checkPostgreSQLConnection(ctx, msg.Dsn)
	case inventorypb.ServiceType_PROXYSQL_SERVICE:
		return cc.checkProxySQLConnection(ctx, msg.Dsn)
	case inventorypb.ServiceType_EXTERNAL_SERVICE, inventorypb.ServiceType_HAPROXY_SERVICE:
		return cc.checkExternalConnection(ctx, msg.Dsn)
	default:
		panic(fmt.Sprintf("unhandled service type: %v", msg.Type))
	}
}

func (cc *ConnectionChecker) sqlPing(ctx context.Context, db *sql.DB) error {
	// use both query tag and SELECT value to cover both comments and values stripping by the server
	var dest string
	err := db.QueryRowContext(ctx, `SELECT /* pmm-agent:connectionchecker */ 'pmm-agent'`).Scan(&dest)
	cc.l.Debugf("sqlPing: %v", err)
	return err
}

func (cc *ConnectionChecker) checkMySQLConnection(ctx context.Context, dsn string) *agentpb.CheckConnectionResponse {
	var res agentpb.CheckConnectionResponse

	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		res.Error = err.Error()
		return &res
	}

	connector, err := mysql.NewConnector(cfg)
	if err != nil {
		res.Error = err.Error()
		return &res
	}

	db := sql.OpenDB(connector)
	defer db.Close() //nolint:errcheck

	if err = cc.sqlPing(ctx, db); err != nil {
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

func (cc *ConnectionChecker) checkMongoDBConnection(ctx context.Context, dsn string, files *agentpb.TextFiles, id uint32) *agentpb.CheckConnectionResponse {
	var res agentpb.CheckConnectionResponse
	var err error

	tempdir := filepath.Join(cc.paths.TempDir, strings.ToLower("check-mongodb-connection"), strconv.Itoa(int(id)))
	dsn, err = templates.RenderDSN(dsn, files, tempdir)
	if err != nil {
		cc.l.Debugf("checkMongoDBConnection: failed to Render DSN: %s", err)
		res.Error = err.Error()
		return &res
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(dsn))
	if err != nil {
		cc.l.Debugf("checkMongoDBConnection: failed to Connect: %s", err)
		res.Error = err.Error()
		return &res
	}
	defer client.Disconnect(ctx) //nolint:errcheck

	if err = client.Ping(ctx, nil); err != nil {
		cc.l.Debugf("checkMongoDBConnection: failed to Ping: %s", err)
		res.Error = err.Error()
		return &res
	}

	resp := client.Database("admin").RunCommand(ctx, bson.D{{Key: "listDatabases", Value: 1}})
	if err = resp.Err(); err != nil {
		cc.l.Debugf("checkMongoDBConnection: failed to runCommand listDatabases: %s", err)
		res.Error = err.Error()
		return &res
	}

	return &res
}

func (cc *ConnectionChecker) checkPostgreSQLConnection(ctx context.Context, dsn string) *agentpb.CheckConnectionResponse {
	var res agentpb.CheckConnectionResponse

	c, err := pq.NewConnector(dsn)
	if err != nil {
		res.Error = err.Error()
		return &res
	}
	db := sql.OpenDB(c)
	defer db.Close() //nolint:errcheck

	if err = cc.sqlPing(ctx, db); err != nil {
		res.Error = err.Error()
	}

	return &res
}

func (cc *ConnectionChecker) checkProxySQLConnection(ctx context.Context, dsn string) *agentpb.CheckConnectionResponse {
	var res agentpb.CheckConnectionResponse

	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		res.Error = err.Error()
		return &res
	}

	connector, err := mysql.NewConnector(cfg)
	if err != nil {
		res.Error = err.Error()
		return &res
	}

	db := sql.OpenDB(connector)
	defer db.Close() //nolint:errcheck

	if err = cc.sqlPing(ctx, db); err != nil {
		res.Error = err.Error()
	}

	return &res
}

func (cc *ConnectionChecker) checkExternalConnection(ctx context.Context, uri string) *agentpb.CheckConnectionResponse {
	var res agentpb.CheckConnectionResponse

	req, _ := http.NewRequestWithContext(ctx, "GET", uri, nil)
	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		res.Error = err.Error()
		return &res
	}

	if resp.StatusCode != 200 {
		res.Error = fmt.Sprintf("Unexpected HTTP status code: %d. Expected: 200", resp.StatusCode)
		return &res
	}

	return &res
}
