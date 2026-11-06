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

	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/version"
)

func TestRDSExporterConfig(t *testing.T) {
	pmmAgentVersion := version.MustParse("2.28.0")

	node1 := &models.Node{
		NodeID:     "node1",
		NodeType:   models.RemoteRDSNodeType,
		NodeName:   "prod-mysql56",
		NodeModel:  "db.t2.micro",
		Region:     new("us-east-1"),
		AZ:         "us-east-1c",
		Address:    "rds-mysql56.xyzzy.us-east-1.rds.amazonaws.com",
		InstanceID: "rds-mysql56",
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
		NodeID:     "node2",
		NodeType:   models.RemoteRDSNodeType,
		NodeName:   "test-mysql57",
		NodeModel:  "db.t2.micro",
		Region:     new("us-east-1"),
		AZ:         "us-east-1c",
		Address:    "rds-mysql57.xyzzy.us-east-1.rds.amazonaws.com",
		InstanceID: "rds-mysql57",
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
	nodeServices := map[string][]*models.Service{
		node1.NodeID: {{
			ServiceID:      "service1",
			ServiceType:    models.MySQLServiceType,
			ServiceName:    "prod-mysql56",
			NodeID:         node1.NodeID,
			Environment:    "prod",
			Cluster:        "mysql56",
			ReplicationSet: "rs56",
		}},
	}
	actual, err := rdsExporterConfig(pairs, nodeServices, redactSecrets, pmmAgentVersion)
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
        cluster: mysql56
        environment: prod
        foo: bar
        node_id: node1
        node_model: db.t2.micro
        node_name: prod-mysql56
        node_type: remote_rds
        replication_set: rs56
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

func TestRDSExporterConfigRoleARN(t *testing.T) {
	pmmAgentVersion := version.MustParse("2.28.0")

	node := &models.Node{
		NodeID:     "node1",
		NodeType:   models.RemoteRDSNodeType,
		NodeName:   "prod-mysql56",
		NodeModel:  "db.t2.micro",
		Region:     new("us-east-1"),
		AZ:         "us-east-1c",
		Address:    "rds-mysql56.xyzzy.us-east-1.rds.amazonaws.com",
		InstanceID: "rds-mysql56",
	}
	agent := &models.Agent{
		AgentID:   "agent1",
		AgentType: models.RDSExporterType,
		NodeID:    &node.NodeID,
		AWSOptions: models.AWSOptions{
			AWSRoleARN: "arn:aws:iam::123456789012:role/pmm-monitoring",
		},
	}

	actual, err := rdsExporterConfig(map[*models.Node]*models.Agent{node: agent}, nil, redactSecrets, pmmAgentVersion)
	require.NoError(t, err)

	expected := strings.TrimSpace(`
---
instances:
    - region: us-east-1
      instance: rds-mysql56
      aws_role_arn: arn:aws:iam::123456789012:role/pmm-monitoring
      disable_basic_metrics: false
      disable_enhanced_metrics: false
      labels:
        agent_id: agent1
        agent_type: rds_exporter
        az: us-east-1c
        node_id: node1
        node_model: db.t2.micro
        node_name: prod-mysql56
        node_type: remote_rds
	`) + "\n"

	require.Equal(t, expected, actual.TextFiles["config"])
	require.Equal(t, []string{}, actual.RedactWords)
}

func TestRDSExporterLabels(t *testing.T) {
	t.Parallel()

	node := &models.Node{
		NodeID:     "node1",
		NodeType:   models.RemoteRDSNodeType,
		NodeName:   "prod-mysql",
		Region:     new("us-east-1"),
		InstanceID: "prod-mysql",
	}
	agent := &models.Agent{AgentID: "agent1", AgentType: models.RDSExporterType}
	service := func(id, environment, cluster, replicationSet string) *models.Service {
		return &models.Service{
			ServiceID:      id,
			ServiceType:    models.MySQLServiceType,
			ServiceName:    id,
			NodeID:         node.NodeID,
			Environment:    environment,
			Cluster:        cluster,
			ReplicationSet: replicationSet,
		}
	}
	nodeLabels := model.LabelSet{
		"agent_id":   "agent1",
		"agent_type": "rds_exporter",
		"node_id":    "node1",
		"node_name":  "prod-mysql",
		"node_type":  "remote_rds",
	}
	withLabels := func(extra model.LabelSet) model.LabelSet {
		return nodeLabels.Merge(extra)
	}

	for _, tc := range []struct {
		name     string
		services []*models.Service
		expected model.LabelSet
	}{{
		name:     "no Service",
		expected: nodeLabels,
	}, {
		name:     "one Service",
		services: []*models.Service{service("s1", "prod", "c1", "rs1")},
		expected: withLabels(model.LabelSet{"environment": "prod", "cluster": "c1", "replication_set": "rs1"}),
	}, {
		name:     "empty values make no labels",
		services: []*models.Service{service("s1", "prod", "", " ")},
		expected: withLabels(model.LabelSet{"environment": "prod"}),
	}, {
		name: "Services that agree",
		services: []*models.Service{
			service("s1", "prod", "c1", "rs1"),
			service("s2", "prod", "c1", "rs1"),
		},
		expected: withLabels(model.LabelSet{"environment": "prod", "cluster": "c1", "replication_set": "rs1"}),
	}, {
		name: "Services that disagree keep only the labels they share",
		services: []*models.Service{
			service("s1", "prod", "c1", "rs1"),
			service("s2", "prod", "c2", ""),
		},
		expected: withLabels(model.LabelSet{"environment": "prod"}),
	}, {
		name: "a Service without a value drops the label",
		services: []*models.Service{
			service("s1", "", "c1", "rs1"),
			service("s2", "prod", "c1", "rs1"),
		},
		expected: withLabels(model.LabelSet{"cluster": "c1", "replication_set": "rs1"}),
	}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actual, err := mergeLabels(node, tc.services, agent)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, actual)
		})
	}

	t.Run("labels merge in the order of the Service series", func(t *testing.T) {
		t.Parallel()

		node := &models.Node{NodeID: "node1", NodeType: models.RemoteRDSNodeType, NodeName: "prod-mysql"}
		require.NoError(t, node.SetCustomLabels(map[string]string{"environment": "node", "cluster": "node"}))
		s := service("s1", "prod", "c1", "rs1")
		require.NoError(t, s.SetCustomLabels(map[string]string{"replication_set": "custom"}))
		agent := &models.Agent{AgentID: "agent1", AgentType: models.RDSExporterType}
		require.NoError(t, agent.SetCustomLabels(map[string]string{"cluster": "agent"}))

		actual, err := mergeLabels(node, []*models.Service{s}, agent)
		require.NoError(t, err)
		assert.Equal(t, withLabels(model.LabelSet{
			"environment":     "prod",
			"cluster":         "agent",
			"replication_set": "custom",
		}), actual)
	})
}

