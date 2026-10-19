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

package otel

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newSchemaMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherFunc(func(expected, actual string) error {
		if !strings.Contains(actual, expected) {
			return fmt.Errorf("%s does not contain %s", actual, expected)
		}
		return nil
	})))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, mock.ExpectationsWereMet())
		_ = db.Close()
	})
	return db, mock
}

func TestEnsureLogsTable(t *testing.T) {
	db, mock := newSchemaMock(t)
	column := "PatternHash UInt64 MATERIALIZED cityHash64(" + PatternExpr("Body") + ") COMMENT '" + patternRulesComment + "'"
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS otel.logs").WithoutArgs().WillReturnResult(sqlmock.NewResult(0, 0))
	// Tables created before the column get it.
	mock.ExpectExec("ALTER TABLE otel.logs ADD COLUMN IF NOT EXISTS " + column).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT comment FROM system.columns WHERE database = 'otel' AND table = 'logs' AND name = 'PatternHash'").
		WillReturnRows(sqlmock.NewRows([]string{"comment"}).AddRow("pattern rules 0123456789abcdef"))
	// Other rules: new lines get this version's expression, and older ones are rewritten in the background.
	mock.ExpectExec("ALTER TABLE otel.logs MODIFY COLUMN PatternHash UInt64 MATERIALIZED cityHash64(" + PatternExpr("Body") + ") CODEC(ZSTD(1))").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("ALTER TABLE otel.logs MATERIALIZE COLUMN PatternHash").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("ALTER TABLE otel.logs COMMENT COLUMN PatternHash '" + patternRulesComment + "'").WillReturnResult(sqlmock.NewResult(0, 0))

	require.NoError(t, ensureLogsTable(t.Context(), db, "MergeTree", 7))
}

func TestEnsureLogsTable_SameRules(t *testing.T) {
	db, mock := newSchemaMock(t)
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS otel.logs").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("ALTER TABLE otel.logs ADD COLUMN IF NOT EXISTS PatternHash").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT comment FROM system.columns").WillReturnRows(sqlmock.NewRows([]string{"comment"}).AddRow(patternRulesComment))

	require.NoError(t, ensureLogsTable(t.Context(), db, "MergeTree", 7))
}

func TestPatternRulesComment(t *testing.T) {
	assert.Regexp(t, "^pattern rules [0-9a-f]{16}$", patternRulesComment)
}

func TestLogsTableDDL(t *testing.T) {
	require.Contains(t, logsTableDDL("MergeTree", 7), "PatternHash UInt64 MATERIALIZED cityHash64("+PatternExpr("Body")+") COMMENT '"+patternRulesComment+"'")
}
