// pmm-managed
// Copyright (C) 2017 Percona LLC
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
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgent(t *testing.T) {
	t.Run("UnifiedLabels", func(t *testing.T) {
		agent := &Agent{
			AgentID:      "agent_id",
			CustomLabels: []byte(`{"foo": "bar"}`),
		}
		actual, err := agent.UnifiedLabels()
		require.NoError(t, err)
		expected := map[string]string{
			"agent_id": "agent_id",
			"foo":      "bar",
		}
		assert.Equal(t, expected, actual)
	})

	t.Run("DSN", func(t *testing.T) {
		agent := &Agent{
			Username: pointer.ToString("username"),
			Password: pointer.ToString("s3cur3 p@$$w0r4."),
		}
		service := &Service{
			Address: pointer.ToString("1.2.3.4"),
			Port:    pointer.ToUint16(12345),
		}
		for typ, expected := range map[AgentType]string{
			MySQLdExporterType:          "username:s3cur3 p@$$w0r4.@tcp(1.2.3.4:12345)/database?timeout=1s",
			ProxySQLExporterType:        "username:s3cur3 p@$$w0r4.@tcp(1.2.3.4:12345)/database?timeout=1s",
			QANMySQLPerfSchemaAgentType: "username:s3cur3 p@$$w0r4.@tcp(1.2.3.4:12345)/database?clientFoundRows=true&parseTime=true&timeout=1s",
			QANMySQLSlowlogAgentType:    "username:s3cur3 p@$$w0r4.@tcp(1.2.3.4:12345)/database?clientFoundRows=true&parseTime=true&timeout=1s",
			MongoDBExporterType:         "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/database?connectTimeoutMS=1000",
			QANMongoDBProfilerAgentType: "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/database?connectTimeoutMS=1000",
			PostgresExporterType:        "postgres://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/database?connect_timeout=1&sslmode=disable",
		} {
			t.Run(string(typ), func(t *testing.T) {
				agent.AgentType = typ
				assert.Equal(t, expected, agent.DSN(service, time.Second, "database"))
			})
		}

		t.Run("MongoDBNoDatabase", func(t *testing.T) {
			agent.AgentType = MongoDBExporterType

			assert.Equal(t, "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/?connectTimeoutMS=1000", agent.DSN(service, time.Second, ""))
			assert.Equal(t, "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/", agent.DSN(service, 0, ""))
		})
	})

	t.Run("DSN socket", func(t *testing.T) {
		agent := &Agent{
			Username: pointer.ToString("username"),
			Password: pointer.ToString("s3cur3 p@$$w0r4."),
		}
		service := &Service{
			Socket: pointer.ToString("/var/run/mysqld/mysqld.sock"),
		}
		for typ, expected := range map[AgentType]string{
			MySQLdExporterType:          "username:s3cur3 p@$$w0r4.@unix(/var/run/mysqld/mysqld.sock)/database?timeout=1s",
			ProxySQLExporterType:        "username:s3cur3 p@$$w0r4.@unix(/var/run/mysqld/mysqld.sock)/database?timeout=1s",
			QANMySQLPerfSchemaAgentType: "username:s3cur3 p@$$w0r4.@unix(/var/run/mysqld/mysqld.sock)/database?clientFoundRows=true&parseTime=true&timeout=1s",
			QANMySQLSlowlogAgentType:    "username:s3cur3 p@$$w0r4.@unix(/var/run/mysqld/mysqld.sock)/database?clientFoundRows=true&parseTime=true&timeout=1s",
		} {
			t.Run(string(typ), func(t *testing.T) {
				agent.AgentType = typ
				assert.Equal(t, expected, agent.DSN(service, time.Second, "database"))
			})
		}
	})

	t.Run("DSN ssl", func(t *testing.T) {
		agent := &Agent{
			Username: pointer.ToString("username"),
			Password: pointer.ToString("s3cur3 p@$$w0r4."),
			TLS:      true,
		}
		service := &Service{
			Address: pointer.ToString("1.2.3.4"),
			Port:    pointer.ToUint16(12345),
		}
		for typ, expected := range map[AgentType]string{
			MySQLdExporterType:          "username:s3cur3 p@$$w0r4.@tcp(1.2.3.4:12345)/database?timeout=1s&tls=true",
			ProxySQLExporterType:        "username:s3cur3 p@$$w0r4.@tcp(1.2.3.4:12345)/database?timeout=1s&tls=true",
			QANMySQLPerfSchemaAgentType: "username:s3cur3 p@$$w0r4.@tcp(1.2.3.4:12345)/database?clientFoundRows=true&parseTime=true&timeout=1s&tls=true",
			QANMySQLSlowlogAgentType:    "username:s3cur3 p@$$w0r4.@tcp(1.2.3.4:12345)/database?clientFoundRows=true&parseTime=true&timeout=1s&tls=true",
			MongoDBExporterType:         "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/database?connectTimeoutMS=1000&ssl=true",
			QANMongoDBProfilerAgentType: "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/database?connectTimeoutMS=1000&ssl=true",
			PostgresExporterType:        "postgres://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/database?connect_timeout=1&sslmode=verify-full",
		} {
			t.Run(string(typ), func(t *testing.T) {
				agent.AgentType = typ
				assert.Equal(t, expected, agent.DSN(service, time.Second, "database"))
			})
		}

		t.Run("MongoDBNoDatabase", func(t *testing.T) {
			agent.AgentType = MongoDBExporterType

			assert.Equal(t, "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/?connectTimeoutMS=1000&ssl=true", agent.DSN(service, time.Second, ""))
			assert.Equal(t, "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/?ssl=true", agent.DSN(service, 0, ""))
		})
	})
	t.Run("DSN ssl-skip-verify", func(t *testing.T) {
		agent := &Agent{
			Username:      pointer.ToString("username"),
			Password:      pointer.ToString("s3cur3 p@$$w0r4."),
			TLS:           true,
			TLSSkipVerify: true,
		}
		service := &Service{
			Address: pointer.ToString("1.2.3.4"),
			Port:    pointer.ToUint16(12345),
		}
		for typ, expected := range map[AgentType]string{
			MySQLdExporterType:          "username:s3cur3 p@$$w0r4.@tcp(1.2.3.4:12345)/database?timeout=1s&tls=skip-verify",
			ProxySQLExporterType:        "username:s3cur3 p@$$w0r4.@tcp(1.2.3.4:12345)/database?timeout=1s&tls=skip-verify",
			QANMySQLPerfSchemaAgentType: "username:s3cur3 p@$$w0r4.@tcp(1.2.3.4:12345)/database?clientFoundRows=true&parseTime=true&timeout=1s&tls=skip-verify",
			QANMySQLSlowlogAgentType:    "username:s3cur3 p@$$w0r4.@tcp(1.2.3.4:12345)/database?clientFoundRows=true&parseTime=true&timeout=1s&tls=skip-verify",
			MongoDBExporterType:         "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/database?connectTimeoutMS=1000&ssl=true&tlsInsecure=true",
			QANMongoDBProfilerAgentType: "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/database?connectTimeoutMS=1000&ssl=true&tlsInsecure=true",
			PostgresExporterType:        "postgres://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/database?connect_timeout=1&sslmode=require",
		} {
			t.Run(string(typ), func(t *testing.T) {
				agent.AgentType = typ
				assert.Equal(t, expected, agent.DSN(service, time.Second, "database"))
			})
		}

		t.Run("MongoDBNoDatabase", func(t *testing.T) {
			agent.AgentType = MongoDBExporterType

			assert.Equal(t, "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/?connectTimeoutMS=1000&ssl=true&tlsInsecure=true", agent.DSN(service, time.Second, ""))
			assert.Equal(t, "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/?ssl=true&tlsInsecure=true", agent.DSN(service, 0, ""))
		})
	})
}

