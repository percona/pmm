package migrations

import (
	"bytes"
	"embed"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"text/template"

	"github.com/golang-migrate/migrate/v4"
)

//go:embed templates/*.sql
var eFS embed.FS

func GenerateTestSetupMigrations(data map[string]map[string]any, path string) error {
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

func renderMigrations(data map[string]map[string]any) ([]memMigration, error) {
	entries, err := fs.ReadDir(eFS, "templates")
	if err != nil {
		return nil, err
	}

	var migrations []memMigration
	for i, entry := range entries {
		if entry.IsDir() {
			continue
		}

		content, err := eFS.ReadFile("templates/" + entry.Name())
		if err != nil {
			return nil, err
		}

		migration := memMigration{
			Version:    uint(i + 1),
			Identifier: entry.Name(),
		}
		if _, ok := data[entry.Name()]; !ok {
			migration.Up = string(content)
			migrations = append(migrations, migration)
			continue
		}

		var buf bytes.Buffer
		tmpl, err := template.New(entry.Name()).Parse(string(content))
		if err != nil {
			return nil, err
		}
		if err := tmpl.Execute(&buf, data[entry.Name()]); err != nil {
			return nil, err
		}
		migration.Up = buf.String()
		migrations = append(migrations, migration)
	}

	return migrations, nil
}

func Run(dsn string, data map[string]map[string]any) error {
	migrations, err := renderMigrations(data)
	if err != nil {
		return err
	}
	src := newMemMigrations(migrations)
	m, err := migrate.NewWithSourceInstance("memMigrations", src, dsn)
	if err != nil {
		return err
	}

	err = m.Up()
	if errors.Is(err, migrate.ErrNoChange) {
		return nil
	}

	return err
}
