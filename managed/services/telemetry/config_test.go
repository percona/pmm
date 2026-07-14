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
	require.Nil(t, err)
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
	assert.Equal(t, actual, expected)
	logger, _ := test.NewNullLogger()
	err = actual.Init(logger.WithField("test", t.Name()))
	require.Nil(t, err)
}
