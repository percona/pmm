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

package vmalert

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/utils/database"
	"github.com/percona/pmm/managed/utils/testdb"
)

func setupVMAlert(t *testing.T) (*reform.DB, *ExternalRules, *Service) {
	t.Helper()
	check := require.New(t)
	sqlDB := testdb.Open(t, database.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	rules := NewExternalRules()
	svc, err := NewVMAlert(rules, "http://127.0.0.1:8880/")
	check.NoError(err)

	check.NoError(svc.IsReady(context.Background()))

	return db, rules, svc
}

func teardownVMAlert(t *testing.T, rules *ExternalRules, db *reform.DB) {
	t.Helper()
	check := assert.New(t)

	check.NoError(rules.WriteRules(""))
	check.NoError(db.DBInterface().(*sql.DB).Close())
}

func TestVMAlert(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		check := require.New(t)
		db, rules, svc := setupVMAlert(t)
		defer teardownVMAlert(t, rules, db)
		check.NoError(svc.updateConfiguration(context.Background()))
	})

	t.Run("Normal", func(t *testing.T) {
		check := require.New(t)
		db, rules, svc := setupVMAlert(t)
		defer teardownVMAlert(t, rules, db)
		check.NoError(svc.updateConfiguration(context.Background()))
		check.NoError(rules.WriteRules(strings.TrimSpace(`
groups:
  - name: example
    rules:
    - alert: HighRequestLatency
      expr: job:request_latency_seconds:mean5m{job="myjob"} > 0.5
      for: 10m
      labels:
          severity: page
      annotations:
          summary: High request latency
			`)))
		check.NoError(svc.updateConfiguration(context.Background()))
	})
}
