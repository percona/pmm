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

package agents

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/version"
)

func TestValkeyExporterConfig(t *testing.T) {
	t.Parallel()

	pmmAgentVersion := version.MustParse("2.44.0")
	node := &models.Node{Address: "1.2.3.4"}
	service := &models.Service{
		ServiceType: models.ValkeyServiceType,
		Address:     new("1.2.3.4"),
		Port:        new(uint16(6379)),
	}

	t.Run("DefaultTimeoutUsesFlag", func(t *testing.T) {
		t.Parallel()
		exporter := &models.Agent{
			AgentID:   "agent-id",
			AgentType: models.ValkeyExporterType,
			Username:  new("username"),
			Password:  new("secret"),
		}
		actual := valkeyExporterConfig(node, service, exporter, redactSecrets, pmmAgentVersion)
		expected := &agentv1.SetStateRequest_AgentProcess{
			Type:               inventoryv1.AgentType_AGENT_TYPE_VALKEY_EXPORTER,
			TemplateLeftDelim:  "{{",
			TemplateRightDelim: "}}",
			Args: []string{
				"--connection-timeout=3s",
				"--include-config-metrics",
				"--include-system-metrics",
				"--redis.addr=redis://username:secret@1.2.3.4:6379",
				"--web.listen-address=0.0.0.0:{{ .listen_port }}",
			},
			RedactWords: []string{"secret"},
		}
		require.Equal(t, expected, actual)
	})

	t.Run("CustomTimeoutUsesFlag", func(t *testing.T) {
		t.Parallel()
		exporter := &models.Agent{
			AgentID:   "agent-id",
			AgentType: models.ValkeyExporterType,
			Username:  new("username"),
			Password:  new("secret"),
		}
		exporter.ExporterOptions.ConnectionTimeout = new(1500 * time.Millisecond)

		actual := valkeyExporterConfig(node, service, exporter, redactSecrets, pmmAgentVersion)
		require.Contains(t, actual.Args, "--connection-timeout=1.5s")
		require.Contains(t, actual.Args, "--redis.addr=redis://username:secret@1.2.3.4:6379")
	})

	// valkey_exporter only knows --log-level. Passing --log.level made it print
	// its usage, exit with code 2 and land the agent in the DONE state.
	t.Run("LogLevel", func(t *testing.T) {
		t.Parallel()

		for name, tc := range map[string]struct {
			logLevel string
			expected string
		}{
			"info": {"info", "--log-level=info"},
			// valkey_exporter has no fatal level and silently falls back to info.
			"fatal": {"fatal", "--log-level=error"},
		} {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				exporter := &models.Agent{
					AgentID:   "agent-id",
					AgentType: models.ValkeyExporterType,
					LogLevel:  new(tc.logLevel),
				}

				actual := valkeyExporterConfig(node, service, exporter, redactSecrets, pmmAgentVersion)
				require.Contains(t, actual.Args, tc.expected)
				require.NotContains(t, strings.Join(actual.Args, " "), "--log.level")
			})
		}
	})

	t.Run("TLS", func(t *testing.T) {
		t.Parallel()

		// ValkeyOptions.TLS is deliberately left unset: the exporter arguments and the DSN scheme
		// both key off Agent.TLS, and the two flags must not drift apart.
		type exporterFixture struct {
			tls        bool
			skipVerify bool
			valkey     models.ValkeyOptions
		}

		newExporter := func(f exporterFixture) *models.Agent {
			return &models.Agent{
				AgentID:       "agent-id",
				AgentType:     models.ValkeyExporterType,
				Username:      new("username"),
				Password:      new("secret"),
				TLS:           f.tls,
				TLSSkipVerify: f.skipVerify,
				ValkeyOptions: f.valkey,
			}
		}

		allCertificates := models.ValkeyOptions{SSLCa: "ca-pem", SSLCert: "cert-pem", SSLKey: "key-pem"}

		requireNoCertificateArgs := func(t *testing.T, args []string) {
			t.Helper()
			for _, arg := range args {
				require.False(t, strings.HasPrefix(arg, "--tls-"), "unexpected argument %q", arg)
			}
		}

		requireNoTLSArgs := func(t *testing.T, args []string) {
			t.Helper()
			requireNoCertificateArgs(t, args)
			require.NotContains(t, args, "--skip-tls-verification")
		}

		t.Run("MutualTLS", func(t *testing.T) {
			t.Parallel()

			actual := valkeyExporterConfig(node, service, newExporter(exporterFixture{tls: true, valkey: allCertificates}), redactSecrets, pmmAgentVersion)
			expected := &agentv1.SetStateRequest_AgentProcess{
				Type:               inventoryv1.AgentType_AGENT_TYPE_VALKEY_EXPORTER,
				TemplateLeftDelim:  "{{",
				TemplateRightDelim: "}}",
				Args: []string{
					"--connection-timeout=3s",
					"--include-config-metrics",
					"--include-system-metrics",
					"--redis.addr=rediss://username:secret@1.2.3.4:6379",
					"--tls-ca-cert-file={{ .TextFiles.tlsCa }}",
					"--tls-client-cert-file={{ .TextFiles.tlsCert }}",
					"--tls-client-key-file={{ .TextFiles.tlsKey }}",
					"--web.listen-address=0.0.0.0:{{ .listen_port }}",
				},
				TextFiles: map[string]string{
					"tlsCa":   "ca-pem",
					"tlsCert": "cert-pem",
					"tlsKey":  "key-pem",
				},
				RedactWords: []string{"secret", "key-pem"},
			}
			requireNoDuplicateFlags(t, actual.Args)
			require.Equal(t, expected, actual)
		})

		t.Run("SkipVerifyWithoutCertificates", func(t *testing.T) {
			t.Parallel()

			actual := valkeyExporterConfig(node, service, newExporter(exporterFixture{tls: true, skipVerify: true}), redactSecrets, pmmAgentVersion)
			require.Contains(t, actual.Args, "--skip-tls-verification")
			require.Contains(t, actual.Args, "--redis.addr=rediss://username:secret@1.2.3.4:6379")
			require.Nil(t, actual.TextFiles)
			requireNoCertificateArgs(t, actual.Args)
		})

		t.Run("SkipVerifyWithCertificates", func(t *testing.T) {
			t.Parallel()

			actual := valkeyExporterConfig(node, service, newExporter(exporterFixture{tls: true, skipVerify: true, valkey: allCertificates}), redactSecrets, pmmAgentVersion)
			require.Contains(t, actual.Args, "--skip-tls-verification")
			require.Contains(t, actual.Args, "--tls-ca-cert-file={{ .TextFiles.tlsCa }}")
			require.Contains(t, actual.Args, "--tls-client-cert-file={{ .TextFiles.tlsCert }}")
			require.Contains(t, actual.Args, "--tls-client-key-file={{ .TextFiles.tlsKey }}")
		})

		t.Run("CertificateAuthorityOnly", func(t *testing.T) {
			t.Parallel()

			actual := valkeyExporterConfig(node, service, newExporter(exporterFixture{tls: true, valkey: models.ValkeyOptions{SSLCa: "ca-pem"}}), redactSecrets, pmmAgentVersion)
			require.Contains(t, actual.Args, "--tls-ca-cert-file={{ .TextFiles.tlsCa }}")
			require.NotContains(t, actual.Args, "--tls-client-cert-file={{ .TextFiles.tlsCert }}")
			require.NotContains(t, actual.Args, "--tls-client-key-file={{ .TextFiles.tlsKey }}")
			require.Equal(t, map[string]string{"tlsCa": "ca-pem"}, actual.TextFiles)
		})

		t.Run("ClientCertificateWithoutCertificateAuthority", func(t *testing.T) {
			t.Parallel()

			options := models.ValkeyOptions{SSLCert: "cert-pem", SSLKey: "key-pem"}
			actual := valkeyExporterConfig(node, service, newExporter(exporterFixture{tls: true, valkey: options}), redactSecrets, pmmAgentVersion)
			require.NotContains(t, actual.Args, "--tls-ca-cert-file={{ .TextFiles.tlsCa }}")
			require.Contains(t, actual.Args, "--tls-client-cert-file={{ .TextFiles.tlsCert }}")
			require.Contains(t, actual.Args, "--tls-client-key-file={{ .TextFiles.tlsKey }}")
		})

		// An incomplete key pair is rejected by the exporter, not silently completed here.
		t.Run("ClientCertificateWithoutKey", func(t *testing.T) {
			t.Parallel()

			actual := valkeyExporterConfig(node, service, newExporter(exporterFixture{tls: true, valkey: models.ValkeyOptions{SSLCert: "cert-pem"}}), redactSecrets, pmmAgentVersion)
			require.Contains(t, actual.Args, "--tls-client-cert-file={{ .TextFiles.tlsCert }}")
			require.NotContains(t, actual.Args, "--tls-client-key-file={{ .TextFiles.tlsKey }}")
			require.Equal(t, map[string]string{"tlsCert": "cert-pem"}, actual.TextFiles)
		})

		t.Run("PrivateKeyWithoutCertificate", func(t *testing.T) {
			t.Parallel()

			actual := valkeyExporterConfig(node, service, newExporter(exporterFixture{tls: true, valkey: models.ValkeyOptions{SSLKey: "key-pem"}}), redactSecrets, pmmAgentVersion)
			require.Contains(t, actual.Args, "--tls-client-key-file={{ .TextFiles.tlsKey }}")
			require.NotContains(t, actual.Args, "--tls-client-cert-file={{ .TextFiles.tlsCert }}")
			require.Equal(t, map[string]string{"tlsKey": "key-pem"}, actual.TextFiles)
		})

		// The files still reach the host, but nothing must point the exporter at them over a plaintext link.
		t.Run("CertificatesIgnoredWhenTLSDisabled", func(t *testing.T) {
			t.Parallel()

			actual := valkeyExporterConfig(node, service, newExporter(exporterFixture{valkey: allCertificates}), redactSecrets, pmmAgentVersion)
			requireNoTLSArgs(t, actual.Args)
			require.Contains(t, actual.Args, "--redis.addr=redis://username:secret@1.2.3.4:6379")
			require.Equal(t, map[string]string{"tlsCa": "ca-pem", "tlsCert": "cert-pem", "tlsKey": "key-pem"}, actual.TextFiles)
		})

		t.Run("SkipVerifyIgnoredWhenTLSDisabled", func(t *testing.T) {
			t.Parallel()

			actual := valkeyExporterConfig(node, service, newExporter(exporterFixture{skipVerify: true, valkey: allCertificates}), redactSecrets, pmmAgentVersion)
			requireNoTLSArgs(t, actual.Args)
			require.Contains(t, actual.Args, "--redis.addr=redis://username:secret@1.2.3.4:6379")
		})

		t.Run("NoTLSArgumentsByDefault", func(t *testing.T) {
			t.Parallel()

			actual := valkeyExporterConfig(node, service, newExporter(exporterFixture{}), redactSecrets, pmmAgentVersion)
			requireNoTLSArgs(t, actual.Args)
			require.Nil(t, actual.TextFiles)
		})

		t.Run("SocketConnection", func(t *testing.T) {
			t.Parallel()

			socketService := &models.Service{
				ServiceType: models.ValkeyServiceType,
				Socket:      new("/tmp/valkey.sock"),
			}

			actual := valkeyExporterConfig(node, socketService, newExporter(exporterFixture{tls: true, valkey: allCertificates}), redactSecrets, pmmAgentVersion)
			require.Contains(t, actual.Args, "--tls-ca-cert-file={{ .TextFiles.tlsCa }}")
			require.Contains(t, actual.Args, "--tls-client-cert-file={{ .TextFiles.tlsCert }}")
			require.Contains(t, actual.Args, "--tls-client-key-file={{ .TextFiles.tlsKey }}")
			require.Contains(t, actual.Args, "--redis.addr=rediss://username:secret@%2Ftmp%2Fvalkey.sock")
		})

		// No feature gate applies: every flag exists in the exporter shipped since Valkey support landed.
		t.Run("UnknownAgentVersion", func(t *testing.T) {
			t.Parallel()

			actual := valkeyExporterConfig(node, service, newExporter(exporterFixture{tls: true, skipVerify: true, valkey: allCertificates}), redactSecrets, nil)
			require.Contains(t, actual.Args, "--skip-tls-verification")
			require.Contains(t, actual.Args, "--tls-ca-cert-file={{ .TextFiles.tlsCa }}")
			require.Contains(t, actual.Args, "--tls-client-cert-file={{ .TextFiles.tlsCert }}")
			require.Contains(t, actual.Args, "--tls-client-key-file={{ .TextFiles.tlsKey }}")
		})

		t.Run("PrivateKeyIsRedacted", func(t *testing.T) {
			t.Parallel()

			actual := valkeyExporterConfig(node, service, newExporter(exporterFixture{tls: true, valkey: allCertificates}), redactSecrets, pmmAgentVersion)
			require.Contains(t, actual.RedactWords, "key-pem")
			require.NotContains(t, actual.RedactWords, "ca-pem")
			require.NotContains(t, actual.RedactWords, "cert-pem")
		})

		// Exposing secrets drops the redaction list only; the exporter still needs the material.
		t.Run("SecretsExposedOnRequest", func(t *testing.T) {
			t.Parallel()

			actual := valkeyExporterConfig(node, service, newExporter(exporterFixture{tls: true, valkey: allCertificates}), exposeSecrets, pmmAgentVersion)
			require.Nil(t, actual.RedactWords)
			require.Equal(t, "key-pem", actual.TextFiles["tlsKey"])
			require.Contains(t, actual.Args, "--tls-client-key-file={{ .TextFiles.tlsKey }}")
		})

		// pmm-agent renders the certificate contents as templates, so the delimiters have to
		// avoid anything the certificates themselves contain.
		t.Run("DelimitersAvoidCertificateContent", func(t *testing.T) {
			t.Parallel()

			for name, tc := range map[string]struct {
				options  models.ValkeyOptions
				expected string
			}{
				"ca":   {models.ValkeyOptions{SSLCa: "ca {{ pem"}, "--tls-ca-cert-file=[[ .TextFiles.tlsCa ]]"},
				"cert": {models.ValkeyOptions{SSLCert: "cert {{ pem"}, "--tls-client-cert-file=[[ .TextFiles.tlsCert ]]"},
				"key":  {models.ValkeyOptions{SSLKey: "key {{ pem"}, "--tls-client-key-file=[[ .TextFiles.tlsKey ]]"},
			} {
				t.Run(name, func(t *testing.T) {
					t.Parallel()

					actual := valkeyExporterConfig(node, service, newExporter(exporterFixture{tls: true, valkey: tc.options}), redactSecrets, pmmAgentVersion)
					require.Equal(t, "[[", actual.TemplateLeftDelim)
					require.Equal(t, "]]", actual.TemplateRightDelim)
					require.Contains(t, actual.Args, tc.expected)
					require.Contains(t, actual.Args, "--web.listen-address=0.0.0.0:[[ .listen_port ]]")
					require.NotContains(t, strings.Join(actual.Args, " "), "{{")
				})
			}
		})

		t.Run("ArgumentsAreDeterministic", func(t *testing.T) {
			t.Parallel()

			fixture := exporterFixture{tls: true, skipVerify: true, valkey: allCertificates}

			first := valkeyExporterConfig(node, service, newExporter(fixture), redactSecrets, pmmAgentVersion)
			for range 10 {
				require.Equal(t, first.Args, valkeyExporterConfig(node, service, newExporter(fixture), redactSecrets, pmmAgentVersion).Args)
			}
		})
	})
}
