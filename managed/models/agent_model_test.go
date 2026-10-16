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
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/secrets"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/utils/vault"
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
		require.Equal(t, expected, actual)
	})

	t.Run("EnvironmentVariableNames", func(t *testing.T) {
		t.Run("round-trips valid names", func(t *testing.T) {
			agent := &models.Agent{}
			require.NoError(t, agent.SetEnvironmentVariableNames([]string{"KRB5_KTNAME", "https_proxy"}))

			names, err := agent.GetEnvironmentVariableNames()
			require.NoError(t, err)
			assert.Equal(t, []string{"KRB5_KTNAME", "https_proxy"}, names)
		})

		t.Run("empty slice clears stored names", func(t *testing.T) {
			agent := &models.Agent{}
			require.NoError(t, agent.SetEnvironmentVariableNames([]string{"KRB5_KTNAME"}))
			require.NoError(t, agent.SetEnvironmentVariableNames(nil))

			names, err := agent.GetEnvironmentVariableNames()
			require.NoError(t, err)
			assert.Empty(t, names)
		})

		t.Run("rejects pmm-agent's own reserved namespace", func(t *testing.T) {
			agent := &models.Agent{}
			err := agent.SetEnvironmentVariableNames([]string{"PMM_AGENT_SERVER_PASSWORD"})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "reserved for pmm-agent")
			assert.Nil(t, agent.EnvironmentVariables)
		})

		t.Run("rejects malformed names", func(t *testing.T) {
			agent := &models.Agent{}
			err := agent.SetEnvironmentVariableNames([]string{"KRB5-KTNAME"})
			require.Error(t, err)
			assert.Nil(t, agent.EnvironmentVariables)
		})

		t.Run("duplicate input round-trips as one stored name", func(t *testing.T) {
			agent := &models.Agent{}
			require.NoError(t, agent.SetEnvironmentVariableNames([]string{"KRB5_KTNAME", " KRB5_KTNAME "}))

			names, err := agent.GetEnvironmentVariableNames()
			require.NoError(t, err)
			assert.Equal(t, []string{"KRB5_KTNAME"}, names)
		})

		t.Run("grandfathers an already-stored name that would now fail validation", func(t *testing.T) {
			agent := &models.Agent{EnvironmentVariables: []byte(`["krb5-ktname"]`)}

			// Resending the pre-existing (now-invalid) name alongside a new, valid one must not
			// fail: the caller only intended to add KRB5_CONFIG, and this field is full-replace.
			require.NoError(t, agent.SetEnvironmentVariableNames([]string{"krb5-ktname", "KRB5_CONFIG"}))

			names, err := agent.GetEnvironmentVariableNames()
			require.NoError(t, err)
			assert.Equal(t, []string{"krb5-ktname", "KRB5_CONFIG"}, names)
		})

		t.Run("grandfathers a whitespace-padded legacy name stored before validation existed", func(t *testing.T) {
			agent := &models.Agent{EnvironmentVariables: []byte(`[" krb5-ktname "]`)}

			// The stored name carries whitespace from before any validation trimmed it; resending
			// it trimmed (as any normal caller would) must still match the grandfathered value.
			require.NoError(t, agent.SetEnvironmentVariableNames([]string{"krb5-ktname", "KRB5_CONFIG"}))

			names, err := agent.GetEnvironmentVariableNames()
			require.NoError(t, err)
			assert.Equal(t, []string{"krb5-ktname", "KRB5_CONFIG"}, names)
		})

		t.Run("an undecodable stored value is repaired by sending a new list", func(t *testing.T) {
			agent := &models.Agent{EnvironmentVariables: []byte(`{"not":"an array"}`)}

			// ToAPIAgent reports an undecodable column as empty and documents that sending a new
			// list repairs the row; grandfathering must not turn that into a permanent failure.
			require.NoError(t, agent.SetEnvironmentVariableNames([]string{"KRB5_CONFIG"}))

			names, err := agent.GetEnvironmentVariableNames()
			require.NoError(t, err)
			assert.Equal(t, []string{"KRB5_CONFIG"}, names)
		})

		t.Run("an undecodable stored value grandfathers nothing", func(t *testing.T) {
			agent := &models.Agent{EnvironmentVariables: []byte(`{"not":"an array"}`)}

			err := agent.SetEnvironmentVariableNames([]string{"krb5-ktname"})
			require.Error(t, err)
		})

		t.Run("a blank stored name is not grandfathered", func(t *testing.T) {
			agent := &models.Agent{EnvironmentVariables: []byte(`["   ", "KRB5_KTNAME"]`)}

			// A blank entry written before validation existed must not let the empty string be
			// stored again: pmm-agent cannot resolve it and warns on every state update.
			err := agent.SetEnvironmentVariableNames([]string{"   ", "KRB5_CONFIG"})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "cannot be empty")
		})

		t.Run("still rejects a new invalid name even with a grandfathered one present", func(t *testing.T) {
			agent := &models.Agent{EnvironmentVariables: []byte(`["krb5-ktname"]`)}

			err := agent.SetEnvironmentVariableNames([]string{"krb5-ktname", "also-bad"})
			require.Error(t, err)

			names, err := agent.GetEnvironmentVariableNames()
			require.NoError(t, err)
			assert.Equal(t, []string{"krb5-ktname"}, names, "a rejected update must not overwrite the stored value")
		})
	})

	t.Run("DSN", func(t *testing.T) {
		agent := &models.Agent{
			Username:          new("username"),
			Password:          new("s3cur3 p@$$w0r4."),
			ExporterOptions:   models.ExporterOptions{},
			QANOptions:        models.QANOptions{},
			MongoDBOptions:    models.MongoDBOptions{},
			MySQLOptions:      models.MySQLOptions{},
			PostgreSQLOptions: models.PostgreSQLOptions{},
		}
		service := &models.Service{
			Address: new("1.2.3.4"),
			Port:    new(uint16(12345)),
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
				require.Equal(t, expected, agent.DSN(service, models.DSNParams{DialTimeout: time.Second, Database: "database"}, nil, nil))
			})
		}

		t.Run("MongoDBNoDatabase", func(t *testing.T) {
			agent.AgentType = models.MongoDBExporterType

			require.Equal(t, "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/?connectTimeoutMS=1000&directConnection=true&serverSelectionTimeoutMS=1000", agent.DSN(service, models.DSNParams{DialTimeout: time.Second}, nil, nil))
			require.Equal(t, "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/?directConnection=true", agent.DSN(service, models.DSNParams{}, nil, nil))
		})
	})

	t.Run("DSN socket", func(t *testing.T) {
		agent := &models.Agent{
			Username:        new("username"),
			Password:        new("s3cur3 p@$$w0r4."),
			ExporterOptions: models.ExporterOptions{},
			QANOptions:      models.QANOptions{},
			MySQLOptions:    models.MySQLOptions{},
		}
		service := &models.Service{
			Socket: new("/var/run/mysqld/mysqld.sock"),
		}
		for typ, expected := range map[models.AgentType]string{
			models.MySQLdExporterType:          "username:s3cur3 p@$$w0r4.@unix(/var/run/mysqld/mysqld.sock)/database?timeout=1s",
			models.ProxySQLExporterType:        "username:s3cur3 p@$$w0r4.@unix(/var/run/mysqld/mysqld.sock)/database?timeout=1s",
			models.QANMySQLPerfSchemaAgentType: "username:s3cur3 p@$$w0r4.@unix(/var/run/mysqld/mysqld.sock)/database?clientFoundRows=true&parseTime=true&timeout=1s",
			models.QANMySQLSlowlogAgentType:    "username:s3cur3 p@$$w0r4.@unix(/var/run/mysqld/mysqld.sock)/database?clientFoundRows=true&parseTime=true&timeout=1s",
		} {
			t.Run(string(typ), func(t *testing.T) {
				agent.AgentType = typ
				require.Equal(t, expected, agent.DSN(service, models.DSNParams{DialTimeout: time.Second, Database: "database"}, nil, nil))
			})
		}
	})

	t.Run("DSN timeout", func(t *testing.T) {
		agent := &models.Agent{
			Username:        new("username"),
			Password:        new("s3cur3 p@$$w0r4."),
			ExporterOptions: models.ExporterOptions{},
			QANOptions:      models.QANOptions{},
			MongoDBOptions:  models.MongoDBOptions{},
		}
		service := &models.Service{
			Socket: new("/var/run/mysqld/mysqld.sock"),
		}
		for typ, expected := range map[models.AgentType]string{
			models.MongoDBExporterType:         "mongodb://username:s3cur3%20p%40$$w0r4.@%2Fvar%2Frun%2Fmysqld%2Fmysqld.sock/database?connectTimeoutMS=1000&directConnection=true&serverSelectionTimeoutMS=1000",
			models.QANMongoDBProfilerAgentType: "mongodb://username:s3cur3%20p%40$$w0r4.@%2Fvar%2Frun%2Fmysqld%2Fmysqld.sock/database?connectTimeoutMS=1000&directConnection=true&serverSelectionTimeoutMS=1000",
		} {
			t.Run(string(typ), func(t *testing.T) {
				agent.AgentType = typ
				require.Equal(t, expected, agent.DSN(service, models.DSNParams{DialTimeout: time.Second, Database: "database"}, nil, nil))
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
			Username:          new("username"),
			Password:          new("s3cur3 p@$$w0r4."),
			TLS:               true,
			ExporterOptions:   models.ExporterOptions{},
			MongoDBOptions:    mongoDBOptions,
			MySQLOptions:      mysqlOptions,
			PostgreSQLOptions: postgresqlOptions,
		}
		service := &models.Service{
			Address: new("1.2.3.4"),
			Port:    new(uint16(12345)),
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
				require.Equal(t, expected, agent.DSN(service, models.DSNParams{DialTimeout: time.Second, Database: "database"}, nil, nil))
			})
		}

		t.Run("MongoDBNoDatabase", func(t *testing.T) {
			agent.AgentType = models.MongoDBExporterType
			// reset values from the previous test
			agent.MongoDBOptions.TLSCertificateKeyFilePassword = ""
			agent.MongoDBOptions.AuthenticationMechanism = ""

			require.Equal(t, "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/?connectTimeoutMS=1000&directConnection=true&serverSelectionTimeoutMS=1000&ssl=true&tlsCaFile={{.TextFiles.caFilePlaceholder}}&tlsCertificateKeyFile={{.TextFiles.certificateKeyFilePlaceholder}}", agent.DSN(service, models.DSNParams{DialTimeout: time.Second}, nil, nil))
			require.Equal(t, "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/?directConnection=true&ssl=true&tlsCaFile={{.TextFiles.caFilePlaceholder}}&tlsCertificateKeyFile={{.TextFiles.certificateKeyFilePlaceholder}}", agent.DSN(service, models.DSNParams{}, nil, nil))
			expectedFiles := map[string]string{
				"caFilePlaceholder":             "cert",
				"certificateKeyFilePlaceholder": "key",
			}
			require.Equal(t, expectedFiles, agent.Files())
		})

		t.Run("MongoDB Auth Database", func(t *testing.T) {
			agent.AgentType = models.MongoDBExporterType
			// reset values from the previous test
			agent.MongoDBOptions.TLSCertificateKeyFilePassword = ""
			agent.MongoDBOptions.AuthenticationMechanism = "MONGO-X509"
			agent.MongoDBOptions.AuthenticationDatabase = "$external"

			require.Equal(t, "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/?authMechanism=MONGO-X509&authSource=%24external&connectTimeoutMS=1000&directConnection=true&serverSelectionTimeoutMS=1000&ssl=true&tlsCaFile={{.TextFiles.caFilePlaceholder}}&tlsCertificateKeyFile={{.TextFiles.certificateKeyFilePlaceholder}}", agent.DSN(service, models.DSNParams{DialTimeout: time.Second}, nil, nil))
			require.Equal(t, "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/?authMechanism=MONGO-X509&authSource=%24external&directConnection=true&ssl=true&tlsCaFile={{.TextFiles.caFilePlaceholder}}&tlsCertificateKeyFile={{.TextFiles.certificateKeyFilePlaceholder}}", agent.DSN(service, models.DSNParams{}, nil, nil))
			expectedFiles := map[string]string{
				"caFilePlaceholder":             "cert",
				"certificateKeyFilePlaceholder": "key",
			}
			require.Equal(t, expectedFiles, agent.Files())
		})
	})

	t.Run("DSN ssl-skip-verify", func(t *testing.T) {
		agent := &models.Agent{
			Username:          new("username"),
			Password:          new("s3cur3 p@$$w0r4."),
			TLS:               true,
			TLSSkipVerify:     true,
			ExporterOptions:   models.ExporterOptions{},
			QANOptions:        models.QANOptions{},
			MongoDBOptions:    models.MongoDBOptions{},
			MySQLOptions:      models.MySQLOptions{},
			PostgreSQLOptions: models.PostgreSQLOptions{},
		}
		service := &models.Service{
			Address: new("1.2.3.4"),
			Port:    new(uint16(12345)),
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
				require.Equal(t, expected, agent.DSN(service, models.DSNParams{DialTimeout: time.Second, Database: "database"}, nil, nil))
			})
		}

		t.Run("MongoDBNoDatabase", func(t *testing.T) {
			agent.AgentType = models.MongoDBExporterType

			require.Equal(t, "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/?connectTimeoutMS=1000&directConnection=true&serverSelectionTimeoutMS=1000&ssl=true&tlsInsecure=true", agent.DSN(service, models.DSNParams{DialTimeout: time.Second}, nil, nil))
			require.Equal(t, "mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/?directConnection=true&ssl=true&tlsInsecure=true", agent.DSN(service, models.DSNParams{}, nil, nil))
		})
	})
}

