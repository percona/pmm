// Copyright (C) 2023 Percona LLC
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
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/gomodule/redigo/redis"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/prometheus/common/expfmt"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/percona/pmm/agent/config"
	"github.com/percona/pmm/agent/tlshelpers"
	"github.com/percona/pmm/agent/utils/mongo_fix"
	"github.com/percona/pmm/agent/utils/templates"
	agent_version "github.com/percona/pmm/agent/utils/version"
	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/version"
)

// configGetter allows for getting a config.
type configGetter interface {
	Get() *config.Config
}

// ConnectionChecker is a struct to check connection to services.
type ConnectionChecker struct {
	l   *logrus.Entry
	cfg configGetter
}

// New creates new ConnectionChecker.
func New(cfg configGetter) *ConnectionChecker {
	return &ConnectionChecker{
		l:   logrus.WithField("component", "connectionchecker"),
		cfg: cfg,
	}
}

// Check checks connection to a service. It returns context cancelation/timeout or driver errors as is.
func (cc *ConnectionChecker) Check(ctx context.Context, msg *agentv1.CheckConnectionRequest, id uint32) *agentv1.CheckConnectionResponse {
	timeout := msg.Timeout.AsDuration()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	switch msg.Type {
	case inventoryv1.ServiceType_SERVICE_TYPE_MYSQL_SERVICE:
		return cc.checkMySQLConnection(ctx, msg.Dsn, msg.TextFiles, msg.TlsSkipVerify, id)
	case inventoryv1.ServiceType_SERVICE_TYPE_MONGODB_SERVICE:
		return cc.checkMongoDBConnection(ctx, msg.Dsn, msg.TextFiles, id)
	case inventoryv1.ServiceType_SERVICE_TYPE_POSTGRESQL_SERVICE:
		return cc.checkPostgreSQLConnection(ctx, msg.Dsn, msg.TextFiles, id)
	case inventoryv1.ServiceType_SERVICE_TYPE_VALKEY_SERVICE:
		return cc.checkValkeyConnection(
			ctx,
			msg.Dsn,
			msg.Tls,
			msg.TextFiles,
			msg.TlsSkipVerify,
			id)
	case inventoryv1.ServiceType_SERVICE_TYPE_PROXYSQL_SERVICE:
		return cc.checkProxySQLConnection(ctx, msg.Dsn)
	case inventoryv1.ServiceType_SERVICE_TYPE_EXTERNAL_SERVICE, inventoryv1.ServiceType_SERVICE_TYPE_HAPROXY_SERVICE:
		return cc.checkExternalConnection(ctx, msg.Dsn, msg.TlsSkipVerify)
	default:
		panic(fmt.Sprintf("unknown service type: %v", msg.Type))
	}
}

func (cc *ConnectionChecker) sqlPing(ctx context.Context, db *sql.DB) error {
	// use both query tag and SELECT value to cover both comments and values stripping by the server
	var dest string
	err := db.QueryRowContext(ctx, `SELECT /* agent='connectionchecker' */ 'pmm-agent'`).Scan(&dest)
	cc.l.Debugf("sqlPing: %v", err)
	return err
}

func (cc *ConnectionChecker) checkMySQLConnection(ctx context.Context, dsn string, files *agentv1.TextFiles, tlsSkipVerify bool, id uint32) *agentv1.CheckConnectionResponse { //nolint:lll
	var res agentv1.CheckConnectionResponse
	var err error

	if files != nil {
		err = tlshelpers.RegisterMySQLCerts(files.Files, tlsSkipVerify)
		if err != nil {
			cc.l.Debugf("checkMySQLConnection: failed to register cert: %s", err)
			res.Error = err.Error()
			return &res
		}
	}

	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		cc.l.Debugf("checkMySQLConnection: failed to parse DSN: %s", err)
		res.Error = err.Error()
		return &res
	}

	tempdir := filepath.Join(cc.cfg.Get().Paths.TempDir, strings.ToLower("check-mysql-connection"), strconv.Itoa(int(id)))
	_, err = templates.RenderDSN(dsn, files, tempdir)
	defer templates.CleanupTempDir(tempdir, cc.l)
	if err != nil {
		cc.l.Debugf("checkMySQLDBConnection: failed to Render DSN: %s", err)
		res.Error = err.Error()
		return &res
	}

	connector, err := mysql.NewConnector(cfg)
	if err != nil {
		cc.l.Debugf("checkMySQLConnection: failed to create connector: %s", err)
		res.Error = err.Error()
		return &res
	}

	db := sql.OpenDB(connector)
	defer db.Close() //nolint:errcheck

	if err = cc.sqlPing(ctx, db); err != nil {
		if errors.As(err, &x509.HostnameError{}) {
			res.Error = errors.Wrap(err,
				"mysql ssl certificate is misconfigured, make sure the certificate includes the requested hostname/IP in CN or subjectAltName fields").Error()
		} else {
			res.Error = err.Error()
		}
	}

	return &res
}

