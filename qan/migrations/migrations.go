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

// Package migrations applies the qan ClickHouse schema migrations.
package migrations

import (
	"embed"
	"errors"
	"fmt"
	"io"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/clickhouse" // register clickhouse migrate driver (and the database/sql driver)
	bindata "github.com/golang-migrate/migrate/v4/source/go_bindata"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/qan/utils/templatefs"
)

//go:embed sql/*.sql
var eFS embed.FS

// engineData returns the table-engine substitutions for the migration templates.
// Single-node engines; cluster (Replicated*) variants are a follow-up.
func engineData() map[string]any {
	return map[string]any{
		"MergeTree":            "MergeTree",
		"AggregatingMergeTree": "AggregatingMergeTree",
		"SummingMergeTree":     "SummingMergeTree",
		"ReplacingMergeTree":   "ReplacingMergeTree",
	}
}

// Run applies all pending migrations against the given clickhouse:// DSN.
func Run(dsn string) error {
	l := logrus.WithField("component", "migrations")

	tfs := templatefs.NewTemplateFS(eFS, engineData(), "sql")
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

	// If the database is in a dirty state, force back one version and retry (PMM-14305).
	if errDirty, ok := errors.AsType[*migrate.ErrDirty](err); ok {
		l.Infof("Migration %d was unsuccessful, trying to fix it...", errDirty.Version)
		ver := errDirty.Version - 1
		if ver == 0 {
			ver = -1
		}
		err = m.Force(ver)
		if err != nil {
			return fmt.Errorf("can't force the migration %d: %w", ver, err)
		}
		err = m.Up()
		if errors.Is(err, migrate.ErrNoChange) || errors.Is(err, io.EOF) {
			return nil
		}
	}

	return err
}