func TestPostgresAgentTLS(t *testing.T) {
	agent := &models.Agent{
		Username:          new("username"),
		Password:          new("s3cur3 p@$$w0r4."),
		AgentType:         models.PostgresExporterType,
		ExporterOptions:   models.ExporterOptions{},
		PostgreSQLOptions: models.PostgreSQLOptions{},
	}
	service := &models.Service{
		Address: new("1.2.3.4"),
		Port:    new(uint16(12345)),
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
			require.Equal(t, testCase.expected, agent.DSN(service, models.DSNParams{DialTimeout: time.Second, Database: "database"}, nil, nil))
		})
		t.Run(fmt.Sprintf("AutodiscoveryLimit set TLS:%v/TLSSkipVerify:%v", testCase.tls, testCase.tlsSkipVerify), func(t *testing.T) {
			agent.TLS = testCase.tls
			agent.TLSSkipVerify = testCase.tlsSkipVerify
			agent.PostgreSQLOptions = models.PostgreSQLOptions{AutoDiscoveryLimit: new(int32(10))}
			assert.Equal(t, testCase.expected, agent.DSN(service, models.DSNParams{DialTimeout: time.Second, Database: "database"}, nil, nil))
		})
	}
}

func TestValkey(t *testing.T) {
	t.Parallel()

	t.Run("Redis DSN", func(t *testing.T) {
		t.Parallel()

		agent := &models.Agent{
			Username:        new("username"),
			Password:        new("s3cur3 p@$$w0r4."),
			AgentType:       models.ValkeyExporterType,
			ExporterOptions: models.ExporterOptions{},
			ValkeyOptions:   models.ValkeyOptions{},
		}
		service := &models.Service{
			Address: new("1.2.3.4"),
			Port:    new(uint16(12345)),
		}

		expected := "redis://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345"

		require.Equal(t, expected, agent.DSN(service, models.DSNParams{DialTimeout: time.Second, Database: "database"}, nil, nil))
	})

	t.Run("Valkey DSN with TLS", func(t *testing.T) {
		t.Parallel()

		agent := &models.Agent{
			Username:        new("username"),
			Password:        new("s3cur3 p@$$w0r4."),
			AgentType:       models.ValkeyExporterType,
			ExporterOptions: models.ExporterOptions{},
			TLS:             true,
			ValkeyOptions: models.ValkeyOptions{
				SSLCa:   "aa",
				SSLCert: "bb",
				SSLKey:  "cc",
			},
		}
		service := &models.Service{
			Address: new("1.2.3.4"),
			Port:    new(uint16(12345)),
		}

		expected := "rediss://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345"

		require.Equal(t, expected, agent.DSN(service, models.DSNParams{DialTimeout: time.Second, Database: "database"}, nil, nil))
	})

	t.Run("Files", func(t *testing.T) {
		t.Parallel()

		all := models.ValkeyOptions{SSLCa: "aa", SSLCert: "bb", SSLKey: "cc"}

		for name, tc := range map[string]struct {
			tls      bool
			options  models.ValkeyOptions
			expected map[string]string
		}{
			"all":              {true, all, map[string]string{"tlsCa": "aa", "tlsCert": "bb", "tlsKey": "cc"}},
			"ca":               {true, models.ValkeyOptions{SSLCa: "aa"}, map[string]string{"tlsCa": "aa"}},
			"pair":             {true, models.ValkeyOptions{SSLCert: "bb", SSLKey: "cc"}, map[string]string{"tlsCert": "bb", "tlsKey": "cc"}},
			"cert without key": {true, models.ValkeyOptions{SSLCa: "aa", SSLCert: "bb"}, map[string]string{"tlsCa": "aa"}},
			"key without cert": {true, models.ValkeyOptions{SSLKey: "cc"}, nil},
			"none":             {true, models.ValkeyOptions{}, nil},
			"tls disabled":     {false, all, nil},
		} {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				agent := models.Agent{AgentType: models.ValkeyExporterType, TLS: tc.tls, ValkeyOptions: tc.options}

				require.Equal(t, tc.expected, agent.Files())
			})
		}
	})

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		for name, tc := range map[string]struct {
			options models.ValkeyOptions
			valid   bool
		}{
			"no material":   {models.ValkeyOptions{}, true},
			"ca only":       {models.ValkeyOptions{SSLCa: "aa"}, true},
			"complete pair": {models.ValkeyOptions{SSLCert: "bb", SSLKey: "cc"}, true},
			"cert only":     {models.ValkeyOptions{SSLCert: "bb"}, false},
			"key only":      {models.ValkeyOptions{SSLKey: "cc"}, false},
		} {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				err := tc.options.Validate()
				if tc.valid {
					require.NoError(t, err)
					return
				}

				require.Error(t, err)
				require.Equal(t, codes.InvalidArgument, status.Code(err))
			})
		}
	})

	t.Run("TemplateDelimiters avoid certificate content", func(t *testing.T) {
		t.Parallel()

		service := &models.Service{ServiceType: models.ValkeyServiceType, Address: new("1.2.3.4")}

		for name, options := range map[string]models.ValkeyOptions{
			"ca":   {SSLCa: "aa {{ bb"},
			"cert": {SSLCert: "aa {{ bb"},
			"key":  {SSLKey: "aa {{ bb"},
		} {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				agent := models.Agent{AgentType: models.ValkeyExporterType, ValkeyOptions: options}

				require.Equal(t, &models.DelimiterPair{Left: "[[", Right: "]]"}, agent.TemplateDelimiters(service))
			})
		}
	})
}

