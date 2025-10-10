package migrations

import (
	"embed"
	"errors"
	"fmt"
	"log"
	"net/url"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/qan-api2/utils/templatefs"
)

const (
	metricsEngineSimple           = "MergeTree"
	metricsEngineCluster          = "ReplicatedMergeTree('/clickhouse/tables/{shard}/metrics', '{replica}')"
	schemaMigrationsEngineCluster = "ReplicatedMergeTree('/clickhouse/tables/{shard}/schema_migrations', '{replica}') ORDER BY version"
)

//go:embed sql/*.sql
var migrationFS embed.FS

func IsClickhouseCluster(dsn string, clusterName string) (bool, error) {
	var args []interface{}
	sql := "SELECT sum(is_local = 0) AS remote_hosts FROM system.clusters"
	if clusterName != "" {
		sql = fmt.Sprintf("%s WHERE cluster = ?", sql)
		args = append(args, clusterName)
	}

	log.Printf("Executing query: %s; args: %v", sql, args)

	db, err := sqlx.Connect("clickhouse", dsn)
	if err != nil {
		return false, err
	}
	defer db.Close() //nolint:errcheck

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

	logrus.Debugf("ClickHouse cluster detected, setting schema_migrations table engine to: %s", schemaMigrationsEngineCluster)

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

	return u.String(), nil
}

func GetEngine(dsn string) string {
	isCluster, err := IsClickhouseCluster(dsn, "")
	if err != nil {
		logrus.Fatalf("Error checking ClickHouse cluster status: %v", err)
	}
	if isCluster {
		return metricsEngineCluster
	}

	return metricsEngineSimple
}

func Run(dsn string, templateData map[string]any, isCluster bool, clusterName string) error {
	if isCluster {
		log.Printf("ClickHouse cluster detected, adjusting DSN for migrations, original dsn: %s", dsn)
		dsn, err := addClusterSchemaMigrationsParams(dsn, clusterName)
		if err != nil {
			return err
		}
		log.Printf("Adjusted DSN for migrations: %s", dsn)
	}

	// Prepare TemplateFS with provided template data
	tfs := templatefs.NewTemplateFS(migrationFS, templateData)

	// Use TemplateFS directly with golang-migrate
	d, err := iofs.New(tfs, "sql")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithSourceInstance("iofs", d, dsn)
	if err != nil {
		return err
	}

	// run up to the latest migration
	err = m.Up()
	if errors.Is(err, migrate.ErrNoChange) {
		return nil
	}
	return err
}
