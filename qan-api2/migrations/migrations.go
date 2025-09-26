package migrations

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
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
		var version uint
		parts := strings.SplitN(name, "_", 2)
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid migration filename: %s", name)
		}
		n, err := fmt.Sscanf(parts[0], "%d", &version)
		if n != 1 || err != nil {
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
			Version:    version,
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
	for _, mig := range migrations {
		logrus.Debugf("[Run] Migration loaded: version=%d, identifier=%s\n", mig.Version, mig.Identifier)
	}
	src := newMemMigrations(migrations)
	m, err := migrate.NewWithSourceInstance("memMigrations", src, dsn)
	if err != nil {
		return err
	}

	err = m.Up()
	if err != nil {
		if errors.Is(err, migrate.ErrNoChange) || errors.Is(err, io.EOF) {
			return nil
		}
		logrus.Errorf("[Run] Migration failed: %v\n", err)
	}

	return err
}
