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

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/require"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/version"
)

func TestRDSExporterConfig(t *testing.T) {
	pmmAgentVersion := version.MustParse("2.28.0")

	node1 := &models.Node{
		NodeID:    "node1",
		NodeType:  models.RemoteRDSNodeType,
		NodeName:  "prod-mysql56",
		NodeModel: "db.t2.micro",
		Region:    pointer.ToString("us-east-1"),
		AZ:        "us-east-1c",
		Address:   "rds-mysql56",
	}
	err := node1.SetCustomLabels(map[string]string{
		"foo": "bar",
	})
	require.NoError(t, err)
	agent1 := &models.Agent{
		AgentID:   "agent1",
		AgentType: models.RDSExporterType,
		NodeID:    &node1.NodeID,
		AWSOptions: models.AWSOptions{
			AWSAccessKey:            "access_key1",
			AWSSecretKey:            "secret_key1",
			RDSBasicMetricsDisabled: true,
		},
	}

	node2 := &models.Node{
		NodeID:    "node2",
		NodeType:  models.RemoteRDSNodeType,
		NodeName:  "test-mysql57",
		NodeModel: "db.t2.micro",
		Region:    pointer.ToString("us-east-1"),
		AZ:        "us-east-1c",
		Address:   "rds-mysql57",
	}
	err = node2.SetCustomLabels(map[string]string{
		"baz": "qux",
	})
	require.NoError(t, err)
	agent2 := &models.Agent{
		AgentID:   "agent2",
		AgentType: models.RDSExporterType,
		NodeID:    &node2.NodeID,
		AWSOptions: models.AWSOptions{
			AWSAccessKey: "access_key2",
			AWSSecretKey: "secret_key2",
		},
	}

	pairs := map[*models.Node]*models.Agent{
		node2: agent2,
		node1: agent1,
	}
	actual, err := rdsExporterConfig(pairs, redactSecrets, pmmAgentVersion)
	require.NoError(t, err)
	expected := &agentv1.SetStateRequest_AgentProcess{
		Type:               inventoryv1.AgentType_AGENT_TYPE_RDS_EXPORTER,
		TemplateLeftDelim:  "{{",
		TemplateRightDelim: "}}",
		Args: []string{
			"--config.file={{ .TextFiles.config }}",
			"--web.listen-address=:{{ .listen_port }}",
		},
		TextFiles: map[string]string{
			`config`: strings.TrimSpace(`
---
instances:
    - region: us-east-1
      instance: rds-mysql56
      aws_access_key: access_key1
      aws_secret_key: secret_key1
      disable_basic_metrics: true
      disable_enhanced_metrics: false
      labels:
        agent_id: agent1
        agent_type: rds_exporter
        az: us-east-1c
        foo: bar
        node_id: node1
        node_model: db.t2.micro
        node_name: prod-mysql56
        node_type: remote_rds
    - region: us-east-1
      instance: rds-mysql57
      aws_access_key: access_key2
      aws_secret_key: secret_key2
      disable_basic_metrics: false
      disable_enhanced_metrics: false
      labels:
        agent_id: agent2
        agent_type: rds_exporter
        az: us-east-1c
        baz: qux
        node_id: node2
        node_model: db.t2.micro
        node_name: test-mysql57
        node_type: remote_rds
			`) + "\n",
		},
		RedactWords: []string{"secret_key1", "secret_key2"},
	}
	require.Equal(t, expected.Args, actual.Args)
	require.Equal(t, expected.Env, actual.Env)
	require.Equal(t, expected.TextFiles["config"], actual.TextFiles["config"])
	require.Equal(t, expected, actual)
}
