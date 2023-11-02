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

package management

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	mysql "github.com/percona/pmm/api/managementpb/json/client/my_sql"
)

func TestAddMySQL(t *testing.T) {
	t.Run("TablestatEnabled", func(t *testing.T) {
		res := &addMySQLResult{
			Service: &mysql.AddMySQLOKBodyService{
				ServiceID:   "/service_id/1",
				ServiceName: "mysql-1",
			},
			MysqldExporter: &mysql.AddMySQLOKBodyMysqldExporter{
				TablestatsGroupTableLimit: 1000,
				TablestatsGroupDisabled:   false,
			},
			TableCount: 500,
		}
		expected := strings.TrimSpace(`
MySQL Service added.
Service ID  : /service_id/1
Service name: mysql-1

Table statistics collection enabled (the limit is 1000, the actual table count is 500).
		`)
		assert.Equal(t, expected, strings.TrimSpace(res.String()))
	})

	t.Run("TablestatEnabledNoLimit", func(t *testing.T) {
		res := &addMySQLResult{
			Service: &mysql.AddMySQLOKBodyService{
				ServiceID:   "/service_id/1",
				ServiceName: "mysql-1",
			},
			MysqldExporter: &mysql.AddMySQLOKBodyMysqldExporter{
				TablestatsGroupTableLimit: 0,
				TablestatsGroupDisabled:   false,
			},
			TableCount: 2000,
		}
		expected := strings.TrimSpace(`
MySQL Service added.
Service ID  : /service_id/1
Service name: mysql-1

Table statistics collection enabled (the table count limit is not set).
		`)
		assert.Equal(t, expected, strings.TrimSpace(res.String()))
	})

	t.Run("TablestatEnabledUnknown", func(t *testing.T) {
		res := &addMySQLResult{
			Service: &mysql.AddMySQLOKBodyService{
				ServiceID:   "/service_id/1",
				ServiceName: "mysql-1",
			},
			MysqldExporter: &mysql.AddMySQLOKBodyMysqldExporter{
				TablestatsGroupTableLimit: 1000,
				TablestatsGroupDisabled:   false,
			},
			TableCount: 0,
		}
		expected := strings.TrimSpace(`
MySQL Service added.
Service ID  : /service_id/1
Service name: mysql-1

Table statistics collection enabled (the limit is 1000, the actual table count is unknown).
		`)
		assert.Equal(t, expected, strings.TrimSpace(res.String()))
	})

	t.Run("TablestatDisabled", func(t *testing.T) {
		res := &addMySQLResult{
			Service: &mysql.AddMySQLOKBodyService{
				ServiceID:   "/service_id/1",
				ServiceName: "mysql-1",
			},
			MysqldExporter: &mysql.AddMySQLOKBodyMysqldExporter{
				TablestatsGroupTableLimit: 1000,
				TablestatsGroupDisabled:   true,
				TLS:                       true,
				TLSCa:                     "ca",
				TLSCert:                   "cert",
				TLSKey:                    "key",
			},
			TableCount: 2000,
		}
		expected := strings.TrimSpace(`
MySQL Service added.
Service ID  : /service_id/1
Service name: mysql-1

Table statistics collection disabled (the limit is 1000, the actual table count is 2000).
		`)
		assert.Equal(t, expected, strings.TrimSpace(res.String()))
	})

	t.Run("TablestatDisabledAlways", func(t *testing.T) {
		res := &addMySQLResult{
			Service: &mysql.AddMySQLOKBodyService{
				ServiceID:   "/service_id/1",
				ServiceName: "mysql-1",
			},
			MysqldExporter: &mysql.AddMySQLOKBodyMysqldExporter{
				TablestatsGroupTableLimit: -1,
				TablestatsGroupDisabled:   true,
			},
			TableCount: 2000,
		}
		expected := strings.TrimSpace(`
MySQL Service added.
Service ID  : /service_id/1
Service name: mysql-1

Table statistics collection disabled (always).
		`)
		assert.Equal(t, expected, strings.TrimSpace(res.String()))
	})

	t.Run("EmptyMysqlExporter", func(t *testing.T) {
		res := &addMySQLResult{
			MysqldExporter: nil,
		}
		expected := ""
		assert.Equal(t, expected, strings.TrimSpace(res.TablestatStatus()))
	})
}

func TestRun(t *testing.T) {
	t.Run("CreateUser", func(t *testing.T) {
		cmd := &AddMySQLCommand{
			CreateUser: true,
		}
		_, err := cmd.RunCmd()

		if assert.Error(t, err) {
			expected := "Unrecognized option. To create a user, see 'https://docs.percona.com/percona-monitoring-and-management/setting-up/client/mysql.html#create-a-database-account-for-pmm'"
			assert.Equal(t, expected, err.Error())
		}
	})
}
