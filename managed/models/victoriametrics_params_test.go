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

package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVictoriaMetricsParams(t *testing.T) {
	t.Run("read non exist baseConfigFile", func(t *testing.T) {
		_, err := NewVictoriaMetricsParams("nonExistConfigFile.yml", VMBaseURL)
		require.NoError(t, err)
	})
	t.Run("check params for VMAlert", func(t *testing.T) {
		vmp, err := NewVictoriaMetricsParams("../testdata/victoriametrics/prometheus.external.alerts.yml", VMBaseURL)
		require.NoError(t, err)
		require.Equal(t, []string{"--rule=/srv/external_rules/rul1.yml", "--rule=/srv/external_rules/rule2.yml", "--evaluationInterval=10s"}, vmp.VMAlertFlags)
	})
	t.Run("check external VM", func(t *testing.T) {
		tests := []struct {
			url  string
			want bool
		}{
			{
				"http://127.0.0.1:9090/prometheus",
				false,
			},
			{
				"http://127.0.0.1:9090/prometheus/",
				false,
			},
			{
				"http://victoriametrics:8428/",
				true,
			},
			{
				"https://example.com:9090/",
				true,
			},
		}
		for _, tt := range tests {
			t.Run(tt.url, func(t *testing.T) {
				vmp, err := NewVictoriaMetricsParams(BasePrometheusConfigPath, tt.url)
				require.NoError(t, err)
				assert.Equalf(t, tt.want, vmp.ExternalVM(), "ExternalVM()")
			})
		}
	})
}

func TestParseVictoriaMetricsURL(t *testing.T) {
	t.Run("valid URLs get a trailing slash", func(t *testing.T) {
		for raw, want := range map[string]string{
			VMBaseURL:                           VMBaseURL,
			"http://victoriametrics:8428":       "http://victoriametrics:8428/",
			"https://user:pass@vm:8428/path":    "https://user:pass@vm:8428/path/",
			"http://pmm-ha-vmauth.pmm.svc:8427": "http://pmm-ha-vmauth.pmm.svc:8427/",
		} {
			u, err := ParseVictoriaMetricsURL(raw)
			require.NoError(t, err, raw)
			assert.Equal(t, want, u.String(), raw)
		}
	})

	t.Run("URLs without an http scheme and a host are rejected", func(t *testing.T) {
		for _, raw := range []string{"", "vm:8428", "//vm:8428/", "vm.example.com", "ftp://vm:8428", "http:///path"} {
			_, err := ParseVictoriaMetricsURL(raw)
			require.Error(t, err, raw)
			assert.Contains(t, err.Error(), "invalid VictoriaMetrics URL", raw)
		}
	})

	t.Run("errors never echo credentials", func(t *testing.T) {
		_, err := ParseVictoriaMetricsURL("http://user:secret@[::1")
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "secret")

		_, err = ParseVictoriaMetricsURL("ftp://user:secret@vm:8428")
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "secret")
	})
}