func TestPostgresWithSocket(t *testing.T) {
	t.Run("empty-password", func(t *testing.T) {
		agent := &models.Agent{
			Username:          new("username"),
			AgentType:         models.PostgresExporterType,
			TLS:               true,
			TLSSkipVerify:     false,
			ExporterOptions:   models.ExporterOptions{},
			PostgreSQLOptions: models.PostgreSQLOptions{},
		}
		service := &models.Service{
			Socket: new("/var/run/postgres"),
		}
		expect := "postgres://username@/database?connect_timeout=1&host=%2Fvar%2Frun%2Fpostgres&sslmode=verify-ca"
		require.Equal(t, expect, agent.DSN(service, models.DSNParams{DialTimeout: time.Second, Database: "database"}, nil, nil))
	})

	t.Run("empty-user-password", func(t *testing.T) {
		agent := &models.Agent{
			AgentType:         models.PostgresExporterType,
			ExporterOptions:   models.ExporterOptions{},
			PostgreSQLOptions: models.PostgreSQLOptions{},
		}
		service := &models.Service{
			Socket: new("/var/run/postgres"),
		}
		expect := "postgres:///database?connect_timeout=1&host=%2Fvar%2Frun%2Fpostgres&sslmode=disable"
		require.Equal(t, expect, agent.DSN(service, models.DSNParams{DialTimeout: time.Second, Database: "database"}, nil, nil))
	})

	t.Run("dir-with-symbols", func(t *testing.T) {
		agent := &models.Agent{
			AgentType:         models.PostgresExporterType,
			ExporterOptions:   models.ExporterOptions{},
			PostgreSQLOptions: models.PostgreSQLOptions{},
		}
		service := &models.Service{
			Socket: new(`/tmp/123\ A0m\%\$\@\8\,\+\-`),
		}
		expect := "postgres:///database?connect_timeout=1&host=%2Ftmp%2F123%5C+A0m%5C%25%5C%24%5C%40%5C8%5C%2C%5C%2B%5C-&sslmode=disable"
		require.Equal(t, expect, agent.DSN(service, models.DSNParams{DialTimeout: time.Second, Database: "database"}, nil, nil))
	})
}

