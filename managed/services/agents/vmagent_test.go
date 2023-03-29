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
}
