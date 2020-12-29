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
				assert.Equal(t, expected, agent.DSN(service, time.Second, "database", nil))
			})
		}

		t.Run("MongoDBNoDatabase", func(t *testing.T) {
			agent.AgentType = MongoDBExporterType

			assert.Equal(t, "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/?connectTimeoutMS=1000", agent.DSN(service, time.Second, "", nil))
			assert.Equal(t, "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/", agent.DSN(service, 0, "", nil))
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
				assert.Equal(t, expected, agent.DSN(service, time.Second, "database", nil))
			})
		}
	})

	t.Run("DSN ssl", func(t *testing.T) {
		mongoDBOptions := MongoDBOptions{
			TLSCertificateKey:             "key",
			TLSCertificateKeyFilePassword: "pass",
			TLSCa:                         "cert",
		}
		agent := &Agent{
			Username:       pointer.ToString("username"),
			Password:       pointer.ToString("s3cur3 p@$$w0r4."),
			TLS:            true,
			MongoDBOptions: &mongoDBOptions,
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
			MongoDBExporterType:         "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/database?connectTimeoutMS=1000&ssl=true&tlsCaFile={{.TextFiles.caFilePlaceholder}}&tlsCertificateKeyFile={{.TextFiles.certificateKeyFilePlaceholder}}&tlsCertificateKeyFilePassword=pass",
			QANMongoDBProfilerAgentType: "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/database?connectTimeoutMS=1000&ssl=true&tlsCaFile={{.TextFiles.caFilePlaceholder}}&tlsCertificateKeyFile={{.TextFiles.certificateKeyFilePlaceholder}}&tlsCertificateKeyFilePassword=pass",
			PostgresExporterType:        "postgres://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/database?connect_timeout=1&sslmode=verify-full",
		} {
			t.Run(string(typ), func(t *testing.T) {
				agent.AgentType = typ
				assert.Equal(t, expected, agent.DSN(service, time.Second, "database", nil))
			})
		}

		t.Run("MongoDBNoDatabase", func(t *testing.T) {
			agent.AgentType = MongoDBExporterType
			agent.MongoDBOptions.TLSCertificateKeyFilePassword = ""

			assert.Equal(t, "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/?connectTimeoutMS=1000&ssl=true&tlsCaFile={{.TextFiles.caFilePlaceholder}}&tlsCertificateKeyFile={{.TextFiles.certificateKeyFilePlaceholder}}", agent.DSN(service, time.Second, "", nil))
			assert.Equal(t, "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/?ssl=true&tlsCaFile={{.TextFiles.caFilePlaceholder}}&tlsCertificateKeyFile={{.TextFiles.certificateKeyFilePlaceholder}}", agent.DSN(service, 0, "", nil))
			expectedFiles := map[string]string{
				"caFilePlaceholder":             "cert",
				"certificateKeyFilePlaceholder": "key",
			}
			assert.Equal(t, expectedFiles, agent.Files())
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
				assert.Equal(t, expected, agent.DSN(service, time.Second, "database", nil))
			})
		}

		t.Run("MongoDBNoDatabase", func(t *testing.T) {
			agent.AgentType = MongoDBExporterType

			assert.Equal(t, "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/?connectTimeoutMS=1000&ssl=true&tlsInsecure=true", agent.DSN(service, time.Second, "", nil))
			assert.Equal(t, "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/?ssl=true&tlsInsecure=true", agent.DSN(service, 0, "", nil))
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
			assert.Equal(t, testCase.expected, agent.DSN(service, time.Second, "database", nil))
		})
	}
}

func TestPostgresWithSocket(t *testing.T) {
	t.Run("empty-passowrd", func(t *testing.T) {
		agent := &Agent{
			Username:      pointer.ToString("username"),
			AgentType:     PostgresExporterType,
			TLS:           true,
			TLSSkipVerify: false,
		}
		service := &Service{
			Socket: pointer.ToString("/var/run/postgres"),
		}
		expect := "postgres://username@/database?connect_timeout=1&host=%2Fvar%2Frun%2Fpostgres&sslmode=verify-full"
		assert.Equal(t, expect, agent.DSN(service, time.Second, "database", nil))
	})

	t.Run("empty-user-passowrd", func(t *testing.T) {
		agent := &Agent{
			AgentType: PostgresExporterType,
		}
		service := &Service{
			Socket: pointer.ToString("/var/run/postgres"),
		}
		expect := "postgres:///database?connect_timeout=1&host=%2Fvar%2Frun%2Fpostgres&sslmode=disable"
		assert.Equal(t, expect, agent.DSN(service, time.Second, "database", nil))
	})

	t.Run("dir-with-symbols", func(t *testing.T) {
		agent := &Agent{
			AgentType: PostgresExporterType,
		}
		service := &Service{
			Socket: pointer.ToString(`/tmp/123\ A0m\%\$\@\8\,\+\-`),
		}
		expect := "postgres:///database?connect_timeout=1&host=%2Ftmp%2F123%5C+A0m%5C%25%5C%24%5C%40%5C8%5C%2C%5C%2B%5C-&sslmode=disable"
		assert.Equal(t, expect, agent.DSN(service, time.Second, "database", nil))
	})
}

func TestMongoWithSocket(t *testing.T) {
	t.Run("empty-passowrd", func(t *testing.T) {
		agent := &Agent{
			Username:      pointer.ToString("username"),
			AgentType:     MongoDBExporterType,
			TLS:           true,
			TLSSkipVerify: false,
		}
		service := &Service{
			Socket: pointer.ToString("/tmp/mongodb-27017.sock"),
		}
		expect := "mongodb://username@%2Ftmp%2Fmongodb-27017.sock/database?connectTimeoutMS=1000&ssl=true"
		assert.Equal(t, expect, agent.DSN(service, time.Second, "database", nil))
	})

	t.Run("empty-user-passowrd", func(t *testing.T) {
		agent := &Agent{
			AgentType: MongoDBExporterType,
		}
		service := &Service{
			Socket: pointer.ToString("/tmp/mongodb-27017.sock"),
		}
		expect := "mongodb://%2Ftmp%2Fmongodb-27017.sock/database?connectTimeoutMS=1000"
		assert.Equal(t, expect, agent.DSN(service, time.Second, "database", nil))
	})

	t.Run("dir-with-symbols", func(t *testing.T) {
		agent := &Agent{
			AgentType: MongoDBExporterType,
		}
		service := &Service{
			Socket: pointer.ToString(`/tmp/123\ A0m\%\$\@\8\,\+\-/mongodb-27017.sock`),
		}
		expect := "mongodb://%2Ftmp%2F123%5C%20A0m%5C%25%5C$%5C%40%5C8%5C,%5C+%5C-%2Fmongodb-27017.sock/database?connectTimeoutMS=1000"
		assert.Equal(t, expect, agent.DSN(service, time.Second, "database", nil))
	})
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
