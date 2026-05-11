package migrations

import (
	"embed"
	"errors"
	"fmt"
	"io"
	"net/url"

	"github.com/golang-migrate/migrate/v4"
	bindata "github.com/golang-migrate/migrate/v4/source/go_bindata"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/qan-api2/utils/templatefs"
	"github.com/percona/pmm/utils/dsnutils"
)

const (
	metricsEngineSimple           = "MergeTree"
	metricsEngineCluster          = "ReplicatedMergeTree"
	schemaMigrationsEngineCluster = "ReplicatedMergeTree ORDER BY version"
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

	logrus.WithField("component", "migrations").Printf("Executing query: %s; args: %v", sql, args)
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

// Force schema_migrations table cluster engine.
func addSchemaMigrationsParams(dsn string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}

	// Values prefixed with "x", such as "x-migrations-table-engine", are part of the query.
	// Since x-migrations-table-engine contains special chars, it must not be escaped.
	q := u.Query()
	q.Set("x-migrations-table-engine", schemaMigrationsEngineCluster)
	u.RawQuery = q.Encode()
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
	l := logrus.WithField("component", "migrations")
	if isCluster {
		isClusterReady, err := IsClickhouseClusterReady(dsn, clusterName)
		if err != nil {
			return err
		}
		if isClusterReady {
			l.Infof("ClickHouse cluster detected, adjusting DSN for migrations, original dsn: %s", dsnutils.RedactDSN(dsn))
			dsn, err = addSchemaMigrationsParams(dsn)
			if err != nil {
				return err
			}
			l.Infof("Adjusted DSN for migrations: %s", dsnutils.RedactDSN(dsn))
		}
	}

	tfs := templatefs.NewTemplateFS(eFS, templateData, "sql")
	names, err := tfs.Names()
	if err != nil {
		return err
	}
	instance, err := bindata.WithInstance(bindata.Resource(names, tfs.ReadFile))
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

	// If the database is in dirty state, try to fix it (PMM-14305)
	if errDirty, ok := errors.AsType[*migrate.ErrDirty](err); ok {
		l.Infof("Migration %d was unsuccessful, trying to fix it...", errDirty.Version)

		ver := errDirty.Version - 1
		if ver == 0 {
			// since 0th migration does not exist, we set it to -1, which means "start from scratch"
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
	}

	return err
}