func TestMongoWithSocket(t *testing.T) {
	t.Run("empty-password", func(t *testing.T) {
		agent := &models.Agent{
			Username:        new("username"),
			AgentType:       models.MongoDBExporterType,
			TLS:             true,
			TLSSkipVerify:   false,
			ExporterOptions: models.ExporterOptions{},
			MongoDBOptions:  models.MongoDBOptions{},
		}
		service := &models.Service{
			Socket: new("/tmp/mongodb-27017.sock"),
		}
		expect := "mongodb://username@%2Ftmp%2Fmongodb-27017.sock/database?connectTimeoutMS=1000&directConnection=true&serverSelectionTimeoutMS=1000&ssl=true"
		require.Equal(t, expect, agent.DSN(service, models.DSNParams{DialTimeout: time.Second, Database: "database"}, nil, nil))
	})

	t.Run("empty-user-password", func(t *testing.T) {
		agent := &models.Agent{
			AgentType:       models.MongoDBExporterType,
			ExporterOptions: models.ExporterOptions{},
			MongoDBOptions:  models.MongoDBOptions{},
		}
		service := &models.Service{
			Socket: new("/tmp/mongodb-27017.sock"),
		}
		expect := "mongodb://%2Ftmp%2Fmongodb-27017.sock/database?connectTimeoutMS=1000&directConnection=true&serverSelectionTimeoutMS=1000"
		require.Equal(t, expect, agent.DSN(service, models.DSNParams{DialTimeout: time.Second, Database: "database"}, nil, nil))
	})

	t.Run("dir-with-symbols", func(t *testing.T) {
		agent := &models.Agent{
			AgentType:       models.MongoDBExporterType,
			ExporterOptions: models.ExporterOptions{},
			MongoDBOptions:  models.MongoDBOptions{},
		}
		service := &models.Service{
			Socket: new(`/tmp/123\ A0m\%\$\@\8\,\+\-/mongodb-27017.sock`),
		}
		expect := "mongodb://%2Ftmp%2F123%5C%20A0m%5C%25%5C$%5C%40%5C8%5C,%5C+%5C-%2Fmongodb-27017.sock/database?connectTimeoutMS=1000&directConnection=true&serverSelectionTimeoutMS=1000"
		require.Equal(t, expect, agent.DSN(service, models.DSNParams{DialTimeout: time.Second, Database: "database"}, nil, nil))
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

		{new(int32(1000)), -1, false},
		{new(int32(1000)), 0, true},
		{new(int32(1000)), 500, false},
		{new(int32(1000)), 2000, true},
	} {
		c := "nil"
		if testCase.count != nil {
			c = strconv.Itoa(int(*testCase.count))
		}
		t.Run(fmt.Sprintf("Count:%s/Limit:%d", c, testCase.limit), func(t *testing.T) {
			agent := &models.Agent{
				AgentType: models.MySQLdExporterType,
				MySQLOptions: models.MySQLOptions{
					TableCount:                     testCase.count,
					TableCountTablestatsGroupLimit: testCase.limit,
				},
			}
			require.Equal(t, testCase.expected, agent.IsMySQLTablestatsGroupEnabled())
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
				MachineID: new("ExporterNode"),
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
				ServiceID:    new("external"),
				RunsOnNodeID: new("ExporterNodeID"),
				ListenPort:   new(uint16(9121)),
				ExporterOptions: models.ExporterOptions{
					PushMetrics:   true,
					MetricsPath:   "/metrics",
					MetricsScheme: "http",
				},
			},

			&models.Agent{
				AgentID:      "ExporterAgentPull",
				AgentType:    models.ExternalExporterType,
				ServiceID:    new("external"),
				RunsOnNodeID: new("ExporterNodeID"),
				ListenPort:   new(uint16(9121)),
				Username:     new("user"),
				Password:     new("secret"),
				ExporterOptions: models.ExporterOptions{
					PushMetrics:   false,
					MetricsPath:   "/metrics",
					MetricsScheme: "http",
				},
			},

			&models.Agent{
				AgentID:      "ExporterServerless",
				AgentType:    models.ExternalExporterType,
				RunsOnNodeID: new("ExporterServerlessNodeID"),
				ServiceID:    new("redis_exporter-external"),
				ListenPort:   new(uint16(9121)),
				Username:     new("user"),
				Password:     new("secret"),
				ExporterOptions: models.ExporterOptions{
					PushMetrics:   false,
					MetricsPath:   "/metrics",
					MetricsScheme: "http",
				},
			},

			&models.Agent{
				AgentID:      "ExporterServerlessWithQueryParams",
				AgentType:    models.ExternalExporterType,
				RunsOnNodeID: new("ExporterServerlessNodeID2"),
				ServiceID:    new("nomad_exporter-external"),
				ListenPort:   new(uint16(9121)),
				Username:     new("user"),
				Password:     new("secret"),
				ExporterOptions: models.ExporterOptions{
					PushMetrics:   false,
					MetricsPath:   "/metrics?format=prometheus&output=json",
					MetricsScheme: "http",
				},
			},

			&models.Agent{
				AgentID:      "ExporterServerlessWithEmptyMetricsPath",
				AgentType:    models.ExternalExporterType,
				RunsOnNodeID: new("ExporterServerlessNodeID2"),
				ServiceID:    new("nomad_exporter-external"),
				ListenPort:   new(uint16(9121)),
				Username:     new("user"),
				Password:     new("secret"),
				ExporterOptions: models.ExporterOptions{
					PushMetrics:   false,
					MetricsPath:   "/",
					MetricsScheme: "http",
				},
			},
		} {
			if v, ok := str.(*models.Agent); ok {
				encrypted, err := models.EncryptAgent(*v)
				require.NoError(t, err)
				str = new(encrypted)
			}
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
				require.NoError(t, err)
				actual, err := agent.ExporterURL(q)
				require.NoError(t, err)
				require.Equal(t, expected, actual)
			})
		}
	})
}