func (cc *ConnectionChecker) checkMongoDBConnection(ctx context.Context, dsn string, files *agentv1.TextFiles, id uint32) *agentv1.CheckConnectionResponse {
	const helloCommandVersion = "4.2.10"

	var res agentv1.CheckConnectionResponse
	var err error

	tempdir := filepath.Join(cc.cfg.Get().Paths.TempDir, strings.ToLower("check-mongodb-connection"), strconv.Itoa(int(id)))
	dsn, err = templates.RenderDSN(dsn, files, tempdir)
	defer templates.CleanupTempDir(tempdir, cc.l)
	if err != nil {
		cc.l.Debugf("checkMongoDBConnection: failed to Render DSN: %s", err)
		res.Error = err.Error()
		return &res
	}

	opts, err := mongo_fix.ClientOptionsForDSN(dsn)
	if err != nil {
		cc.l.Debugf("failed to parse DSN: %s", err)
		res.Error = err.Error()
		return &res
	}

	client, err := mongo.Connect(ctx, opts)
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

	mongoVersion, err := agent_version.GetMongoDBVersion(ctx, client)
	if err != nil {
		cc.l.Debugf("checkMongoDBConnection: failed to get MongoDB version: %s", err)
		res.Error = err.Error()
		return &res
	}

	serverInfo := struct {
		ArbiterOnly bool `bson:"arbiterOnly"`
	}{}

	// use hello command for newer MongoDB versions
	command := "hello"
	if mongoVersion.Less(version.MustParse(helloCommandVersion)) {
		command = "isMaster"
	}

	err = client.Database("admin").RunCommand(ctx, bson.D{{Key: command, Value: 1}}).Decode(&serverInfo)
	if err != nil {
		cc.l.Debugf("checkMongoDBConnection: failed to runCommand %s: %s", command, err)
		res.Error = err.Error()
		return &res
	}

	if !serverInfo.ArbiterOnly {
		resp := client.Database("admin").RunCommand(ctx, bson.D{{Key: "getDiagnosticData", Value: 1}})
		if err = resp.Err(); err != nil {
			cc.l.Debugf("checkMongoDBConnection: failed to runCommand getDiagnosticData: %s", err)
			res.Error = err.Error()
			return &res
		}
	}

	return &res
}

func (cc *ConnectionChecker) checkPostgreSQLConnection(ctx context.Context, dsn string, files *agentv1.TextFiles, id uint32) *agentv1.CheckConnectionResponse {
	var res agentv1.CheckConnectionResponse
	var err error

	tempdir := filepath.Join(cc.cfg.Get().Paths.TempDir, strings.ToLower("check-postgresql-connection"), strconv.Itoa(int(id)))
	dsn, err = templates.RenderDSN(dsn, files, tempdir)
	defer templates.CleanupTempDir(tempdir, cc.l)
	if err != nil {
		cc.l.Debugf("checkPostgreSQLConnection: failed to Render DSN: %s", err)
		res.Error = err.Error()
		return &res
	}

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

func (cc *ConnectionChecker) checkValkeyConnection(
	ctx context.Context,
	dsn string,
	tls bool,
	files *agentv1.TextFiles,
	tlsSkipVerify bool,
	id uint32,
) *agentv1.CheckConnectionResponse {
	var res agentv1.CheckConnectionResponse
	var err error

	tempdir := filepath.Join(cc.cfg.Get().Paths.TempDir, "check-valkey-connection", strconv.Itoa(int(id)))
	dsn, err = templates.RenderDSN(dsn, files, tempdir)
	defer templates.CleanupTempDir(tempdir, cc.l)
	if err != nil {
		cc.l.Debugf("checkValkeyConnection: failed to Render DSN: %s", err)
		res.Error = err.Error()
		return &res
	}

	opts, err := tlshelpers.GetValkeyTLSConfig(files, tls, tlsSkipVerify)
	if err != nil {
		cc.l.Debugf("checkValkeyConnection: failed to get TLS config: %s", err)
		res.Error = err.Error()
		return &res
	}
	c, err := redis.DialURLContext(ctx, dsn, opts...)
	if err != nil {
		res.Error = err.Error()
		return &res
	}

	defer c.Close() //nolint:errcheck
	return &res
}

func (cc *ConnectionChecker) checkProxySQLConnection(ctx context.Context, dsn string) *agentv1.CheckConnectionResponse {
	var res agentv1.CheckConnectionResponse

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

func (cc *ConnectionChecker) checkExternalConnection(ctx context.Context, uri string, tlsSkipVerify bool) *agentv1.CheckConnectionResponse {
	var res agentv1.CheckConnectionResponse

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		res.Error = err.Error()
		return &res
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: tlsSkipVerify, //nolint:gosec // allow this for self-signed certs
		},
	}
	client := &http.Client{Transport: tr}

	resp, err := client.Do(req)
	if err != nil {
		res.Error = err.Error()
		return &res
	}
	defer resp.Body.Close() //nolint:gosec,errcheck,nolintlint

	if resp.StatusCode != http.StatusOK {
		res.Error = fmt.Sprintf("Unexpected HTTP status code: %d. Expected: 200", resp.StatusCode)
		return &res
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		res.Error = fmt.Sprintf("Cannot read body of exporter's response: %v", err)
		return &res
	}

	var parser expfmt.TextParser
	_, err = parser.TextToMetricFamilies(strings.NewReader(string(body)))
	if err != nil {
		res.Error = fmt.Sprintf("Unexpected exporter's response format: %v", err)
		return &res
	}

	return &res
}
