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
	"time"

	"github.com/AlekSi/pointer"
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
		Address: pointer.ToString("1.2.3.4"),
		Port:    pointer.ToUint16(6379),
	}
	exporter := &models.Agent{
		AgentID:   "agent-id",
		AgentType: models.ValkeyExporterType,
		Username:  pointer.ToString("username"),
		Password:  pointer.ToString("secret"),
	}

	t.Run("DefaultTimeoutUsesFlag", func(t *testing.T) {
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
		exporter.ExporterOptions.Timeout = 1500 * time.Millisecond

		actual := valkeyExporterConfig(node, service, exporter, redactSecrets, pmmAgentVersion)
		require.Contains(t, actual.Args, "--connection-timeout=1.5s")
		require.Contains(t, actual.Args, "--redis.addr=redis://username:secret@1.2.3.4:6379")
	})
}
