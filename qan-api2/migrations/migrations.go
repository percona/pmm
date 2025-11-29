package migrations

import (
	"embed"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"strings"

	clickhouse "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/golang-migrate/migrate/v4"
	bindata "github.com/golang-migrate/migrate/v4/source/go_bindata"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/qan-api2/utils/templatefs"
	"github.com/percona/pmm/utils/dsnutils"
)

const (
	metricsEngineSimple = "MergeTree"
	// Use {uuid} macro for unique ZooKeeper paths as recommended by ClickHouse maintainers.
	// See: https://github.com/ClickHouse/ClickHouse/issues/3288
	// Using {database}/{table} doesn't guarantee uniqueness due to table renames and async DROP.
	metricsEngineCluster          = "ReplicatedMergeTree('/clickhouse/tables/{shard}/{uuid}', '{replica}')"
	schemaMigrationsEngineCluster = "ReplicatedMergeTree('/clickhouse/tables/{shard}/{uuid}', '{replica}') ORDER BY version"

	// ClickHouse error code for TABLE_ALREADY_EXISTS
	tableAlreadyExistsCode = 57
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

	tfs := templatefs.NewTemplateFS(eFS, templateData, "sql")
	names, err := tfs.Names()
	if err != nil {
		return err
	}
	instance, err := bindata.WithInstance(
		bindata.Resource(
			names,
			tfs.ReadFile))
	if err != nil {
		return err
	}

	m, err := migrate.NewWithSourceInstance("go-bindata", instance, dsn)
	if err != nil {
		return err
	}

	err = m.Up()
	if err == nil || errors.Is(err, migrate.ErrNoChange) || errors.Is(err, io.EOF) {
		return nil
	}

	// Handle TABLE_ALREADY_EXISTS error (code 57) which can occur when multiple PMM pods
	// try to run migrations simultaneously in HA mode. This is expected behavior and
	// means another pod already created the table successfully.
	if isTableAlreadyExistsError(err) {
		log.Println("Table already exists (created by another PMM instance), treating as success")
		return nil
	}

	// If the database is in dirty state, try to fix it (PMM-14305)
	var errDirty migrate.ErrDirty
	if errors.As(err, &errDirty) {
		log.Printf("Migration %d was unsuccessful, trying to fix it...", errDirty.Version)

		ver := errDirty.Version - 1
		if ver == 0 {
			// Note: since 0th migration does not exist, we set it to -1, which means "start from scratch"
			ver = -1
		}
		err = m.Force(ver)
		if err != nil {
			return fmt.Errorf("can't force the migration %d: %w", ver, err)
		}

		// try to run migrations again, starting from the forced version
		err = m.Up()
		if errors.Is(err, migrate.ErrNoChange) || errors.Is(err, io.EOF) {
			return nil
		}

		// Check again for TABLE_ALREADY_EXISTS after retry
		if isTableAlreadyExistsError(err) {
			log.Println("Table already exists after retry (created by another PMM instance), treating as success")
			return nil
		}
	}

	return err
}

// isTableAlreadyExistsError checks if the error is a ClickHouse TABLE_ALREADY_EXISTS error (code 57).
// This can happen in HA mode when multiple PMM pods try to run migrations simultaneously.
func isTableAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}

	// Check for ClickHouse exception with code 57
	var chErr *clickhouse.Exception
	if errors.As(err, &chErr) && chErr.Code == tableAlreadyExistsCode {
		return true
	}

	// Also check error message as fallback (the error might be wrapped)
	errStr := err.Error()
	return strings.Contains(errStr, "TABLE_ALREADY_EXISTS") ||
		strings.Contains(errStr, "Code: 57")
}