func TestEffectiveDialTimeout(t *testing.T) {
	t.Parallel()

	custom := 7 * time.Second
	t.Run("explicit ExporterOptions.ConnectionTimeout", func(t *testing.T) {
		t.Parallel()
		a := &models.Agent{
			AgentType: models.QANMySQLPerfSchemaAgentType,
			ExporterOptions: models.ExporterOptions{
				ConnectionTimeout: &custom,
			},
		}
		require.Equal(t, custom, a.EffectiveDialTimeout())
	})

	t.Run("exporters with 2s default", func(t *testing.T) {
		t.Parallel()

		for _, typ := range []models.AgentType{
			models.MySQLdExporterType,
			models.MongoDBExporterType,
			models.PostgresExporterType,
			models.ProxySQLExporterType,
		} {
			t.Run(string(typ), func(t *testing.T) {
				t.Parallel()
				a := &models.Agent{AgentType: typ}
				require.Equal(t, 2*time.Second, a.EffectiveDialTimeout())
			})
		}
	})

	t.Run("valkey_exporter returns 3s default", func(t *testing.T) {
		t.Parallel()
		a := &models.Agent{AgentType: models.ValkeyExporterType}
		require.Equal(t, 3*time.Second, a.EffectiveDialTimeout())
	})

	t.Run("non-exporter agent falls back to 2s", func(t *testing.T) {
		t.Parallel()
		a := &models.Agent{AgentType: models.QANMySQLPerfSchemaAgentType}
		require.Equal(t, 2*time.Second, a.EffectiveDialTimeout())
	})
}

