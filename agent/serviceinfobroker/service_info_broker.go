// Copyright 2023 Percona LLC
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

// Package serviceinfobroker helps extract various information from databases.
package serviceinfobroker

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/percona/pmm/agent/config"
	"github.com/percona/pmm/agent/tlshelpers"
	"github.com/percona/pmm/agent/utils/mongo_fix"
	"github.com/percona/pmm/agent/utils/templates"
	"github.com/percona/pmm/agent/utils/version"
	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
)

// configGetter allows to get a config.
type configGetter interface {
	Get() *config.Config
}

// ServiceInfoBroker helps query various information from services.
type ServiceInfoBroker struct {
	l   *logrus.Entry
	cfg configGetter
}

// New creates a new ServiceInfoBroker.
func New(cfg configGetter) *ServiceInfoBroker {
	return &ServiceInfoBroker{
		l:   logrus.WithField("component", "serviceinfobroker"),
		cfg: cfg,
	}
}

// GetInfoFromService gathers information from a service. It returns context cancelation/timeout or driver errors as is.
func (sib *ServiceInfoBroker) GetInfoFromService(ctx context.Context, msg *agentpb.ServiceInfoRequest, id uint32) *agentpb.ServiceInfoResponse {
	timeout := msg.Timeout.AsDuration()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	switch msg.Type {
	case inventorypb.ServiceType_MYSQL_SERVICE:
		return sib.getMySQLInfo(ctx, msg.Dsn, msg.TextFiles, msg.TlsSkipVerify, id)
	case inventorypb.ServiceType_MONGODB_SERVICE:
		return sib.getMongoDBInfo(ctx, msg.Dsn, msg.TextFiles, id)
	case inventorypb.ServiceType_POSTGRESQL_SERVICE:
		return sib.getPostgreSQLInfo(ctx, msg.Dsn, msg.TextFiles, id)
	case inventorypb.ServiceType_PROXYSQL_SERVICE:
		return sib.getProxySQLInfo(ctx, msg.Dsn)
	// NOTE: these types may be implemented later.
	case inventorypb.ServiceType_EXTERNAL_SERVICE, inventorypb.ServiceType_HAPROXY_SERVICE:
		return &agentpb.ServiceInfoResponse{}
	default:
		panic(fmt.Sprintf("unknown service type: %v", msg.Type))
	}
}

func (sib *ServiceInfoBroker) getMySQLInfo(ctx context.Context, dsn string, files *agentpb.TextFiles, tlsSkipVerify bool, id uint32) *agentpb.ServiceInfoResponse {
	var res agentpb.ServiceInfoResponse
	var err error

	if files != nil {
		err = tlshelpers.RegisterMySQLCerts(files.Files, tlsSkipVerify)
		if err != nil {
			sib.l.Debugf("getMySQLInfo: failed to register cert: %s", err)
			res.Error = err.Error()
			return &res
		}
	}

	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		sib.l.Debugf("getMySQLInfo: failed to parse DSN: %s", err)
		res.Error = err.Error()
		return &res
	}

	tempdir := filepath.Join(sib.cfg.Get().Paths.TempDir, strings.ToLower("get-mysql-info"), strconv.Itoa(int(id)))
	_, err = templates.RenderDSN(dsn, files, tempdir)
	if err != nil {
		sib.l.Debugf("getMySQLInfo: failed to Render DSN: %s", err)
		res.Error = err.Error()
		return &res
	}

	connector, err := mysql.NewConnector(cfg)
	if err != nil {
		sib.l.Debugf("getMySQLInfo: failed to create connector: %s", err)
		res.Error = err.Error()
		return &res
	}

	db := sql.OpenDB(connector)
	defer db.Close() //nolint:errcheck

	var count uint64
	if err = db.QueryRowContext(ctx, "SELECT /* agent='serviceinfobroker' */ COUNT(*) FROM information_schema.tables").Scan(&count); err != nil {
		res.Error = err.Error()
		return &res
	}

	res.TableCount = int32(count)
	if count > math.MaxInt32 {
		res.TableCount = math.MaxInt32
	}

	var version string
	if err = db.QueryRowContext(ctx, "SELECT /* agent='serviceinfobroker' */ VERSION()").Scan(&version); err != nil {
		res.Error = err.Error()
	}

	res.Version = version
	return &res
}

