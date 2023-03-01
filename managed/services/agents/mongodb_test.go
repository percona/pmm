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

package agents

import (
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/version"
)

func TestMongodbExporterConfig225(t *testing.T) {
	pmmAgentVersion := version.MustParse("2.25.0")

	mongodb := &models.Service{
		Address: pointer.ToString("1.2.3.4"),
		Port:    pointer.ToUint16(27017),
	}
	exporter := &models.Agent{
		AgentID:       "agent-id",
		AgentType:     models.MongoDBExporterType,
		Username:      pointer.ToString("username"),
		Password:      pointer.ToString("s3cur3 p@$$w0r4."),
		AgentPassword: pointer.ToString("agent-password"),
	}
	actual, err := mongodbExporterConfig(mongodb, exporter, redactSecrets, pmmAgentVersion)
	expected := &agentpb.SetStateRequest_AgentProcess{
		Type:               inventorypb.AgentType_MONGODB_EXPORTER,
		TemplateLeftDelim:  "{{",
		TemplateRightDelim: "}}",
		Args: []string{
			"--compatible-mode",
			"--discovering-mode",
			"--mongodb.global-conn-pool",
			"--web.listen-address=:{{ .listen_port }}",
		},
		Env: []string{
			"MONGODB_URI=mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:27017/?connectTimeoutMS=1000&serverSelectionTimeoutMS=1000",
			"HTTP_AUTH=pmm:agent-password",
		},
		RedactWords: []string{"s3cur3 p@$$w0r4.", "agent-password"},
	}
	require.NoError(t, err)
	requireNoDuplicateFlags(t, actual.Args)
	require.Equal(t, expected.Args, actual.Args)
	require.Equal(t, expected.Env, actual.Env)
	require.Equal(t, expected, actual)

	t.Run("Having collstats limit", func(t *testing.T) {
		exporter.MongoDBOptions = &models.MongoDBOptions{
			StatsCollections: []string{"col1", "col2", "col3"},
			CollectionsLimit: 79014,
		}
		expected.Args = []string{
			"--collector.collstats-limit=79014",
			"--compatible-mode",
			"--discovering-mode",
			"--mongodb.collstats-colls=col1,col2,col3",
			"--mongodb.global-conn-pool",
			"--web.listen-address=:{{ .listen_port }}",
		}
		actual, err := mongodbExporterConfig(mongodb, exporter, exposeSecrets, pmmAgentVersion)
		require.NoError(t, err)
		require.Equal(t, expected.Args, actual.Args)
	})
}