func TestExporterOptionsIsEmpty(t *testing.T) {
	t.Parallel()

	t.Run("nil connection timeout is empty", func(t *testing.T) {
		t.Parallel()

		require.True(t, (models.ExporterOptions{}).IsEmpty())
	})

	t.Run("zero connection timeout is empty", func(t *testing.T) {
		t.Parallel()

		require.True(t, (models.ExporterOptions{
			ConnectionTimeout: pointer.ToDuration(0),
		}).IsEmpty())
	})

	t.Run("non-zero connection timeout is not empty", func(t *testing.T) {
		t.Parallel()

		require.False(t, (models.ExporterOptions{
			ConnectionTimeout: pointer.ToDuration(time.Second),
		}).IsEmpty())
	})
}

func TestAWSOptionsCredentialsKey(t *testing.T) {
	t.Parallel()

	const roleARN = "arn:aws:iam::123456789012:role/pmm-monitoring"

	for _, tc := range []struct {
		name     string
		options  models.AWSOptions
		expected string
	}{{
		name:     "empty options yield empty key",
		options:  models.AWSOptions{},
		expected: "",
	}, {
		name:     "access key is returned verbatim to preserve existing group IDs",
		options:  models.AWSOptions{AWSAccessKey: "AKIAIOSFODNN7EXAMPLE", AWSSecretKey: "secret"},
		expected: "AKIAIOSFODNN7EXAMPLE",
	}, {
		name:     "role ARN is hashed",
		options:  models.AWSOptions{AWSRoleARN: roleARN},
		expected: "role-4c562ef173064add",
	}, {
		name:     "metrics flags alone yield empty key",
		options:  models.AWSOptions{RDSBasicMetricsDisabled: true},
		expected: "",
	}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, tc.options.CredentialsKey())
		})
	}
}

func TestAWSOptionsCredentialsKeyDistinguishesRoles(t *testing.T) {
	t.Parallel()

	a := models.AWSOptions{AWSRoleARN: "arn:aws:iam::111111111111:role/pmm"}
	b := models.AWSOptions{AWSRoleARN: "arn:aws:iam::222222222222:role/pmm"}

	assert.NotEqual(t, a.CredentialsKey(), b.CredentialsKey())
	assert.Equal(t, a.CredentialsKey(), models.AWSOptions{AWSRoleARN: a.AWSRoleARN}.CredentialsKey())
}

func TestAWSOptionsIsEmpty(t *testing.T) {
	t.Parallel()

	assert.True(t, models.AWSOptions{}.IsEmpty())
	assert.False(t, models.AWSOptions{AWSRoleARN: "arn:aws:iam::123456789012:role/pmm"}.IsEmpty())
}

func TestAWSOptionsValidate(t *testing.T) {
	t.Parallel()

	const roleARN = "arn:aws:iam::123456789012:role/pmm-monitoring"

	for _, tc := range []struct {
		name    string
		options models.AWSOptions
		wantErr bool
	}{{
		name:    "empty is valid",
		options: models.AWSOptions{},
	}, {
		name:    "static keys alone are valid",
		options: models.AWSOptions{AWSAccessKey: "AKIA", AWSSecretKey: "secret"},
	}, {
		name:    "role ARN alone is valid",
		options: models.AWSOptions{AWSRoleARN: roleARN},
	}, {
		name:    "role ARN with access key is rejected",
		options: models.AWSOptions{AWSRoleARN: roleARN, AWSAccessKey: "AKIA"},
		wantErr: true,
	}, {
		name:    "role ARN with secret key is rejected",
		options: models.AWSOptions{AWSRoleARN: roleARN, AWSSecretKey: "secret"},
		wantErr: true,
	}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.options.Validate()
			if !tc.wantErr {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Equal(t, codes.InvalidArgument, status.Code(err))
		})
	}
}

