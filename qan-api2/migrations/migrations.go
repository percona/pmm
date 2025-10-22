package migrations

import (
	"embed"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"

	"github.com/golang-migrate/migrate/v4"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/qan-api2/utils/templatefs"
	"github.com/percona/pmm/utils/dsnutils"
)

const (
	metricsEngineSimple           = "MergeTree"
	metricsEngineCluster          = "ReplicatedMergeTree('/clickhouse/tables/{shard}/metrics', '{replica}')"
	schemaMigrationsEngineCluster = "ReplicatedMergeTree('/clickhouse/tables/{shard}/schema_migrations', '{replica}') ORDER BY version"
)

//go:embed sql/*.sql
var eFS embed.FS

func IsClickhouseClusterReady(dsn string, clusterName string) (bool, error) {
	var args []interface{}
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

	log.Printf("Executing query: %s; args: %v", sql, args)
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
func addClusterSchemaMigrationsParams(dsn string, clusterName string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}

	// Values x-cluster-name and x-migrations-table-engine goes as part of query.
	// Since only x-migrations-table-engine contains special chars only this one is needed not to be escaped.
	q := u.Query()
	if clusterName != "" {
		logrus.Printf("Using ClickHouse cluster name: %s", clusterName)
		q.Set("x-cluster-name", clusterName)
	}

	encoded := q.Encode()
	if encoded != "" {
		u.RawQuery = encoded + "&x-migrations-table-engine=" + schemaMigrationsEngineCluster
	} else {
		u.RawQuery = "x-migrations-table-engine=" + schemaMigrationsEngineCluster
	}
	logrus.Debugf("ClickHouse cluster detected, setting schema_migrations table engine to: %s", schemaMigrationsEngineCluster)

	return u.String(), nil
}

func GetEngine(isCluster bool) string {
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

	if isCluster {
		isClusterReady, err := IsClickhouseClusterReady(dsn, clusterName)
		if err != nil {
			return err
		}
		if isClusterReady {
			log.Printf("ClickHouse cluster detected, adjusting DSN for migrations, original dsn: %s", dsnutils.RedactDSN(dsn))
			dsn, err = addClusterSchemaMigrationsParams(dsn, clusterName)
			if err != nil {
				return err
			}
			log.Printf("Adjusted DSN for migrations: %s", dsnutils.RedactDSN(dsn))
		}
	}

	m, err := migrate.NewWithSourceInstance("templatefs", drv, dsn)
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