func TestMongodbExporterConfig226(t *testing.T) {
	pmmAgentVersion := version.MustParse("2.26.0")

	mongodb := &models.Service{
		Address: pointer.ToString("1.2.3.4"),
		Port:    pointer.ToUint16(27017),
	}
	exporter := &models.Agent{
		AgentID:       "agent-id",
		AgentType:     models.MongoDBExporterType,
		Username:      pointer.ToString("username"),
		Password:      pointer.ToString("s3cur3 p@$$w0r4."),
		AgentPassword: pointer.ToString("agent-password"),
	}
	actual, err := mongodbExporterConfig(mongodb, exporter, redactSecrets, pmmAgentVersion)
	expected := &agentpb.SetStateRequest_AgentProcess{
		Type:               inventorypb.AgentType_MONGODB_EXPORTER,
		TemplateLeftDelim:  "{{",
		TemplateRightDelim: "}}",
		Args: []string{
			"--collector.diagnosticdata",
			"--collector.replicasetstatus",
			"--compatible-mode",
			"--discovering-mode",
			"--mongodb.global-conn-pool",
			"--web.listen-address=:{{ .listen_port }}",
		},
		Env: []string{
			"MONGODB_URI=mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:27017/?connectTimeoutMS=1000&serverSelectionTimeoutMS=1000",
			"HTTP_AUTH=pmm:agent-password",
		},
		RedactWords: []string{"s3cur3 p@$$w0r4.", "agent-password"},
	}
	require.NoError(t, err)
	requireNoDuplicateFlags(t, actual.Args)
	require.Equal(t, expected.Args, actual.Args)
	require.Equal(t, expected.Env, actual.Env)
	require.Equal(t, expected, actual)

	t.Run("Having collstats limit", func(t *testing.T) {
		exporter.MongoDBOptions = &models.MongoDBOptions{
			StatsCollections: []string{"col1", "col2", "col3"},
			CollectionsLimit: 79014,
		}
		expected.Args = []string{
			"--collector.collstats-limit=79014",
			"--collector.diagnosticdata",
			"--collector.replicasetstatus",
			"--compatible-mode",
			"--discovering-mode",
			"--mongodb.collstats-colls=col1,col2,col3",
			"--mongodb.global-conn-pool",
			"--mongodb.indexstats-colls=col1,col2,col3",
			"--web.listen-address=:{{ .listen_port }}",
		}
		actual, err := mongodbExporterConfig(mongodb, exporter, exposeSecrets, pmmAgentVersion)
		require.NoError(t, err)
		require.Equal(t, expected.Args, actual.Args)
	})

	t.Run("Enabling all collectors with non zero limit", func(t *testing.T) {
		exporter.MongoDBOptions = &models.MongoDBOptions{
			StatsCollections:    []string{"col1", "col2", "col3"},
			CollectionsLimit:    79014,
			EnableAllCollectors: true,
		}

		expected.Args = []string{
			"--collector.collstats",
			"--collector.collstats-limit=79014",
			"--collector.dbstats",
			"--collector.diagnosticdata",
			"--collector.indexstats",
			"--collector.replicasetstatus",
			"--collector.topmetrics",
			"--compatible-mode",
			"--discovering-mode",
			"--mongodb.collstats-colls=col1,col2,col3",
			"--mongodb.global-conn-pool",
			"--mongodb.indexstats-colls=col1,col2,col3",
			"--web.listen-address=:{{ .listen_port }}",
		}
		actual, err := mongodbExporterConfig(mongodb, exporter, exposeSecrets, pmmAgentVersion)
		require.NoError(t, err)
		require.Equal(t, expected.Args, actual.Args)
	})

	t.Run("Enabling all collectors", func(t *testing.T) {
		exporter.MongoDBOptions = &models.MongoDBOptions{
			EnableAllCollectors: true,
			StatsCollections:    []string{"db1.col1.one", "db2.col2", "db3"},
		}

		expected.Args = []string{
			"--collector.collstats",
			"--collector.collstats-limit=0",
			"--collector.dbstats",
			"--collector.diagnosticdata",
			"--collector.indexstats",
			"--collector.replicasetstatus",
			"--collector.topmetrics",
			"--compatible-mode",
			"--discovering-mode",
			// this should be here even if limit=0 because it could be used to filter dbstats
			// since dbstats is not depending the number of collections present in the db.
			"--mongodb.collstats-colls=db1.col1.one,db2.col2,db3",
			"--mongodb.global-conn-pool",
			"--mongodb.indexstats-colls=db1.col1.one,db2.col2,db3",
			"--web.listen-address=:{{ .listen_port }}",
		}
		actual, err := mongodbExporterConfig(mongodb, exporter, exposeSecrets, pmmAgentVersion)
		require.NoError(t, err)
		require.Equal(t, expected.Args, actual.Args)
	})

	t.Run("collstats-limit=-1 -> automatically set the limit", func(t *testing.T) {
		exporter.MongoDBOptions = &models.MongoDBOptions{
			EnableAllCollectors: true,
			StatsCollections:    []string{"db1.col1.one", "db2.col2", "db3"},
			CollectionsLimit:    -1,
		}

		expected.Args = []string{
			"--collector.collstats",
			"--collector.collstats-limit=200", // 200 is the default for auto-set
			"--collector.dbstats",
			"--collector.diagnosticdata",
			"--collector.indexstats",
			"--collector.replicasetstatus",
			"--collector.topmetrics",
			"--compatible-mode",
			"--discovering-mode",
			"--mongodb.collstats-colls=db1.col1.one,db2.col2,db3",
			"--mongodb.global-conn-pool",
			"--mongodb.indexstats-colls=db1.col1.one,db2.col2,db3",
			"--web.listen-address=:{{ .listen_port }}",
		}
		actual, err := mongodbExporterConfig(mongodb, exporter, exposeSecrets, pmmAgentVersion)
		require.NoError(t, err)
		require.Equal(t, expected.Args, actual.Args)
	})
}

