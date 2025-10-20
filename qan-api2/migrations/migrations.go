package migrations

import (
	"embed"
	"errors"
	"fmt"
	"net/url"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/qan-api2/utils/templatefs"
)

const (
	metricsEngineSimple           = "MergeTree"
	metricsEngineCluster          = "ReplicatedMergeTree"
	schemaMigrationsEngineCluster = "ReplicatedMergeTree"
)

//go:embed sql/*.sql
var migrationFS embed.FS

func IsClickhouseCluster(dsn string, clusterName string) (bool, error) {
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
func addClusterSchemaMigrationsParams(dsn string, clusterName string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}

	q := u.Query()
	if clusterName != "" {
		logrus.Infof("Using ClickHouse cluster name: %s", clusterName)
		q.Set("x-cluster-name", clusterName)
	}
	q.Set("x-migrations-table-engine", schemaMigrationsEngineCluster)

	u.RawQuery = q.Encode()
	logrus.Debugf("ClickHouse cluster detected, setting schema_migrations table engine to: %s", schemaMigrationsEngineCluster)

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
		u, err := url.Parse(dsn)
		if err != nil {
			return fmt.Errorf("could not parse DSN: %w", err)
		}
		logrus.Infof("ClickHouse cluster detected, adjusting DSN for migrations; original DSN: %s", u.Redacted())
		dsn, err := addClusterSchemaMigrationsParams(dsn, clusterName)
		if err != nil {
			return err
		}
		u, err = url.Parse(dsn)
		if err != nil {
			return fmt.Errorf("could not parse DSN: %w", err)
		}
		logrus.Infof("Adjusted DSN for migrations: %s", u.Redacted())
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