func TestCheckRDSExporterSupported(t *testing.T) {
	t.Parallel()

	pmmAgent := func(v string) *models.Agent {
		agent := &models.Agent{AgentID: "pmm-agent", AgentType: models.PMMAgentType}
		if v != "" {
			agent.Version = &v
		}
		return agent
	}
	keysExporter := &models.Agent{
		AgentID:    "keys",
		AgentType:  models.RDSExporterType,
		AWSOptions: models.AWSOptions{AWSAccessKey: "AKIAIOSFODNN7EXAMPLE", AWSSecretKey: "secret"},
	}
	roleExporter := &models.Agent{
		AgentID:    "role",
		AgentType:  models.RDSExporterType,
		AWSOptions: models.AWSOptions{AWSRoleARN: "arn:aws:iam::123456789012:role/pmm-monitoring"},
	}

	t.Run("static keys on any version", func(t *testing.T) {
		t.Parallel()

		assert.NoError(t, checkRDSExporterSupported(pmmAgent("3.3.1"), keysExporter))
		assert.NoError(t, checkRDSExporterSupported(pmmAgent(""), keysExporter))
	})

	t.Run("role on 3.4.0", func(t *testing.T) {
		t.Parallel()

		assert.NoError(t, checkRDSExporterSupported(pmmAgent("3.4.0"), roleExporter))
		assert.NoError(t, checkRDSExporterSupported(pmmAgent("3.4.0-beta1"), roleExporter))
	})

	t.Run("role on a downgraded pmm-agent", func(t *testing.T) {
		t.Parallel()

		err := checkRDSExporterSupported(pmmAgent("3.3.1"), roleExporter)
		var notSupported models.AgentNotSupportedError
		require.ErrorAs(t, err, &notSupported)
		assert.Equal(t, "3.3.1", notSupported.AgentVersion)
	})

	t.Run("role on a pmm-agent without a version", func(t *testing.T) {
		t.Parallel()

		require.Error(t, checkRDSExporterSupported(pmmAgent(""), roleExporter))
	})
}