func TestMongodbExporterConfig(t *testing.T) {
	pmmAgentVersion := version.MustParse("2.0.0")

	mongodb := &models.Service{
		Address: pointer.ToString("1.2.3.4"),
		Port:    pointer.ToUint16(27017),
	}
	exporter := &models.Agent{
		AgentID:       "agent-id",
		AgentType:     models.MongoDBExporterType,
		Username:      pointer.ToString("username"),
		Password:      pointer.ToString("s3cur3 p@$$w0r4."),
		AgentPassword: pointer.ToString("agent-password"),
	}
	actual, err := mongodbExporterConfig(mongodb, exporter, redactSecrets, pmmAgentVersion)
	expected := &agentpb.SetStateRequest_AgentProcess{
		Type:               inventorypb.AgentType_MONGODB_EXPORTER,
		TemplateLeftDelim:  "{{",
		TemplateRightDelim: "}}",
		Args: []string{
			"--collect.collection",
			"--collect.database",
			"--collect.topmetrics",
			"--no-collect.connpoolstats",
			"--no-collect.indexusage",
			"--web.listen-address=:{{ .listen_port }}",
		},
		Env: []string{
			"MONGODB_URI=mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:27017/?connectTimeoutMS=1000&serverSelectionTimeoutMS=1000",
			"HTTP_AUTH=pmm:agent-password",
		},
		RedactWords: []string{"s3cur3 p@$$w0r4.", "agent-password"},
	}
	require.NoError(t, err)
	requireNoDuplicateFlags(t, actual.Args)
	require.Equal(t, expected.Args, actual.Args)
	require.Equal(t, expected.Env, actual.Env)
	require.Equal(t, expected, actual)

	t.Run("EmptyPassword", func(t *testing.T) {
		exporter.Password = nil
		actual, err := mongodbExporterConfig(mongodb, exporter, exposeSecrets, pmmAgentVersion)
		require.NoError(t, err)
		assert.Equal(t, "MONGODB_URI=mongodb://username@1.2.3.4:27017/?connectTimeoutMS=1000&serverSelectionTimeoutMS=1000", actual.Env[0])
	})

	t.Run("EmptyUsername", func(t *testing.T) {
		exporter.Username = nil
		actual, err := mongodbExporterConfig(mongodb, exporter, exposeSecrets, pmmAgentVersion)
		require.NoError(t, err)
		assert.Equal(t, "MONGODB_URI=mongodb://1.2.3.4:27017/?connectTimeoutMS=1000&serverSelectionTimeoutMS=1000", actual.Env[0])
	})
	t.Run("SSLEnabled", func(t *testing.T) {
		exporter.TLS = true
		exporter.MongoDBOptions = &models.MongoDBOptions{
			TLSCertificateKey:             "content-of-tls-certificate-key",
			TLSCertificateKeyFilePassword: "passwordoftls",
			TLSCa:                         "content-of-tls-ca",
		}
		actual, err := mongodbExporterConfig(mongodb, exporter, exposeSecrets, pmmAgentVersion)
		expected := "MONGODB_URI=mongodb://1.2.3.4:27017/?connectTimeoutMS=1000&serverSelectionTimeoutMS=1000&ssl=true&" +
			"tlsCaFile={{.TextFiles.caFilePlaceholder}}&tlsCertificateKeyFile={{.TextFiles.certificateKeyFilePlaceholder}}&tlsCertificateKeyFilePassword=passwordoftls"
		assert.Equal(t, expected, actual.Env[0])
		expectedFiles := map[string]string{
			"certificateKeyFilePlaceholder": exporter.MongoDBOptions.TLSCertificateKey,
			"caFilePlaceholder":             exporter.MongoDBOptions.TLSCa,
		}
		require.NoError(t, err)
		assert.Equal(t, expectedFiles, actual.TextFiles)
	})

	t.Run("AuthenticationDatabase", func(t *testing.T) {
		exporter.TLS = true
		exporter.MongoDBOptions = &models.MongoDBOptions{
			TLSCertificateKey:             "content-of-tls-certificate-key",
			TLSCertificateKeyFilePassword: "passwordoftls",
			TLSCa:                         "content-of-tls-ca",
			AuthenticationMechanism:       "MONGODB-X509",
			AuthenticationDatabase:        "$external",
		}
		actual, err := mongodbExporterConfig(mongodb, exporter, exposeSecrets, pmmAgentVersion)
		expected := `MONGODB_URI=mongodb://1.2.3.4:27017/$external?authMechanism=MONGODB-X509` +
			`&authSource=%24external&connectTimeoutMS=1000&serverSelectionTimeoutMS=1000&ssl=true` +
			`&tlsCaFile={{.TextFiles.caFilePlaceholder}}` +
			`&tlsCertificateKeyFile={{.TextFiles.certificateKeyFilePlaceholder}}` +
			`&tlsCertificateKeyFilePassword=passwordoftls`
		assert.Equal(t, expected, actual.Env[0])
		expectedFiles := map[string]string{
			"certificateKeyFilePlaceholder": exporter.MongoDBOptions.TLSCertificateKey,
			"caFilePlaceholder":             exporter.MongoDBOptions.TLSCa,
		}
		require.NoError(t, err)
		assert.Equal(t, expectedFiles, actual.TextFiles)
	})

	t.Run("DisabledCollectors", func(t *testing.T) {
		exporter.DisabledCollectors = []string{"topmetrics"}
		actual, err := mongodbExporterConfig(mongodb, exporter, exposeSecrets, pmmAgentVersion)
		expected := &agentpb.SetStateRequest_AgentProcess{
			Type:               inventorypb.AgentType_MONGODB_EXPORTER,
			TemplateLeftDelim:  "{{",
			TemplateRightDelim: "}}",
			Args: []string{
				"--collect.collection",
				"--collect.database",
				"--no-collect.connpoolstats",
				"--no-collect.indexusage",
				"--web.listen-address=:{{ .listen_port }}",
			},
		}
		require.NoError(t, err)
		requireNoDuplicateFlags(t, actual.Args)
		require.Equal(t, expected.Args, actual.Args)
	})
}

