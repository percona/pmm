package migrations

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
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
	databaseEngineSimple  = "MergeTree"
	databaseEngineCluster = "ReplicatedMergeTree('/clickhouse/tables/{shard}/metrics', '{replica}')"
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

func isClickhouseCluster(dsn string) bool {
	db, err := sqlx.Connect("clickhouse", dsn)
	if err != nil {
		return false
	}
	defer db.Close() //nolint:errcheck

	rows, err := db.Queryx("SELECT sum(is_local = 0) AS remote_hosts FROM system.clusters;")
	if err != nil {
		return false
	}
	defer rows.Close() //nolint:errcheck

	if rows.Next() {
		var remoteHosts int
		if err := rows.Scan(&remoteHosts); err != nil {
			return false
		}

		return remoteHosts > 0
	}

	return false
}

func GetEngine(dsn string) string {
	if isClickhouseCluster(dsn) {
		return databaseEngineCluster
	}

	return databaseEngineSimple
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

	if isClickhouseCluster(dsn) {
		// Force schema_migrations table engine in DSN
		u, err := url.Parse(dsn)
		if err != nil {
			return err
		}
		q := u.Query()
		q.Set("x-migrations-table-engine", GetEngine(dsn))
		u.RawQuery = q.Encode()
		dsn = u.String()
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