func TestDBCredentialsFromVault(t *testing.T) {
	t.Parallel()

	const password = "p@ss/w:rd {{ }}"
	get := func(secret map[string]string) vault.Getter {
		return func(path, key string) (string, error) {
			if path != "secret/data/pmm" {
				return "", errors.New("unexpected path " + path)
			}
			v, ok := secret[key]
			if !ok {
				return "", &vault.KeyNotFoundError{Path: path, Key: key}
			}
			return v, nil
		}
	}
	withUsername := get(map[string]string{"username": "v-pmm", "password": password})
	withoutUsername := get(map[string]string{"password": password})

	service := &models.Service{Address: new("1.2.3.4"), Port: new(uint16(12345))}

	for _, agentType := range []models.AgentType{
		models.MySQLdExporterType,
		models.QANMySQLPerfSchemaAgentType,
		models.ProxySQLExporterType,
		models.PostgresExporterType,
		models.QANPostgreSQLPgStatementsAgentType,
		models.MongoDBExporterType,
		models.QANMongoDBProfilerAgentType,
		models.ValkeyExporterType,
	} {
		t.Run(string(agentType), func(t *testing.T) {
			t.Parallel()

			agent := &models.Agent{AgentType: agentType, Username: new("given"), VaultPath: new("secret/data/pmm")}
			dsn := agent.DSN(service, models.DSNParams{DialTimeout: time.Second, Database: "db"}, nil, nil)
			assert.NotContains(t, dsn, "given", "the username must travel inside its token")

			for _, tc := range []struct {
				name     string
				get      vault.Getter
				username string
			}{
				{"SecretUsernameWins", withUsername, "v-pmm"},
				{"GivenUsernameStandsIn", withoutUsername, "given"},
			} {
				resolved, values, err := vault.ReplaceTokens(dsn, tc.get)
				require.NoError(t, err, tc.name)
				assert.Contains(t, values, password, tc.name)

				var gotUsername, gotPassword string
				switch agentType {
				case models.MySQLdExporterType, models.QANMySQLPerfSchemaAgentType, models.ProxySQLExporterType:
					cfg, err := mysql.ParseDSN(resolved)
					require.NoError(t, err, tc.name)
					gotUsername, gotPassword = cfg.User, cfg.Passwd
				default:
					u, err := url.Parse(resolved)
					require.NoError(t, err, tc.name)
					gotUsername = u.User.Username()
					gotPassword, _ = u.User.Password()
				}
				assert.Equal(t, tc.username, gotUsername, tc.name)
				assert.Equal(t, password, gotPassword, tc.name)
			}
		})
	}

	t.Run("NoVault", func(t *testing.T) {
		t.Parallel()

		agent := &models.Agent{AgentType: models.MySQLdExporterType, Username: new("u"), Password: new("p")}
		username, password := agent.DBCredentials(vault.EncodingRaw)
		assert.Equal(t, "u", username)
		assert.Equal(t, "p", password)
		assert.Equal(t, &models.DBConfig{User: "u", Password: "p", Address: "1.2.3.4", Port: 12345}, agent.DBConfig(service))
	})
}

func TestTLSFromVault(t *testing.T) {
	t.Parallel()

	const keyPassword = "p@ss&w=rd +x"
	secret := map[string]string{"tls_key": "KEY", "tls_ca": "CA", "tls_certificate_key": "CERT+KEY", "tls_certificate_key_file_password": keyPassword}
	get := func(path, key string) (string, error) {
		if path != "secret/data/pmm" {
			return "", errors.New("unexpected path " + path)
		}
		return secret[key], nil
	}
	service := &models.Service{Address: new("1.2.3.4"), Port: new(uint16(12345))}

	t.Run("FilesHoldTokens", func(t *testing.T) {
		t.Parallel()

		for _, agent := range []*models.Agent{
			{AgentType: models.MySQLdExporterType, MySQLOptions: models.MySQLOptions{TLSKey: "vault:secret/data/pmm#tls_key", TLSCa: "CA"}},
			{AgentType: models.PostgresExporterType, PostgreSQLOptions: models.PostgreSQLOptions{SSLKey: "vault:secret/data/pmm#tls_key", SSLCa: "CA"}},
			{AgentType: models.ValkeyExporterType, TLS: true, ValkeyOptions: models.ValkeyOptions{SSLKey: "vault:secret/data/pmm#tls_key", SSLCert: "C", SSLCa: "CA"}},
		} {
			assert.True(t, agent.UsesVault(), agent.AgentType)
			files := agent.Files()
			resolved := make([]string, 0, len(files))
			for _, content := range files {
				v, _, err := vault.ReplaceTokens(content, get)
				require.NoError(t, err)
				resolved = append(resolved, v)
			}
			assert.Contains(t, resolved, "KEY", agent.AgentType)
			assert.Contains(t, resolved, "CA", agent.AgentType)
			for _, content := range files {
				assert.NotContains(t, content, "vault:", agent.AgentType)
			}
		}

		plain := &models.Agent{AgentType: models.MySQLdExporterType, MySQLOptions: models.MySQLOptions{TLSKey: "KEY"}}
		assert.False(t, plain.UsesVault())
		assert.Equal(t, map[string]string{"tlsKey": "KEY"}, plain.Files())
	})

	t.Run("MongoDBKeyAndItsPassword", func(t *testing.T) {
		t.Parallel()

		agent := &models.Agent{AgentType: models.MongoDBExporterType, MongoDBOptions: models.MongoDBOptions{
			TLSCertificateKey:             "vault:secret/data/pmm#tls_certificate_key",
			TLSCertificateKeyFilePassword: "vault:secret/data/pmm#tls_certificate_key_file_password",
		}}
		assert.True(t, agent.UsesVault())

		dsn := agent.DSN(service, models.DSNParams{DialTimeout: time.Second}, nil, nil)
		assert.NotContains(t, dsn, "vault:")
		resolved, values, err := vault.ReplaceTokens(dsn, get)
		require.NoError(t, err)
		assert.Contains(t, values, keyPassword)

		u, err := url.Parse(resolved)
		require.NoError(t, err)
		assert.Equal(t, keyPassword, u.Query().Get("tlsCertificateKeyFilePassword"))

		key, _, err := vault.ReplaceTokens(agent.Files()["certificateKeyFilePlaceholder"], get)
		require.NoError(t, err)
		assert.Equal(t, "CERT+KEY", key)
	})

	t.Run("TakeVaultSecrets", func(t *testing.T) {
		t.Parallel()

		params := &models.CreateAgentParams{
			Username: "pmm", Password: "s3cr3t", VaultPath: "/secret/data/pmm/",
			MySQLOptions: models.MySQLOptions{TLSCa: "CA", TLSKey: "KEY", TLSCert: "vault:secret/data/other#cert"},
		}
		assert.Equal(t, map[string]string{"username": "pmm", "password": "s3cr3t", "tls_ca": "CA", "tls_key": "KEY"}, params.TakeVaultSecrets())
		assert.Empty(t, params.Password)
		assert.Equal(t, "pmm", params.Username, "it stays as the fallback")
		assert.Equal(t, models.MySQLOptions{
			TLSCa:   "vault:secret/data/pmm#tls_ca",
			TLSKey:  "vault:secret/data/pmm#tls_key",
			TLSCert: "vault:secret/data/other#cert",
		}, params.MySQLOptions)

		// Secrets staged in pmm-agent are taken as well: pmm-agent stores them.
		staged := &models.CreateAgentParams{
			Password: "vault:pmm-agent-staged/abc#password", VaultPath: "secret/data/pmm",
			MySQLOptions: models.MySQLOptions{TLSKey: "vault:pmm-agent-staged/abc#tls_key"},
		}
		assert.Equal(t, map[string]string{
			"password": "vault:pmm-agent-staged/abc#password",
			"tls_key":  "vault:pmm-agent-staged/abc#tls_key",
		}, staged.TakeVaultSecrets())
		assert.Empty(t, staged.Password)
		assert.Equal(t, "vault:secret/data/pmm#tls_key", staged.MySQLOptions.TLSKey)

		// Nothing to take, or no Vault path.
		assert.Nil(t, (&models.CreateAgentParams{VaultPath: "secret/data/pmm", Username: "pmm"}).TakeVaultSecrets())
		plain := &models.CreateAgentParams{Password: "p", MySQLOptions: models.MySQLOptions{TLSKey: "KEY"}}
		assert.Nil(t, plain.TakeVaultSecrets())
		assert.Equal(t, "p", plain.Password)
		assert.Equal(t, "KEY", plain.MySQLOptions.TLSKey)
	})
}

