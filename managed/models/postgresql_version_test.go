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
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"
)

func TestGetPostgreSQLVersion(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		mockedData  []string
		wantVersion PostgreSQLVersion
		wantError   bool
	}{
		{
			name: "PostgreSQL 10.9",
			mockedData: []string{
				"PostgreSQL 10.9 (Debian 10.9-1.pgdg90+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 6.3.0-18+deb9u1) 6.3.0 20170516, 64-bit",
			},
			wantVersion: PostgreSQLVersion{text: "10.9", number: 10.9},
			wantError:   false,
		},
		{
			name: "PostgreSQL 9.4.23",
			mockedData: []string{
				"PostgreSQL 9.4.23 on x86_64-pc-linux-gnu (Debian 9.4.23-1.pgdg90+1), compiled by gcc (Debian 6.3.0-18+deb9u1) 6.3.0 20170516, 64-bit",
			},
			wantVersion: PostgreSQLVersion{text: "9.4", number: 9.4},
			wantError:   false,
		},
		{
			name: "PostgreSQL 12beta2",
			mockedData: []string{
				"PostgreSQL 12beta2 (Debian 12~beta2-1.pgdg100+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 8.3.0-6) 8.3.0, 64-bit",
			},
			wantVersion: PostgreSQLVersion{text: "12", number: 12},
			wantError:   false,
		},
		{
			name: "Non existent PostgreSQL version",
			mockedData: []string{
				"Non existent PostgreSQL version",
			},
			wantVersion: PostgreSQLVersion{},
			wantError:   true,
		},
	}
	column := []string{"version"}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sqlDB, mock, err := sqlmock.New()
			require.NoError(t, err)
			t.Cleanup(func() { sqlDB.Close() }) //nolint:errcheck

			q := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf)).WithTag("pmm-agent:postgresqlversion")
			ctx := context.Background()

			for _, version := range tc.mockedData {
				mock.ExpectQuery("SELECT version()").
					WillReturnRows(sqlmock.NewRows(column).AddRow(version))
			}

			version, err := GetPostgreSQLVersion(ctx, q)
			if tc.wantError {
				assert.Error(t, err)
			} else {
				assert.Equal(t, tc.wantVersion.Float(), version.Float())
				assert.Equal(t, tc.wantVersion.String(), version.String())
				assert.NoError(t, err)
			}
		})
	}
}
