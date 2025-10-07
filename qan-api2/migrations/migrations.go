package migrations

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/golang-migrate/migrate/v4"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

const (
	metricsEngineSimple           = "MergeTree"
	metricsEngineCluster          = "ReplicatedMergeTree('/clickhouse/tables/{shard}/metrics', '{replica}')"
	schemaMigrationsEngineCluster = "ReplicatedMergeTree('/clickhouse/tables/{shard}/schema_migrations', '{replica}') ORDER BY version"
)

//go:embed templates/*.sql
var eFS embed.FS

func renderMigrations(data map[string]map[string]any) ([]memMigration, error) {
	entries, err := fs.ReadDir(eFS, "templates")
	if err != nil {
		return nil, err
	}

	var migrations []memMigration
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".up.sql") {
			continue
		}
		content, err := eFS.ReadFile("templates/" + name)
		if err != nil {
			return nil, err
		}
		parts := strings.SplitN(name, "_", 2)
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid migration filename: %s", name)
		}
		upSQL := string(content)
		if tmpl, err := template.New(name).Parse(upSQL); err == nil {
			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, data[name]); err == nil {
				upSQL = buf.String()
			}
		}
		downSQL := ""
		downName := strings.Replace(name, ".up.sql", ".down.sql", 1)
		if downContent, err := eFS.ReadFile("templates/" + downName); err == nil {
			downSQL = string(downContent)
		}
		migrations = append(migrations, memMigration{
			Identifier: name,
			Up:         upSQL,
			Down:       downSQL,
		})
	}

	return migrations, nil
}

func IsClickhouseCluster(dsn string) (bool, error) {
	db, err := sqlx.Connect("clickhouse", dsn)
	if err != nil {
		return false, err
	}
	defer db.Close() //nolint:errcheck

	rows, err := db.Queryx("SELECT sum(is_local = 0) AS remote_hosts FROM system.clusters;")
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

// Force schema_migrations table engine, optionaly cluster name in DSN.
func addSchemaMigrationsParams(dsn string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}

	logrus.Debugf("ClickHouse cluster detected, setting schema_migrations table engine to: %s", schemaMigrationsEngineCluster)

	q := u.Query()

	// If PMM_CLICKHOUSE_CLUSTER_NAME is set, its value will be added to the DSN as x-cluster-name to ensure migrations target the specified ClickHouse cluster.
	clusterName := os.Getenv("PMM_CLICKHOUSE_CLUSTER_NAME")
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
	isCluster, err := IsClickhouseCluster(dsn)
	if err != nil {
		logrus.Fatalf("Error checking ClickHouse cluster status: %v", err)
	}
	if isCluster {
		return metricsEngineCluster
	}

	return metricsEngineSimple
}

func GenerateMigrations(data map[string]map[string]any, path string) error {
	migrations, err := renderMigrations(data)
	if err != nil {
		return err
	}

	for _, migration := range migrations {
		err = os.WriteFile(filepath.Join(path, migration.Identifier), []byte(migration.Up), 0o644)
		if err != nil {
			return err
		}
	}

	return nil
}

func Run(dsn string, data map[string]map[string]any) error {
	migrations, err := renderMigrations(data)
	if err != nil {
		return err
	}
	log.Printf("rendered %d migrations", len(migrations))
	for i := 0; i < len(migrations); i++ {
		log.Printf("[Run] Migration loaded: version=%d, query=%s", i+1, migrations[i].Up)
	}

	// Build versions slice from migration filenames
	var versions []uint
	for _, mig := range migrations {
		parts := strings.SplitN(mig.Identifier, "_", 2)
		if len(parts) < 2 {
			continue
		}
		var v uint
		if _, err := fmt.Sscanf(parts[0], "%d", &v); err == nil {
			versions = append(versions, v)
		}
		logrus.Debugf("[Run] Migration loaded: version=%d, identifier=%s", v, mig.Identifier)
	}
	src := newMemMigrations(migrations, versions)

	isCluster, err := IsClickhouseCluster(dsn)
	if err != nil {
		return err
	}
	if isCluster {
		log.Printf("ClickHouse cluster detected, adjusting DSN for migrations, original dsn: %s", dsn)
		dsn, err = addSchemaMigrationsParams(dsn)
		if err != nil {
			return err
		}
		log.Printf("Adjusted DSN for migrations: %s", dsn)
	}

	m, err := migrate.NewWithSourceInstance("memMigrations", src, dsn)
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
