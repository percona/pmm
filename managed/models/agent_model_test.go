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

package models_test

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

func TestAgent(t *testing.T) {
	t.Run("UnifiedLabels", func(t *testing.T) {
		agent := &models.Agent{
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
		agent := &models.Agent{
			Username:          pointer.ToString("username"),
			Password:          pointer.ToString("s3cur3 p@$$w0r4."),
			ExporterOptions:   &models.ExporterOptions{},
			QANOptions:        &models.QANOptions{},
			MongoDBOptions:    &models.MongoDBOptions{},
			MySQLOptions:      &models.MySQLOptions{},
			PostgreSQLOptions: &models.PostgreSQLOptions{},
		}
		service := &models.Service{
			Address: pointer.ToString("1.2.3.4"),
			Port:    pointer.ToUint16(12345),
		}
		for typ, expected := range map[models.AgentType]string{
			models.MySQLdExporterType:          "username:s3cur3 p@$$w0r4.@tcp(1.2.3.4:12345)/database?timeout=1s",
			models.ProxySQLExporterType:        "username:s3cur3 p@$$w0r4.@tcp(1.2.3.4:12345)/database?timeout=1s",
			models.QANMySQLPerfSchemaAgentType: "username:s3cur3 p@$$w0r4.@tcp(1.2.3.4:12345)/database?clientFoundRows=true&parseTime=true&timeout=1s",
			models.QANMySQLSlowlogAgentType:    "username:s3cur3 p@$$w0r4.@tcp(1.2.3.4:12345)/database?clientFoundRows=true&parseTime=true&timeout=1s",
			models.MongoDBExporterType:         "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/database?connectTimeoutMS=1000&directConnection=true&serverSelectionTimeoutMS=1000",
			models.QANMongoDBProfilerAgentType: "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/database?connectTimeoutMS=1000&directConnection=true&serverSelectionTimeoutMS=1000",
			models.PostgresExporterType:        "postgres://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/database?connect_timeout=1&sslmode=disable",
		} {
			t.Run(string(typ), func(t *testing.T) {
				agent.AgentType = typ
				assert.Equal(t, expected, agent.DSN(service, models.DSNParams{DialTimeout: time.Second, Database: "database"}, nil, nil))
			})
		}

		t.Run("MongoDBNoDatabase", func(t *testing.T) {
			agent.AgentType = models.MongoDBExporterType

			assert.Equal(t, "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/?connectTimeoutMS=1000&directConnection=true&serverSelectionTimeoutMS=1000", agent.DSN(service, models.DSNParams{DialTimeout: time.Second}, nil, nil))
			assert.Equal(t, "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/?directConnection=true", agent.DSN(service, models.DSNParams{}, nil, nil))
		})
	})

	t.Run("DSN socket", func(t *testing.T) {
		agent := &models.Agent{
			Username:        pointer.ToString("username"),
			Password:        pointer.ToString("s3cur3 p@$$w0r4."),
			ExporterOptions: &models.ExporterOptions{},
			QANOptions:      &models.QANOptions{},
			MySQLOptions:    &models.MySQLOptions{},
		}
		service := &models.Service{
			Socket: pointer.ToString("/var/run/mysqld/mysqld.sock"),
		}
		for typ, expected := range map[models.AgentType]string{
			models.MySQLdExporterType:          "username:s3cur3 p@$$w0r4.@unix(/var/run/mysqld/mysqld.sock)/database?timeout=1s",
			models.ProxySQLExporterType:        "username:s3cur3 p@$$w0r4.@unix(/var/run/mysqld/mysqld.sock)/database?timeout=1s",
			models.QANMySQLPerfSchemaAgentType: "username:s3cur3 p@$$w0r4.@unix(/var/run/mysqld/mysqld.sock)/database?clientFoundRows=true&parseTime=true&timeout=1s",
			models.QANMySQLSlowlogAgentType:    "username:s3cur3 p@$$w0r4.@unix(/var/run/mysqld/mysqld.sock)/database?clientFoundRows=true&parseTime=true&timeout=1s",
		} {
			t.Run(string(typ), func(t *testing.T) {
				agent.AgentType = typ
				assert.Equal(t, expected, agent.DSN(service, models.DSNParams{DialTimeout: time.Second, Database: "database"}, nil, nil))
			})
		}
	})

	t.Run("DSN timeout", func(t *testing.T) {
		agent := &models.Agent{
			Username:        pointer.ToString("username"),
			Password:        pointer.ToString("s3cur3 p@$$w0r4."),
			ExporterOptions: &models.ExporterOptions{},
			QANOptions:      &models.QANOptions{},
			MongoDBOptions:  &models.MongoDBOptions{},
		}
		service := &models.Service{
			Socket: pointer.ToString("/var/run/mysqld/mysqld.sock"),
		}
		for typ, expected := range map[models.AgentType]string{
			models.MongoDBExporterType:         "mongodb://username:s3cur3%20p%40$$w0r4.@%2Fvar%2Frun%2Fmysqld%2Fmysqld.sock/database?connectTimeoutMS=1000&directConnection=true&serverSelectionTimeoutMS=1000",
			models.QANMongoDBProfilerAgentType: "mongodb://username:s3cur3%20p%40$$w0r4.@%2Fvar%2Frun%2Fmysqld%2Fmysqld.sock/database?connectTimeoutMS=1000&directConnection=true&serverSelectionTimeoutMS=1000",
		} {
			t.Run(string(typ), func(t *testing.T) {
				agent.AgentType = typ
				assert.Equal(t, expected, agent.DSN(service, models.DSNParams{DialTimeout: time.Second, Database: "database"}, nil, nil))
			})
		}
	})

	t.Run("DSN ssl", func(t *testing.T) {
		mongoDBOptions := models.MongoDBOptions{
			TLSCertificateKey:             "key",
			TLSCertificateKeyFilePassword: "pass",
			TLSCa:                         "cert",
			AuthenticationMechanism:       "MONGODB-X509",
		}
		mysqlOptions := models.MySQLOptions{
			TLSCa:   "ca",
			TLSCert: "cert",
			TLSKey:  "key",
		}
		postgresqlOptions := models.PostgreSQLOptions{
			SSLCa:   "ca",
			SSLCert: "cert",
			SSLKey:  "key",
		}
		agent := &models.Agent{
			Username:          pointer.ToString("username"),
			Password:          pointer.ToString("s3cur3 p@$$w0r4."),
			TLS:               true,
			ExporterOptions:   &models.ExporterOptions{},
			MongoDBOptions:    &mongoDBOptions,
			MySQLOptions:      &mysqlOptions,
			PostgreSQLOptions: &postgresqlOptions,
		}
		service := &models.Service{
			Address: pointer.ToString("1.2.3.4"),
			Port:    pointer.ToUint16(12345),
		}

		for typ, expected := range map[models.AgentType]string{
			models.MySQLdExporterType:          "username:s3cur3 p@$$w0r4.@tcp(1.2.3.4:12345)/database?timeout=1s&tls=custom",
			models.ProxySQLExporterType:        "username:s3cur3 p@$$w0r4.@tcp(1.2.3.4:12345)/database?timeout=1s&tls=true",
			models.QANMySQLPerfSchemaAgentType: "username:s3cur3 p@$$w0r4.@tcp(1.2.3.4:12345)/database?clientFoundRows=true&parseTime=true&timeout=1s&tls=custom",
			models.QANMySQLSlowlogAgentType:    "username:s3cur3 p@$$w0r4.@tcp(1.2.3.4:12345)/database?clientFoundRows=true&parseTime=true&timeout=1s&tls=custom",
			models.MongoDBExporterType:         "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/database?authMechanism=MONGODB-X509&connectTimeoutMS=1000&directConnection=true&serverSelectionTimeoutMS=1000&ssl=true&tlsCaFile={{.TextFiles.caFilePlaceholder}}&tlsCertificateKeyFile={{.TextFiles.certificateKeyFilePlaceholder}}&tlsCertificateKeyFilePassword=pass",
			models.QANMongoDBProfilerAgentType: "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/database?authMechanism=MONGODB-X509&connectTimeoutMS=1000&directConnection=true&serverSelectionTimeoutMS=1000&ssl=true&tlsCaFile={{.TextFiles.caFilePlaceholder}}&tlsCertificateKeyFile={{.TextFiles.certificateKeyFilePlaceholder}}&tlsCertificateKeyFilePassword=pass",
			models.PostgresExporterType:        "postgres://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/database?connect_timeout=1&sslcert={{.TextFiles.certificateFilePlaceholder}}&sslkey={{.TextFiles.certificateKeyFilePlaceholder}}&sslmode=verify-ca&sslrootcert={{.TextFiles.caFilePlaceholder}}",
		} {
			t.Run(string(typ), func(t *testing.T) {
				agent.AgentType = typ
				assert.Equal(t, expected, agent.DSN(service, models.DSNParams{DialTimeout: time.Second, Database: "database"}, nil, nil))
			})
		}

		t.Run("MongoDBNoDatabase", func(t *testing.T) {
			agent.AgentType = models.MongoDBExporterType
			// reset values from the previous test
			agent.MongoDBOptions.TLSCertificateKeyFilePassword = ""
			agent.MongoDBOptions.AuthenticationMechanism = ""

			assert.Equal(t, "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/?connectTimeoutMS=1000&directConnection=true&serverSelectionTimeoutMS=1000&ssl=true&tlsCaFile={{.TextFiles.caFilePlaceholder}}&tlsCertificateKeyFile={{.TextFiles.certificateKeyFilePlaceholder}}", agent.DSN(service, models.DSNParams{DialTimeout: time.Second}, nil, nil))
			assert.Equal(t, "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/?directConnection=true&ssl=true&tlsCaFile={{.TextFiles.caFilePlaceholder}}&tlsCertificateKeyFile={{.TextFiles.certificateKeyFilePlaceholder}}", agent.DSN(service, models.DSNParams{}, nil, nil))
			expectedFiles := map[string]string{
				"caFilePlaceholder":             "cert",
				"certificateKeyFilePlaceholder": "key",
			}
			assert.Equal(t, expectedFiles, agent.Files())
		})

		t.Run("MongoDB Auth Database", func(t *testing.T) {
			agent.AgentType = models.MongoDBExporterType
			// reset values from the previous test
			agent.MongoDBOptions.TLSCertificateKeyFilePassword = ""
			agent.MongoDBOptions.AuthenticationMechanism = "MONGO-X509"
			agent.MongoDBOptions.AuthenticationDatabase = "$external"

			assert.Equal(t, "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/?authMechanism=MONGO-X509&authSource=%24external&connectTimeoutMS=1000&directConnection=true&serverSelectionTimeoutMS=1000&ssl=true&tlsCaFile={{.TextFiles.caFilePlaceholder}}&tlsCertificateKeyFile={{.TextFiles.certificateKeyFilePlaceholder}}", agent.DSN(service, models.DSNParams{DialTimeout: time.Second}, nil, nil))
			assert.Equal(t, "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/?authMechanism=MONGO-X509&authSource=%24external&directConnection=true&ssl=true&tlsCaFile={{.TextFiles.caFilePlaceholder}}&tlsCertificateKeyFile={{.TextFiles.certificateKeyFilePlaceholder}}", agent.DSN(service, models.DSNParams{}, nil, nil))
			expectedFiles := map[string]string{
				"caFilePlaceholder":             "cert",
				"certificateKeyFilePlaceholder": "key",
			}
			assert.Equal(t, expectedFiles, agent.Files())
		})
	})

	t.Run("DSN ssl-skip-verify", func(t *testing.T) {
		agent := &models.Agent{
			Username:          pointer.ToString("username"),
			Password:          pointer.ToString("s3cur3 p@$$w0r4."),
			TLS:               true,
			TLSSkipVerify:     true,
			ExporterOptions:   &models.ExporterOptions{},
			QANOptions:        &models.QANOptions{},
			MongoDBOptions:    &models.MongoDBOptions{},
			MySQLOptions:      &models.MySQLOptions{},
			PostgreSQLOptions: &models.PostgreSQLOptions{},
		}
		service := &models.Service{
			Address: pointer.ToString("1.2.3.4"),
			Port:    pointer.ToUint16(12345),
		}
		for typ, expected := range map[models.AgentType]string{
			models.MySQLdExporterType:          "username:s3cur3 p@$$w0r4.@tcp(1.2.3.4:12345)/database?timeout=1s&tls=skip-verify",
			models.ProxySQLExporterType:        "username:s3cur3 p@$$w0r4.@tcp(1.2.3.4:12345)/database?timeout=1s&tls=skip-verify",
			models.QANMySQLPerfSchemaAgentType: "username:s3cur3 p@$$w0r4.@tcp(1.2.3.4:12345)/database?clientFoundRows=true&parseTime=true&timeout=1s&tls=skip-verify",
			models.QANMySQLSlowlogAgentType:    "username:s3cur3 p@$$w0r4.@tcp(1.2.3.4:12345)/database?clientFoundRows=true&parseTime=true&timeout=1s&tls=skip-verify",
			models.MongoDBExporterType:         "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/database?connectTimeoutMS=1000&directConnection=true&serverSelectionTimeoutMS=1000&ssl=true&tlsInsecure=true",
			models.QANMongoDBProfilerAgentType: "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/database?connectTimeoutMS=1000&directConnection=true&serverSelectionTimeoutMS=1000&ssl=true&tlsInsecure=true",
			models.PostgresExporterType:        "postgres://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/database?connect_timeout=1&sslmode=require",
		} {
			t.Run(string(typ), func(t *testing.T) {
				agent.AgentType = typ
				assert.Equal(t, expected, agent.DSN(service, models.DSNParams{DialTimeout: time.Second, Database: "database"}, nil, nil))
			})
		}

		t.Run("MongoDBNoDatabase", func(t *testing.T) {
			agent.AgentType = models.MongoDBExporterType

			assert.Equal(t, "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/?connectTimeoutMS=1000&directConnection=true&serverSelectionTimeoutMS=1000&ssl=true&tlsInsecure=true", agent.DSN(service, models.DSNParams{DialTimeout: time.Second}, nil, nil))
			assert.Equal(t, "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/?directConnection=true&ssl=true&tlsInsecure=true", agent.DSN(service, models.DSNParams{}, nil, nil))
		})
	})
}

func TestPostgresAgentTLS(t *testing.T) {
	agent := &models.Agent{
		Username:          pointer.ToString("username"),
		Password:          pointer.ToString("s3cur3 p@$$w0r4."),
		AgentType:         models.PostgresExporterType,
		ExporterOptions:   &models.ExporterOptions{},
		PostgreSQLOptions: &models.PostgreSQLOptions{},
	}
	service := &models.Service{
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
		{true, false, "postgres://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/database?connect_timeout=1&sslmode=verify-ca"},
		{true, true, "postgres://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/database?connect_timeout=1&sslmode=require"},
	} {
		name := fmt.Sprintf("TLS:%v/TLSSkipVerify:%v", testCase.tls, testCase.tlsSkipVerify)
		t.Run(name, func(t *testing.T) {
			agent.TLS = testCase.tls
			agent.TLSSkipVerify = testCase.tlsSkipVerify
			assert.Equal(t, testCase.expected, agent.DSN(service, models.DSNParams{DialTimeout: time.Second, Database: "database"}, nil, nil))
		})
		t.Run(fmt.Sprintf("AutodiscoveryLimit set TLS:%v/TLSSkipVerify:%v", testCase.tls, testCase.tlsSkipVerify), func(t *testing.T) {
			agent.TLS = testCase.tls
			agent.TLSSkipVerify = testCase.tlsSkipVerify
			agent.PostgreSQLOptions = &models.PostgreSQLOptions{AutoDiscoveryLimit: pointer.ToInt32(10)}
			assert.Equal(t, testCase.expected, agent.DSN(service, models.DSNParams{DialTimeout: time.Second, Database: "database"}, nil, nil))
		})
	}
}

func TestPostgresWithSocket(t *testing.T) {
	t.Run("empty-password", func(t *testing.T) {
		agent := &models.Agent{
			Username:          pointer.ToString("username"),
			AgentType:         models.PostgresExporterType,
			TLS:               true,
			TLSSkipVerify:     false,
			ExporterOptions:   &models.ExporterOptions{},
			PostgreSQLOptions: &models.PostgreSQLOptions{},
		}
		service := &models.Service{
			Socket: pointer.ToString("/var/run/postgres"),
		}
		expect := "postgres://username@/database?connect_timeout=1&host=%2Fvar%2Frun%2Fpostgres&sslmode=verify-ca"
		assert.Equal(t, expect, agent.DSN(service, models.DSNParams{DialTimeout: time.Second, Database: "database"}, nil, nil))
	})

	t.Run("empty-user-password", func(t *testing.T) {
		agent := &models.Agent{
			AgentType:         models.PostgresExporterType,
			ExporterOptions:   &models.ExporterOptions{},
			PostgreSQLOptions: &models.PostgreSQLOptions{},
		}
		service := &models.Service{
			Socket: pointer.ToString("/var/run/postgres"),
		}
		expect := "postgres:///database?connect_timeout=1&host=%2Fvar%2Frun%2Fpostgres&sslmode=disable"
		assert.Equal(t, expect, agent.DSN(service, models.DSNParams{DialTimeout: time.Second, Database: "database"}, nil, nil))
	})

	t.Run("dir-with-symbols", func(t *testing.T) {
		agent := &models.Agent{
			AgentType:         models.PostgresExporterType,
			ExporterOptions:   &models.ExporterOptions{},
			PostgreSQLOptions: &models.PostgreSQLOptions{},
		}
		service := &models.Service{
			Socket: pointer.ToString(`/tmp/123\ A0m\%\$\@\8\,\+\-`),
		}
		expect := "postgres:///database?connect_timeout=1&host=%2Ftmp%2F123%5C+A0m%5C%25%5C%24%5C%40%5C8%5C%2C%5C%2B%5C-&sslmode=disable"
		assert.Equal(t, expect, agent.DSN(service, models.DSNParams{DialTimeout: time.Second, Database: "database"}, nil, nil))
	})
}

func TestMongoWithSocket(t *testing.T) {
	t.Run("empty-password", func(t *testing.T) {
		agent := &models.Agent{
			Username:        pointer.ToString("username"),
			AgentType:       models.MongoDBExporterType,
			TLS:             true,
			TLSSkipVerify:   false,
			ExporterOptions: &models.ExporterOptions{},
			MongoDBOptions:  &models.MongoDBOptions{},
		}
		service := &models.Service{
			Socket: pointer.ToString("/tmp/mongodb-27017.sock"),
		}
		expect := "mongodb://username@%2Ftmp%2Fmongodb-27017.sock/database?connectTimeoutMS=1000&directConnection=true&serverSelectionTimeoutMS=1000&ssl=true"
		assert.Equal(t, expect, agent.DSN(service, models.DSNParams{DialTimeout: time.Second, Database: "database"}, nil, nil))
	})

	t.Run("empty-user-password", func(t *testing.T) {
		agent := &models.Agent{
			AgentType:       models.MongoDBExporterType,
			ExporterOptions: &models.ExporterOptions{},
			MongoDBOptions:  &models.MongoDBOptions{},
		}
		service := &models.Service{
			Socket: pointer.ToString("/tmp/mongodb-27017.sock"),
		}
		expect := "mongodb://%2Ftmp%2Fmongodb-27017.sock/database?connectTimeoutMS=1000&directConnection=true&serverSelectionTimeoutMS=1000"
		assert.Equal(t, expect, agent.DSN(service, models.DSNParams{DialTimeout: time.Second, Database: "database"}, nil, nil))
	})

	t.Run("dir-with-symbols", func(t *testing.T) {
		agent := &models.Agent{
			AgentType:       models.MongoDBExporterType,
			ExporterOptions: &models.ExporterOptions{},
			MongoDBOptions:  &models.MongoDBOptions{},
		}
		service := &models.Service{
			Socket: pointer.ToString(`/tmp/123\ A0m\%\$\@\8\,\+\-/mongodb-27017.sock`),
		}
		expect := "mongodb://%2Ftmp%2F123%5C%20A0m%5C%25%5C$%5C%40%5C8%5C,%5C+%5C-%2Fmongodb-27017.sock/database?connectTimeoutMS=1000&directConnection=true&serverSelectionTimeoutMS=1000"
		assert.Equal(t, expect, agent.DSN(service, models.DSNParams{DialTimeout: time.Second, Database: "database"}, nil, nil))
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
			agent := &models.Agent{
				AgentType: models.MySQLdExporterType,
				MySQLOptions: &models.MySQLOptions{
					TableCount:                     testCase.count,
					TableCountTablestatsGroupLimit: testCase.limit,
				},
			}
			assert.Equal(t, testCase.expected, agent.IsMySQLTablestatsGroupEnabled())
		})
	}
}

