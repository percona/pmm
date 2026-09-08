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

// PMM_INSTALL_METHOD is injected by the Helm charts and KUBERNETES_SERVICE_HOST by kubelet, so this
// pair of datapoints is what separates a Kubernetes deployment from a Docker one in telemetry.
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

	installMethod, ok := byID["PMMServerInstallMethod"]
	require.True(t, ok, "PMMServerInstallMethod datapoint is missing")
	assert.Equal(t, string(dsEnvVars), installMethod.Source)
	assert.Equal(t, []ConfigData{{MetricName: "pmm_server_install_method", Column: "PMM_INSTALL_METHOD"}}, installMethod.Data)
	assert.Nil(t, installMethod.Transform, "the install method is reported as-is")

	inKubernetes, ok := byID["PMMServerInKubernetes"]
	require.True(t, ok, "PMMServerInKubernetes datapoint is missing")
	assert.Equal(t, string(dsEnvVars), inKubernetes.Source)
	assert.Equal(t, []ConfigData{{MetricName: "pmm_server_in_kubernetes", Column: "KUBERNETES_SERVICE_HOST"}}, inKubernetes.Data)
	require.NotNil(t, inKubernetes.Transform)
	assert.Equal(t, StripValuesTransform, inKubernetes.Transform.Type, "the cluster address must not be reported")

	dataSource := NewDataSourceEnvVars(DSConfigEnvVars{Enabled: true}, logEntry)

	t.Run("Kubernetes", func(t *testing.T) {
		t.Setenv("PMM_INSTALL_METHOD", "Helm")
		t.Setenv("KUBERNETES_SERVICE_HOST", "10.96.0.1")

		metrics, err := dataSource.FetchMetrics(t.Context(), installMethod)
		require.NoError(t, err)
		assert.Equal(t, []*telemetryv1.GenericReport_Metric{{Key: "pmm_server_install_method", Value: "Helm"}}, metrics)

		metrics, err = dataSource.FetchMetrics(t.Context(), inKubernetes)
		require.NoError(t, err)
		metrics, err = transformExportValues(&inKubernetes, metrics)
		require.NoError(t, err)
		assert.Equal(t, []*telemetryv1.GenericReport_Metric{{Key: "pmm_server_in_kubernetes", Value: "1"}}, metrics)
	})

	t.Run("Docker", func(t *testing.T) {
		t.Setenv("PMM_INSTALL_METHOD", "")
		t.Setenv("KUBERNETES_SERVICE_HOST", "")

		metrics, err := dataSource.FetchMetrics(t.Context(), installMethod)
		require.NoError(t, err)
		assert.Empty(t, metrics)

		metrics, err = dataSource.FetchMetrics(t.Context(), inKubernetes)
		require.NoError(t, err)
		assert.Empty(t, metrics)
	})
}
