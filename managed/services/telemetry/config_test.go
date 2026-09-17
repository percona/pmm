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

package telemetry

import (
	"testing"
	"time"

	telemetryv1 "github.com/percona/platform/gen/telemetry/generic"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestServiceConfigUnmarshal(t *testing.T) {
	input := `
enabled: true
saas_hostname: "check.localhost"
datasources:
  VM:
    enabled: true
    timeout: 2s
    address: http://localhost:80/victoriametrics/
  QANDB_SELECT:
    enabled: true
    timeout: 2s
  PMMDB_SELECT:
    enabled: true
    timeout: 2s
    use_separate_credentials: true
    separate_credentials:
      username: pmm-managed
      password: pmm-managed
  GRAFANADB_SELECT:
    enabled: true
    timeout: 2s
    use_separate_credentials: true
    separate_credentials:
      username: grafana
      password: grafana
  ENV_VARS:
    enabled: true
reporting:
  send: true
  send_on_start: true
  interval: 10s
  retry_backoff: 1s
  retry_count: 2
  send_timeout: 10s
`
	var actual ServiceConfig
	err := yaml.Unmarshal([]byte(input), &actual)
	require.NoError(t, err)
	expected := ServiceConfig{
		Enabled:      true,
		SaasHostname: "check.localhost",
		Reporting: ReportingConfig{
			Send:         true,
			SendOnStart:  true,
			Interval:     time.Second * 10,
			RetryBackoff: time.Second * 1,
			RetryCount:   2,
			SendTimeout:  time.Second * 10,
		},
		DataSources: DataSources{
			VM: &DSConfigVM{
				Enabled: true,
				Timeout: time.Second * 2,
				Address: "http://localhost:80/victoriametrics/",
			},
			QanDBSelect: &DSConfigQAN{
				Enabled: true,
				Timeout: time.Second * 2,
			},
			PmmDBSelect: &DSConfigPMMDB{
				Enabled:                true,
				Timeout:                time.Second * 2,
				UseSeparateCredentials: true,
				SeparateCredentials: struct {
					Username string `yaml:"username"`
					Password string `yaml:"password"`
				}{
					Username: "pmm-managed",
					Password: "pmm-managed",
				},
			},
			GrafanaDBSelect: &DSConfigGrafanaDB{
				Enabled:                true,
				Timeout:                time.Second * 2,
				UseSeparateCredentials: true,
				SeparateCredentials: struct {
					Username string `yaml:"username"`
					Password string `yaml:"password"`
				}{
					Username: "grafana",
					Password: "grafana",
				},
			},
			EnvVars: &DSConfigEnvVars{
				Enabled: true,
			},
		},
	}
	assert.Equal(t, expected, actual)
	logger, _ := test.NewNullLogger()
	err = actual.Init(logger.WithField("test", t.Name()))
	require.NoError(t, err)
}

// Only the Helm charts set PMM_INSTALL_METHOD, and kubelet sets KUBERNETES_SERVICE_HOST in every
// Pod, so this pair of presence flags is what separates a Kubernetes deployment from a Docker one.
func TestDefaultConfigReportsKubernetesDeployment(t *testing.T) {
	logger, _ := test.NewNullLogger()
	logEntry := logger.WithField("test", t.Name())
	cfg := ServiceConfig{l: logEntry}

	telemetry, err := cfg.loadMetricsConfig("")
	require.NoError(t, err)

	byID := make(map[string]Config, len(telemetry))
	for _, each := range telemetry {
		byID[each.ID] = each
	}

	helm, ok := byID["PMMServerInstalledWithHelm"]
	require.True(t, ok, "PMMServerInstalledWithHelm datapoint is missing")
	assert.Equal(t, string(dsEnvVars), helm.Source)
	assert.Equal(t, []ConfigData{{MetricName: "pmm_server_installed_with_helm", Column: "PMM_INSTALL_METHOD"}}, helm.Data)
	require.NotNil(t, helm.Transform)
	assert.Equal(t, StripValuesTransform, helm.Transform.Type, "the annotation value must not be reported")

	inKubernetes, ok := byID["PMMServerInKubernetes"]
	require.True(t, ok, "PMMServerInKubernetes datapoint is missing")
	assert.Equal(t, string(dsEnvVars), inKubernetes.Source)
	assert.Equal(t, []ConfigData{{MetricName: "pmm_server_in_kubernetes", Column: "KUBERNETES_SERVICE_HOST"}}, inKubernetes.Data)
	require.NotNil(t, inKubernetes.Transform)
	assert.Equal(t, StripValuesTransform, inKubernetes.Transform.Type, "the cluster address must not be reported")

	dataSource := NewDataSourceEnvVars(DSConfigEnvVars{Enabled: true}, logEntry)

	// report yields what a datapoint contributes to the telemetry report from the current
	// environment, transformed the way prepareReport transforms it.
	report := func(t *testing.T, config Config) []*telemetryv1.GenericReport_Metric {
		t.Helper()

		metrics, err := dataSource.FetchMetrics(t.Context(), config)
		require.NoError(t, err)
		metrics, err = transformExportValues(&config, metrics)
		require.NoError(t, err)

		return metrics
	}

	// t.Setenv cannot unset a variable, but the datasource skips unset and empty values alike.
	t.Run("Helm on Kubernetes without HA", func(t *testing.T) {
		t.Setenv("PMM_INSTALL_METHOD", "Helm")
		t.Setenv("KUBERNETES_SERVICE_HOST", "10.96.0.1")
		t.Setenv("PMM_HA_ENABLE", "")

		assert.Equal(t, []*telemetryv1.GenericReport_Metric{{Key: "pmm_server_installed_with_helm", Value: "1"}}, report(t, helm))
		assert.Equal(t, []*telemetryv1.GenericReport_Metric{{Key: "pmm_server_in_kubernetes", Value: "1"}}, report(t, inKubernetes))

		// The Kubernetes signal must stand on its own, without implying the HA feature.
		haEnabled, ok := byID["PMMServerHAEnabled"]
		require.True(t, ok, "PMMServerHAEnabled datapoint is missing")
		metrics, err := dataSource.FetchMetrics(t.Context(), haEnabled)
		require.NoError(t, err)
		assert.Empty(t, metrics)
	})

	t.Run("Kubernetes without Helm", func(t *testing.T) {
		t.Setenv("PMM_INSTALL_METHOD", "")
		t.Setenv("KUBERNETES_SERVICE_HOST", "10.96.0.1")

		assert.Empty(t, report(t, helm))
		assert.Equal(t, []*telemetryv1.GenericReport_Metric{{Key: "pmm_server_in_kubernetes", Value: "1"}}, report(t, inKubernetes))
	})

	t.Run("Docker", func(t *testing.T) {
		t.Setenv("PMM_INSTALL_METHOD", "")
		t.Setenv("KUBERNETES_SERVICE_HOST", "")

		assert.Empty(t, report(t, helm))
		assert.Empty(t, report(t, inKubernetes))
	})

	t.Run("environment values are never reported", func(t *testing.T) {
		t.Setenv("PMM_INSTALL_METHOD", "Helm cluster-name.example.com")
		t.Setenv("KUBERNETES_SERVICE_HOST", "10.96.0.1")

		for _, metric := range append(report(t, helm), report(t, inKubernetes)...) {
			assert.Equal(t, "1", metric.Value, "%s must report presence only", metric.Key)
		}
	})
}