func (sib *ServiceInfoBroker) getMongoDBInfo(ctx context.Context, dsn string, files *agentpb.TextFiles, id uint32) *agentpb.ServiceInfoResponse {
	var res agentpb.ServiceInfoResponse
	var err error

	tempdir := filepath.Join(sib.cfg.Get().Paths.TempDir, strings.ToLower("get-mongodb-info"), strconv.Itoa(int(id)))
	dsn, err = templates.RenderDSN(dsn, files, tempdir)
	if err != nil {
		sib.l.Debugf("getMongoDBInfo: failed to Render DSN: %s", err)
		res.Error = err.Error()
		return &res
	}

	opts, err := mongo_fix.ClientOptionsForDSN(dsn)
	if err != nil {
		sib.l.Debugf("failed to parse DSN: %s", err)
		res.Error = err.Error()
		return &res
	}

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		sib.l.Debugf("getMongoDBInfo: failed to Connect: %s", err)
		res.Error = err.Error()
		return &res
	}
	defer client.Disconnect(ctx) //nolint:errcheck

	if err = client.Ping(ctx, nil); err != nil {
		sib.l.Debugf("getMongoDBInfo: failed to Ping: %s", err)
		res.Error = err.Error()
		return &res
	}

	mongoVersion, err := version.GetMongoDBVersion(ctx, client)
	if err != nil {
		sib.l.Debugf("getMongoDBInfo: failed to get MongoDB version: %s", err)
		res.Error = err.Error()
		return &res
	}

	res.Version = mongoVersion.String()
	return &res
}

func (sib *ServiceInfoBroker) getPostgreSQLInfo(ctx context.Context, dsn string, files *agentpb.TextFiles, id uint32) *agentpb.ServiceInfoResponse {
	var res agentpb.ServiceInfoResponse
	var err error

	tempdir := filepath.Join(sib.cfg.Get().Paths.TempDir, strings.ToLower("get-postgresql-info"), strconv.Itoa(int(id)))
	dsn, err = templates.RenderDSN(dsn, files, tempdir)
	if err != nil {
		sib.l.Debugf("getPostgreSQLInfo: failed to Render DSN: %s", err)
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

	var databaseList []string
	databaseListQuery := "SELECT /* agent='serviceinfobroker' */ datname FROM pg_database WHERE datallowconn = true AND datistemplate = false AND has_database_privilege(current_user, datname, 'connect')" //nolint:lll
	rows, err := db.QueryContext(ctx, databaseListQuery)
	if err != nil {
		res.Error = err.Error()
		return &res
	}
	defer rows.Close() //nolint:errcheck
	for rows.Next() {
		var databaseName string
		err := rows.Scan(&databaseName)
		if err != nil {
			res.Error = err.Error()
			return &res
		}

		databaseList = append(databaseList, databaseName)
	}
	res.DatabaseList = databaseList

	var version string
	if err = db.QueryRowContext(ctx, "SHOW /* agent='serviceinfobroker' */ SERVER_VERSION").Scan(&version); err != nil {
		res.Error = err.Error()
	}
	res.Version = version

	var pgsmVersion string
	err = db.QueryRowContext(ctx, "SELECT /* agent='serviceinfobroker' */ extversion FROM pg_extension WHERE extname = 'pg_stat_monitor';").Scan(&pgsmVersion)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		res.Error = err.Error()
	}
	res.PgsmVersion = &pgsmVersion

	return &res
}

func (sib *ServiceInfoBroker) getProxySQLInfo(ctx context.Context, dsn string) *agentpb.ServiceInfoResponse {
	var res agentpb.ServiceInfoResponse

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

	var version string
	if err := db.QueryRowContext(ctx, "SELECT /* agent='serviceinfobroker' */ @@GLOBAL.'admin-version'").Scan(&version); err != nil {
		res.Error = err.Error()
	}

	res.Version = version
	return &res
}
