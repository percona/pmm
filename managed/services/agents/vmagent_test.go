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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

func TestMaxScrapeSize(t *testing.T) {
	t.Run("by default 64MiB", func(t *testing.T) {
		params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, models.VMBaseURL)
		require.NoError(t, err)
		actual := vmAgentConfig("", params)
		assert.Contains(t, actual.Args, "-promscrape.maxScrapeSize="+maxScrapeSizeDefault)
	})
	t.Run("overridden with ENV", func(t *testing.T) {
		params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, models.VMBaseURL)
		require.NoError(t, err)
		newValue := "16MiB"
		t.Setenv(maxScrapeSizeEnv, newValue)
		actual := vmAgentConfig("", params)
		assert.Contains(t, actual.Args, "-promscrape.maxScrapeSize="+newValue)
	})
	t.Run("VMAGENT_ ENV variables", func(t *testing.T) {
		params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, models.VMBaseURL)
		require.NoError(t, err)
		t.Setenv("VMAGENT_promscrape_maxScrapeSize", "16MiB")
		t.Setenv("VM_remoteWrite_basicAuth_password", "password")
		actual := vmAgentConfig("", params)
		assert.Contains(t, actual.Env, "VMAGENT_promscrape_maxScrapeSize=16MiB")
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username={{.server_username}}")
		assert.NotContains(t, actual.Env, "VM_remoteWrite_basicAuth_password=password")
	})
	t.Run("External Victoria Metrics ENV variables", func(t *testing.T) {
		params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, "http://victoriametrics:8428")
		require.NoError(t, err)
		t.Setenv("VMAGENT_promscrape_maxScrapeSize", "16MiB")
		actual := vmAgentConfig("", params)
		assert.Contains(t, actual.Args, "-remoteWrite.url=http://victoriametrics:8428/api/v1/write")
		assert.Contains(t, actual.Env, "VMAGENT_promscrape_maxScrapeSize=16MiB")
		assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username={{.server_username}}")
	})
}