func TestExporterURL(t *testing.T) {
	now, origNowF := models.Now(), models.Now
	models.Now = func() time.Time {
		return now
	}
	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	defer func() {
		models.Now = origNowF
		require.NoError(t, sqlDB.Close())
	}()

	setup := func(t *testing.T) (*reform.Querier, func(t *testing.T)) {
		t.Helper()
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		tx, err := db.Begin()
		require.NoError(t, err)
		q := tx.Querier

		for _, str := range []reform.Struct{
			&models.Node{
				NodeID:    "ExporterNodeID",
				NodeType:  models.ContainerNodeType,
				NodeName:  "Node for Exporter",
				MachineID: pointer.ToString("ExporterNode"),
				Address:   "172.20.0.4",
			},

			&models.Node{
				NodeID:   "ExporterServerlessNodeID",
				NodeType: models.RemoteNodeType,
				NodeName: "Node for Serverless Exporter",
				Address:  "redis_exporter",
			},

			&models.Node{
				NodeID:   "ExporterServerlessNodeID2",
				NodeType: models.RemoteNodeType,
				NodeName: "Node 2 for Serverless Exporter",
				Address:  "nomad_exporter",
			},

			&models.Service{
				ServiceID:     "external",
				ServiceType:   models.ExternalServiceType,
				ServiceName:   "Service on ExporterNodeID",
				NodeID:        "ExporterNodeID",
				ExternalGroup: "redis",
			},

			&models.Service{
				ServiceID:     "redis_exporter-external",
				ServiceType:   models.ExternalServiceType,
				ServiceName:   "Service on ExporterServerlessNode",
				NodeID:        "ExporterServerlessNodeID",
				ExternalGroup: "redis",
			},

			&models.Service{
				ServiceID:     "nomad_exporter-external",
				ServiceType:   models.ExternalServiceType,
				ServiceName:   "Service on ExporterServerlessNode 2",
				NodeID:        "ExporterServerlessNodeID2",
				ExternalGroup: "nomad",
			},

			&models.Agent{
				AgentID:      "ExporterAgentPush",
				AgentType:    models.ExternalExporterType,
				ServiceID:    pointer.ToString("external"),
				RunsOnNodeID: pointer.ToString("ExporterNodeID"),
				ListenPort:   pointer.ToUint16(9121),
				ExporterOptions: &models.ExporterOptions{
					PushMetrics:   true,
					MetricsPath:   pointer.ToString("/metrics"),
					MetricsScheme: pointer.ToString("http"),
				},
			},

			&models.Agent{
				AgentID:      "ExporterAgentPull",
				AgentType:    models.ExternalExporterType,
				ServiceID:    pointer.ToString("external"),
				RunsOnNodeID: pointer.ToString("ExporterNodeID"),
				ListenPort:   pointer.ToUint16(9121),
				Username:     pointer.ToString("user"),
				Password:     pointer.ToString("secret"),
				ExporterOptions: &models.ExporterOptions{
					PushMetrics:   false,
					MetricsPath:   pointer.ToString("/metrics"),
					MetricsScheme: pointer.ToString("http"),
				},
			},

			&models.Agent{
				AgentID:      "ExporterServerless",
				AgentType:    models.ExternalExporterType,
				RunsOnNodeID: pointer.ToString("ExporterServerlessNodeID"),
				ServiceID:    pointer.ToString("redis_exporter-external"),
				ListenPort:   pointer.ToUint16(9121),
				Username:     pointer.ToString("user"),
				Password:     pointer.ToString("secret"),
				ExporterOptions: &models.ExporterOptions{
					PushMetrics:   false,
					MetricsPath:   pointer.ToString("/metrics"),
					MetricsScheme: pointer.ToString("http"),
				},
			},

			&models.Agent{
				AgentID:      "ExporterServerlessWithQueryParams",
				AgentType:    models.ExternalExporterType,
				RunsOnNodeID: pointer.ToString("ExporterServerlessNodeID2"),
				ServiceID:    pointer.ToString("nomad_exporter-external"),
				ListenPort:   pointer.ToUint16(9121),
				Username:     pointer.ToString("user"),
				Password:     pointer.ToString("secret"),
				ExporterOptions: &models.ExporterOptions{
					PushMetrics:   false,
					MetricsPath:   pointer.ToString("/metrics?format=prometheus&output=json"),
					MetricsScheme: pointer.ToString("http"),
				},
			},

			&models.Agent{
				AgentID:      "ExporterServerlessWithEmptyMetricsPath",
				AgentType:    models.ExternalExporterType,
				RunsOnNodeID: pointer.ToString("ExporterServerlessNodeID2"),
				ServiceID:    pointer.ToString("nomad_exporter-external"),
				ListenPort:   pointer.ToUint16(9121),
				Username:     pointer.ToString("user"),
				Password:     pointer.ToString("secret"),
				ExporterOptions: &models.ExporterOptions{
					PushMetrics:   false,
					MetricsPath:   pointer.ToString("/"),
					MetricsScheme: pointer.ToString("http"),
				},
			},
		} {
			require.NoError(t, q.Insert(str), "failed to INSERT %+v", str)
		}

		teardown := func(t *testing.T) {
			t.Helper()
			require.NoError(t, tx.Rollback())
		}
		return q, teardown
	}

	t.Run("ExporterURL", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		for agentID, expected := range map[string]string{
			"ExporterAgentPush":                      "http://127.0.0.1:9121/metrics",
			"ExporterAgentPull":                      "http://user:secret@172.20.0.4:9121/metrics",
			"ExporterServerless":                     "http://user:secret@redis_exporter:9121/metrics",
			"ExporterServerlessWithQueryParams":      "http://user:secret@nomad_exporter:9121/metrics?format=prometheus&output=json",
			"ExporterServerlessWithEmptyMetricsPath": "http://user:secret@nomad_exporter:9121/",
		} {
			t.Run(agentID, func(t *testing.T) {
				agent, err := models.FindAgentByID(q, agentID)
				assert.NoError(t, err)
				actual, err := agent.ExporterURL(q)
				assert.NoError(t, err)
				assert.Equal(t, expected, actual)
			})
		}
	})
}
