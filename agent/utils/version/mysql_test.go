// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package version

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/mysql"
)

func TestGetMySQLVersion(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Log("error creating mock database")
		return
	}
	defer sqlDB.Close() //nolint:errcheck

	q := reform.NewDB(sqlDB, mysql.Dialect, reform.NewPrintfLogger(t.Logf)).WithTag("pmm-agent:mysqlversion")
	ctx := context.Background()

	type mockedVariables struct {
		variable string
		value    string
	}

	type testingCase struct {
		name        string
		mockedData  []mockedVariables
		wantVendor  MySQLVendor
		wantVersion string
	}

	testCases := []*testingCase{
		{
			name: "Percona Server",
			mockedData: []mockedVariables{
				{
					variable: "version",
					value:    "8.0.26-17",
				},
				{
					variable: "version_comment",
					value:    "Percona Server (GPL), Release 17, Revision d7119cd",
				},
			},
			wantVendor:  PerconaVendor,
			wantVersion: "8.0",
		},
		{
			name: "MySQL",
			mockedData: []mockedVariables{
				{
					variable: "version",
					value:    "8.0.28",
				},
				{
					variable: "version_comment",
					value:    "MySQL Community Server - GPL",
				},
			},
			wantVendor:  OracleVendor,
			wantVersion: "8.0",
		},
		{
			name: "MariaDB",
			mockedData: []mockedVariables{
				{
					variable: "version",
					value:    "10.2.43-MariaDB-1:10.2.43+maria~bionic",
				},
				{
					variable: "version_comment",
					value:    "mariadb.org binary distribution",
				},
			},
			wantVendor:  MariaDBVendor,
			wantVersion: "10.2",
		},
		{
			name: "MariaDB-Debian",
			mockedData: []mockedVariables{
				{
					variable: "version",
					value:    "10.1.48-MariaDB-0+deb9u2",
				},
				{
					variable: "version_comment",
					value:    "Debian 9.13",
				},
			},
			wantVendor:  MariaDBVendor,
			wantVersion: "10.1",
		},
	}

	//nolint:paralleltest
	for _, testCase := range testCases {
		tc := testCase //nolint:varnamelen
		t.Run(tc.name, func(t *testing.T) {
			// Don't run in parallel. If this is ran in parallel, there is no way to know
			// in which case we are.
			columns := []string{"variable_name", "value"}
			for _, mockedVar := range tc.mockedData {
				mock.ExpectQuery("SHOW").
					WillReturnRows(sqlmock.NewRows(columns).AddRow(mockedVar.variable, mockedVar.value))
			}

			version, vendor, err := GetMySQLVersion(ctx, q)
			assert.Equal(t, tc.wantVersion, version.String())
			assert.Equal(t, tc.wantVendor, vendor)
			assert.NoError(t, err)
		})
	}
}