type mapSecrets map[string]string

func (m mapSecrets) Get(_ context.Context, path, key string) (string, error) {
	v, ok := m[path+"#"+key]
	if !ok {
		return "", errors.New("no such secret")
	}
	return v, nil
}

// Not parallel: it configures the process-wide Vault client of PMM Server.
func TestServerSecretsFromVault(t *testing.T) {
	stored := mapSecrets{
		"secret/data/pmm#agent_password": "metrics-pw",
		"secret/data/pmm#aws_secret_key": "aws-secret",
		"secret/data/pmm#client_secret":  "azure-secret",
		"secret/data/pmm#exporter":       "exporter-pw",
	}
	secrets.Configure(stored, time.Minute)
	t.Cleanup(func() { secrets.Configure(nil, 0) })

	agent := &models.Agent{
		AgentID:       "agent-id",
		AgentType:     models.RDSExporterType,
		AgentPassword: new("vault:secret/data/pmm#agent_password"),
		AWSOptions:    models.AWSOptions{AWSAccessKey: "AKIA", AWSSecretKey: "vault:secret/data/pmm#aws_secret_key"},
		AzureOptions:  models.AzureOptions{ClientSecret: "vault:secret/data/pmm#client_secret"},
	}
	assert.True(t, agent.HasServerSecrets())
	assert.False(t, agent.UsesVault(), "pmm-agent gets them read")

	resolved, err := agent.WithServerSecrets(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "metrics-pw", resolved.GetAgentPassword())
	assert.Equal(t, "AKIA", resolved.AWSOptions.AWSAccessKey)
	assert.Equal(t, "aws-secret", resolved.AWSOptions.AWSSecretKey)
	assert.Equal(t, "azure-secret", resolved.AzureOptions.ClientSecret)
	assert.Equal(t, "vault:secret/data/pmm#agent_password", *agent.AgentPassword, "the Agent itself is not changed")
	assert.Equal(t, "vault:secret/data/pmm#aws_secret_key", agent.AWSOptions.AWSSecretKey)

	// A rotated secret changes the fingerprint.
	before := resolved.ServerSecretsFingerprint()
	stored["secret/data/pmm#aws_secret_key"] = "rotated"
	resolved, err = agent.WithServerSecrets(t.Context())
	require.NoError(t, err)
	assert.NotEqual(t, before, resolved.ServerSecretsFingerprint())

	t.Run("ExternalExporterPassword", func(t *testing.T) {
		external := &models.Agent{AgentType: models.ExternalExporterType, Username: new("u"), Password: new("vault:secret/data/pmm#exporter")}
		assert.True(t, external.HasServerSecrets())
		assert.False(t, external.UsesVault(), "PMM Server scrapes the exporter with it")

		resolved, err := external.WithServerSecrets(t.Context())
		require.NoError(t, err)
		assert.Equal(t, "exporter-pw", pointer.GetString(resolved.Password))
	})

	t.Run("DatabasePassword", func(t *testing.T) {
		// pmm-agent reads it, as it does for a Vault path, and PMM Server never does.
		db := &models.Agent{AgentType: models.MySQLdExporterType, Username: new("pmm"), Password: new("vault:secret/data/db#password")}
		assert.True(t, db.UsesVault())
		assert.False(t, db.HasServerSecrets())

		_, password := db.DBCredentials(vault.EncodingRaw)
		assert.Equal(t, vault.Token("secret/data/db", "password", vault.EncodingRaw), password)
	})

	t.Run("Unreadable", func(t *testing.T) {
		secrets.Configure(nil, 0)
		t.Cleanup(func() { secrets.Configure(stored, time.Minute) })

		_, err := agent.WithServerSecrets(t.Context())
		require.ErrorIs(t, err, secrets.ErrNotConfigured)
		require.ErrorContains(t, err, "agent_password")
	})
}
