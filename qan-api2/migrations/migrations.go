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

package migrations

import (
	"embed"
	"errors"
	"fmt"
	"io"
	"net/url"

	"github.com/golang-migrate/migrate/v4"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/qan-api2/utils/templatefs"
)

const (
	metricsEngineSimple           = "MergeTree"
	metricsEngineCluster          = "ReplicatedMergeTree('/clickhouse/tables/{shard}/{database}/metrics', '{replica}')"
	schemaMigrationsEngineCluster = "ReplicatedMergeTree('/clickhouse/tables/{shard}/{database}/schema_migrations', '{replica}') ORDER BY version"
)

//go:embed sql/*.sql
var eFS embed.FS

func IsClickhouseClusterReady(dsn string, clusterName string) (bool, error) {
	var args []any
	sql := "SELECT sum(is_local = 0) AS remote_hosts FROM system.clusters"
	if clusterName != "" {
		sql = fmt.Sprintf("%s WHERE cluster = ?", sql)
		args = append(args, clusterName)
	}

	db, err := sqlx.Connect("clickhouse", dsn)
	if err != nil {
		return false, err
	}
	defer db.Close() //nolint:errcheck

	logrus.Infof("Executing query: %s; args: %v", sql, args)
	rows, err := db.Queryx(fmt.Sprintf("%s;", sql), args...)
	if err != nil {
		return false, err
	}
	defer rows.Close() //nolint:errcheck

	if rows.Next() {
		var remoteHosts int
		if err := rows.Scan(&remoteHosts); err != nil {
			return false, err
		}

		return remoteHosts > 0, nil
	}

	return false, nil
}

// Force schema_migrations table cluster engine, optionally cluster name in DSN.
func addClusterSchemaMigrationsParams(u *url.URL, clusterName string) (*url.URL, error) {
	// Values x-cluster-name and x-migrations-table-engine goes as part of a query.
	// Since only x-migrations-table-engine contains special chars, it needs not to be escaped.
	q := u.Query()
	if clusterName != "" {
		logrus.Infof("Using ClickHouse cluster name: %s", clusterName)
		q.Set("x-cluster-name", clusterName)
	}

	encoded := q.Encode()
	if encoded != "" {
		u.RawQuery = encoded + "&"
	}

	u.RawQuery += "x-migrations-table-engine=" + schemaMigrationsEngineCluster
	logrus.Debugf("ClickHouse cluster detected, setting schema_migrations table engine to: %s", schemaMigrationsEngineCluster)

	return u, nil
}

func GetEngine(dsn string, clusterName string) string {
	isCluster, err := IsClickhouseClusterReady(dsn, clusterName)
	if err != nil {
		logrus.Fatalf("Error checking ClickHouse cluster status: %v", err)
	}
	if isCluster {
		return metricsEngineCluster
	}

	return metricsEngineSimple
}

func Run(dsn string, templateData map[string]any, isCluster bool, clusterName string) error {
	// Use TemplateFS as the migration source for golang-migrate
	tfs := templatefs.NewTemplateFS(eFS, templateData)
	drv, err := templatefs.NewDriver(tfs, "sql")
	if err != nil {
		return err
	}

	isClusterReady, err := IsClickhouseClusterReady(dsn, clusterName)
	if err != nil {
		return err
	}

	var u *url.URL
	if isClusterReady {
		u, err = url.Parse(dsn)
		if err != nil {
			return fmt.Errorf("could not parse DSN: %w", err)
		}
		logrus.Infof("ClickHouse cluster detected, adjusting DSN for migrations, original dsn: %s", u.Redacted())
		u, err = addClusterSchemaMigrationsParams(u, clusterName)
		if err != nil {
			return err
		}

		logrus.Infof("Adjusted DSN for migrations: %s", u.Redacted())
	}

	m, err := migrate.NewWithSourceInstance("templatefs", drv, u.String())
	if err != nil {
		return err
	}

	err = m.Up()
	if err != nil {
		if errors.Is(err, migrate.ErrNoChange) || errors.Is(err, io.EOF) {
			return nil
		}
		logrus.Errorf("[Run] Migration failed: %v", err)
	}

	return err
}