func TestPostgresAgentTLS(t *testing.T) {
	agent := &Agent{
		Username:  pointer.ToString("username"),
		Password:  pointer.ToString("s3cur3 p@$$w0r4."),
		AgentType: PostgresExporterType,
	}
	service := &Service{
		Address: pointer.ToString("1.2.3.4"),
		Port:    pointer.ToUint16(12345),
	}

	for _, testCase := range []struct {
		tls           bool
		tlsSkipVerify bool
		expected      string
	}{
		{false, false, "postgres://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/database?connect_timeout=1&sslmode=disable"},
		{false, true, "postgres://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/database?connect_timeout=1&sslmode=disable"},
		{true, false, "postgres://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/database?connect_timeout=1&sslmode=verify-full"},
		{true, true, "postgres://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/database?connect_timeout=1&sslmode=require"},
	} {
		name := fmt.Sprintf("TLS:%v/TLSSkipVerify:%v", testCase.tls, testCase.tlsSkipVerify)
		t.Run(name, func(t *testing.T) {
			agent.TLS = testCase.tls
			agent.TLSSkipVerify = testCase.tlsSkipVerify
			assert.Equal(t, testCase.expected, agent.DSN(service, time.Second, "database"))
		})
	}
}

func TestIsMySQLTablestatsGroupEnabled(t *testing.T) {
	for _, testCase := range []struct {
		count    *int32
		limit    int32
		expected bool
	}{
		{nil, -1, false},
		{nil, 0, true},
		{nil, 500, true},
		{nil, 2000, true},

		{pointer.ToInt32(1000), -1, false},
		{pointer.ToInt32(1000), 0, true},
		{pointer.ToInt32(1000), 500, false},
		{pointer.ToInt32(1000), 2000, true},
	} {
		c := "nil"
		if testCase.count != nil {
			c = strconv.Itoa(int(*testCase.count))
		}
		t.Run(fmt.Sprintf("Count:%s/Limit:%d", c, testCase.limit), func(t *testing.T) {
			agent := &Agent{
				AgentType:                      MySQLdExporterType,
				TableCount:                     testCase.count,
				TableCountTablestatsGroupLimit: testCase.limit,
			}
			assert.Equal(t, testCase.expected, agent.IsMySQLTablestatsGroupEnabled())
		})
	}
}
