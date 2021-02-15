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

package agents

import (
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm-managed/models"
)

func TestMongodbExporterConfig(t *testing.T) {
	pmmAgentVersion := version.MustParse("2.0.0")

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
	actual := mongodbExporterConfig(mongodb, exporter, redactSecrets, pmmAgentVersion)
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
			"MONGODB_URI=mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:27017/?connectTimeoutMS=1000",
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
		actual := mongodbExporterConfig(mongodb, exporter, exposeSecrets, pmmAgentVersion)
		assert.Equal(t, "MONGODB_URI=mongodb://username@1.2.3.4:27017/?connectTimeoutMS=1000", actual.Env[0])
	})

	t.Run("EmptyUsername", func(t *testing.T) {
		exporter.Username = nil
		actual := mongodbExporterConfig(mongodb, exporter, exposeSecrets, pmmAgentVersion)
		assert.Equal(t, "MONGODB_URI=mongodb://1.2.3.4:27017/?connectTimeoutMS=1000", actual.Env[0])
	})
	t.Run("SSLEnabled", func(t *testing.T) {
		exporter.TLS = true
		exporter.MongoDBOptions = &models.MongoDBOptions{
			TLSCertificateKey:             "content-of-tls-certificate-key",
			TLSCertificateKeyFilePassword: "passwordoftls",
			TLSCa:                         "content-of-tls-ca",
		}
		actual := mongodbExporterConfig(mongodb, exporter, exposeSecrets, pmmAgentVersion)
		expected := "MONGODB_URI=mongodb://1.2.3.4:27017/?connectTimeoutMS=1000&ssl=true&" +
			"tlsCaFile={{.TextFiles.caFilePlaceholder}}&tlsCertificateKeyFile={{.TextFiles.certificateKeyFilePlaceholder}}&tlsCertificateKeyFilePassword=passwordoftls"
		assert.Equal(t, expected, actual.Env[0])
		expectedFiles := map[string]string{
			"certificateKeyFilePlaceholder": exporter.MongoDBOptions.TLSCertificateKey,
			"caFilePlaceholder":             exporter.MongoDBOptions.TLSCa,
		}
		assert.Equal(t, expectedFiles, actual.TextFiles)
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
	actual := mongodbExporterConfig(mongodb, exporter, redactSecrets, pmmAgentVersion)
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
			"MONGODB_URI=mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:27017/?connectTimeoutMS=1000",
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
		actual := mongodbExporterConfig(mongodb, exporter, exposeSecrets, pmmAgentVersion)
		assert.Equal(t, "MONGODB_URI=mongodb://username@1.2.3.4:27017/?connectTimeoutMS=1000", actual.Env[0])
	})

	t.Run("EmptyUsername", func(t *testing.T) {
		exporter.Username = nil
		actual := mongodbExporterConfig(mongodb, exporter, exposeSecrets, pmmAgentVersion)
		assert.Equal(t, "MONGODB_URI=mongodb://1.2.3.4:27017/?connectTimeoutMS=1000", actual.Env[0])
	})
}
