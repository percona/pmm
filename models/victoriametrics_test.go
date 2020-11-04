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

package models

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVictoriaMetricsParams(t *testing.T) {
	t.Run("read non exist baseConfigFile", func(t *testing.T) {
		_, err := NewVictoriaMetricsParams("nonExistConfigFile.yml")
		require.NoError(t, err)
	})
	t.Run("check params for VMAlert", func(t *testing.T) {
		vmp, err := NewVictoriaMetricsParams("../testdata/victoriametrics/prometheus.external.alerts.yml")
		require.NoError(t, err)
		require.Equal(t, []string{"--rule=/srv/external_rules/rul1.yml", "--rule=/srv/external_rules/rule2.yml", "--evaluationInterval=10s"}, vmp.VMAlertFlags)
	})
	t.Run("check error for remote_write", func(t *testing.T) {
		_, err := NewVictoriaMetricsParams("../testdata/victoriametrics/prometheus.external.remotewrite.yml")
		require.Error(t, err)
	})
}
