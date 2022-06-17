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
load_defaults: true
# priority is as follows, from the highest priority
#   1. this config value
#   2. PERCONA_TEST_SAAS_HOST env variable
#   3. check.percona.com
saas_hostname: "check.localhost"
endpoints:
  # %s is substituted with 'saas_hostname'
  report: https://%s/v1/telemetry/Report
datasources:
  VM:
    enabled: true
    timeout: 2s
    address: http://localhost:80/victoriametrics/
  QANDB_SELECT:
    enabled: true
    timeout: 2s
    dsn: tcp://127.0.0.1:9000?database=pmm&block_size=10000&pool_size=2
  PMMDB_SELECT:
    enabled: true
    timeout: 2s
    use_separate_credentials: true
    separate_credentials:
      username: pmm-managed
      password: pmm-managed
reporting:
  skip_tls_verification: true
  send_on_start: true
  interval: 10s
  interval_env: "PERCONA_TEST_TELEMETRY_INTERVAL"
  retry_backoff: 1s
  retry_backoff_env: "PERCONA_TEST_TELEMETRY_RETRY_BACKOFF"
  retry_count: 2
  send_timeout: 10s`
	var actual ServiceConfig
	err := yaml.Unmarshal([]byte(input), &actual)
	require.Nil(t, err)
	expected := ServiceConfig{
		Enabled:      true,
		LoadDefaults: true,
		SaasHostname: "check.localhost",
		Endpoints: EndpointsConfig{
			Report: "https://%s/v1/telemetry/Report",
		},
		Reporting: ReportingConfig{
			SkipTLSVerification: true,
			SendOnStart:         true,
			Interval:            time.Second * 10,
			IntervalEnv:         "PERCONA_TEST_TELEMETRY_INTERVAL",
			RetryBackoff:        time.Second * 1,
			RetryBackoffEnv:     "PERCONA_TEST_TELEMETRY_RETRY_BACKOFF",
			RetryCount:          2,
			SendTimeout:         time.Second * 10,
		},
		DataSources: struct {
			VM          *DataSourceVictoriaMetrics `yaml:"VM"`
			QanDBSelect *DSConfigQAN               `yaml:"QANDB_SELECT"` //nolint:tagliatelle
			PmmDBSelect *DSConfigPMMDB             `yaml:"PMMDB_SELECT"` //nolint:tagliatelle
		}{
			VM: &DataSourceVictoriaMetrics{
				Enabled: true,
				Timeout: time.Second * 2,
				Address: "http://localhost:80/victoriametrics/",
			},
			QanDBSelect: &DSConfigQAN{
				Enabled: true,
				Timeout: time.Second * 2,
				DSN:     "tcp://127.0.0.1:9000?database=pmm&block_size=10000&pool_size=2",
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
		},
	}
	assert.Equal(t, actual, expected)
	logger, _ := test.NewNullLogger()
	err = actual.Init(logger.WithField("test", t.Name()))
	require.Nil(t, err)
}