func TestNewMongodbExporterConfig(t *testing.T) {
	pmmAgentVersion := version.MustParse("2.10.0")

	mongodb := &models.Service{
		Address: pointer.ToString("1.2.3.4"),
		Port:    pointer.ToUint16(27017),
	}
	exporter := &models.Agent{
		AgentID:   "agent-id",
		AgentType: models.MongoDBExporterType,
		Username:  pointer.ToString("username"),
		Password:  pointer.ToString("s3cur3 p@$$w0r4."),
	}
	actual, err := mongodbExporterConfig(mongodb, exporter, redactSecrets, pmmAgentVersion)
	require.NoError(t, err)
	expected := &agentpb.SetStateRequest_AgentProcess{
		Type:               inventorypb.AgentType_MONGODB_EXPORTER,
		TemplateLeftDelim:  "{{",
		TemplateRightDelim: "}}",
		Args: []string{
			"--compatible-mode",
			"--mongodb.global-conn-pool",
			"--web.listen-address=:{{ .listen_port }}",
		},
		Env: []string{
			"MONGODB_URI=mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:27017/?connectTimeoutMS=1000&serverSelectionTimeoutMS=1000",
			"HTTP_AUTH=pmm:agent-id",
		},
		RedactWords: []string{"s3cur3 p@$$w0r4."},
	}
	requireNoDuplicateFlags(t, actual.Args)
	require.Equal(t, expected.Args, actual.Args)
	require.Equal(t, expected.Env, actual.Env)
	require.Equal(t, expected, actual)

	t.Run("EmptyPassword", func(t *testing.T) {
		exporter.Password = nil
		actual, err := mongodbExporterConfig(mongodb, exporter, exposeSecrets, pmmAgentVersion)
		require.NoError(t, err)
		assert.Equal(t, "MONGODB_URI=mongodb://username@1.2.3.4:27017/?connectTimeoutMS=1000&serverSelectionTimeoutMS=1000", actual.Env[0])
	})

	t.Run("EmptyUsername", func(t *testing.T) {
		exporter.Username = nil
		actual, err := mongodbExporterConfig(mongodb, exporter, exposeSecrets, pmmAgentVersion)
		require.NoError(t, err)
		assert.Equal(t, "MONGODB_URI=mongodb://1.2.3.4:27017/?connectTimeoutMS=1000&serverSelectionTimeoutMS=1000", actual.Env[0])
	})
}

