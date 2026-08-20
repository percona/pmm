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

package models

import (
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvisionDBAndRole(t *testing.T) {
	t.Parallel()

	params := SetupDBParams{
		Name:     "pmm-managed",
		Username: "pmm-managed",
		Password: "pmm-managed",
	}

	const (
		countDatabasesQuery = `SELECT COUNT(*) FROM pg_database WHERE datname = $1`
		countRolesQuery     = `SELECT COUNT(*) FROM pg_roles WHERE rolname=$1`
	)

	t.Run("creates database and role when both are missing", func(t *testing.T) {
		t.Parallel()

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close() //nolint:errcheck

		mock.ExpectQuery(regexp.QuoteMeta(countDatabasesQuery)).
			WithArgs(params.Name).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectExec(regexp.QuoteMeta(`CREATE DATABASE "pmm-managed"`)).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectQuery(regexp.QuoteMeta(countRolesQuery)).
			WithArgs(params.Username).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectExec(regexp.QuoteMeta(`CREATE USER "pmm-managed" LOGIN PASSWORD 'pmm-managed'`)).
			WillReturnResult(sqlmock.NewResult(0, 0))
		// The identifiers must be quoted into the statement: PostgreSQL does not accept bind
		// parameters in place of a database name or a role name.
		mock.ExpectExec(regexp.QuoteMeta(`GRANT ALL PRIVILEGES ON DATABASE "pmm-managed" TO "pmm-managed"`)).
			WillReturnResult(sqlmock.NewResult(0, 0))

		require.NoError(t, provisionDBAndRole(db, params))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("resets password and grants privileges when the role already exists", func(t *testing.T) {
		t.Parallel()

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close() //nolint:errcheck

		mock.ExpectQuery(regexp.QuoteMeta(countDatabasesQuery)).
			WithArgs(params.Name).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(regexp.QuoteMeta(countRolesQuery)).
			WithArgs(params.Username).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectExec(regexp.QuoteMeta(`ALTER USER "pmm-managed" WITH PASSWORD 'pmm-managed'`)).
			WillReturnResult(sqlmock.NewResult(0, 0))
		// A role provisioned without privileges gets them on this path too.
		mock.ExpectExec(regexp.QuoteMeta(`GRANT ALL PRIVILEGES ON DATABASE "pmm-managed" TO "pmm-managed"`)).
			WillReturnResult(sqlmock.NewResult(0, 0))

		require.NoError(t, provisionDBAndRole(db, params))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("quotes identifiers and passwords containing special characters", func(t *testing.T) {
		t.Parallel()

		special := SetupDBParams{
			Name:     `pmm"db`,
			Username: `pmm"user`,
			Password: `p'wd\n`,
		}

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close() //nolint:errcheck

		mock.ExpectQuery(regexp.QuoteMeta(countDatabasesQuery)).
			WithArgs(special.Name).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectExec(regexp.QuoteMeta(`CREATE DATABASE "pmm""db"`)).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectQuery(regexp.QuoteMeta(countRolesQuery)).
			WithArgs(special.Username).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
		// pq.QuoteLiteral doubles the single quote, escapes the backslash and switches to the
		// C-style escape syntax, which is where the extra leading space in front of E comes from.
		mock.ExpectExec(regexp.QuoteMeta(`CREATE USER "pmm""user" LOGIN PASSWORD  E'p''wd\\n'`)).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec(regexp.QuoteMeta(`GRANT ALL PRIVILEGES ON DATABASE "pmm""db" TO "pmm""user"`)).
			WillReturnResult(sqlmock.NewResult(0, 0))

		require.NoError(t, provisionDBAndRole(db, special))
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
