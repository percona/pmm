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
	"fmt"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestEnsureLogsTable(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherFunc(func(expected, actual string) error {
		if !strings.Contains(actual, expected) {
			return fmt.Errorf("%s does not contain %s", actual, expected)
		}
		return nil
	})))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	column := "PatternHash UInt64 MATERIALIZED cityHash64(" + PatternExpr("Body") + ")"
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS otel.logs").WithoutArgs().WillReturnResult(sqlmock.NewResult(0, 0))
	// Tables created before the column get it; the expression follows the masking rules of this version.
	mock.ExpectExec("ALTER TABLE otel.logs ADD COLUMN IF NOT EXISTS " + column).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("ALTER TABLE otel.logs MODIFY COLUMN " + column).WillReturnResult(sqlmock.NewResult(0, 0))

	require.NoError(t, ensureLogsTable(t.Context(), db, "MergeTree", 7))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLogsTableDDL(t *testing.T) {
	require.Contains(t, logsTableDDL("MergeTree", 7), "PatternHash UInt64 MATERIALIZED cityHash64("+PatternExpr("Body")+")")
}