func TestMongodbExporterConfig228_WebConfigAuth(t *testing.T) {
	t.Parallel()

	pmmAgentVersion := version.MustParse("2.28.0")

	mongodb := &models.Service{
		Address: pointer.ToString("1.2.3.4"),
		Port:    pointer.ToUint16(27017),
	}
	exporter := &models.Agent{
		AgentID:   "agent-id",
		AgentType: models.MongoDBExporterType,
		Username:  pointer.ToString("username"),
		Password:  pointer.ToString("s3cur3 p@$$w0r4."),
	}

	expectedArgs := []string{
		"--collector.diagnosticdata",
		"--collector.replicasetstatus",
		"--compatible-mode",
		"--discovering-mode",
		"--mongodb.global-conn-pool",
		"--web.listen-address=:{{ .listen_port }}",
		"--web.config={{ .TextFiles.webConfigPlaceholder }}",
	}

	expectedEnv := []string{
		"MONGODB_URI=mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:27017/?connectTimeoutMS=1000&serverSelectionTimeoutMS=1000",
	}

	t.Run("Custom_Password", func(t *testing.T) {
		t.Parallel()

		localExporter := &models.Agent{
			AgentID:       exporter.AgentID,
			AgentType:     exporter.AgentType,
			Username:      exporter.Username,
			Password:      exporter.Password,
			AgentPassword: pointer.ToString("agent-custom-password"),
		}
		actual, err := mongodbExporterConfig(mongodb, localExporter, redactSecrets, pmmAgentVersion)

		expected := &agentpb.SetStateRequest_AgentProcess{
			Type:               inventorypb.AgentType_MONGODB_EXPORTER,
			TemplateLeftDelim:  "{{",
			TemplateRightDelim: "}}",
			Args:               expectedArgs,
			TextFiles: map[string]string{
				"webConfigPlaceholder": "basic_auth_users:\n    pmm: agent-custom-password\n",
			},
			Env:         expectedEnv,
			RedactWords: []string{"s3cur3 p@$$w0r4.", "agent-custom-password"},
		}

		require.NoError(t, err)
		requireNoDuplicateFlags(t, actual.Args)
		require.Equal(t, expected, actual)
	})

	t.Run("Default_Password", func(t *testing.T) {
		t.Parallel()

		localExporter := &models.Agent{
			AgentID:   exporter.AgentID,
			AgentType: exporter.AgentType,
			Username:  exporter.Username,
			Password:  exporter.Password,
		}
		actual, err := mongodbExporterConfig(mongodb, localExporter, redactSecrets, pmmAgentVersion)

		expected := &agentpb.SetStateRequest_AgentProcess{
			Type:               inventorypb.AgentType_MONGODB_EXPORTER,
			TemplateLeftDelim:  "{{",
			TemplateRightDelim: "}}",
			Args:               expectedArgs,
			TextFiles: map[string]string{
				"webConfigPlaceholder": "basic_auth_users:\n    pmm: agent-id\n",
			},
			Env:         expectedEnv,
			RedactWords: []string{"s3cur3 p@$$w0r4."},
		}

		require.NoError(t, err)
		requireNoDuplicateFlags(t, actual.Args)
		require.Equal(t, expected, actual)
	})
}
