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

package prometheus

import (
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/pmezard/go-difflib/difflib"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/services/prometheus/internal/prometheus/config"
	sd_config "github.com/percona/pmm-managed/services/prometheus/internal/prometheus/discovery/config"
	"github.com/percona/pmm-managed/services/prometheus/internal/prometheus/discovery/targetgroup"
)

func TestScrapeConfig(t *testing.T) {
	t.Run("scrapeConfigsForMySQLdExporter", func(t *testing.T) {
		t.Run("Normal", func(t *testing.T) {
			// Setup models
			node := &models.Node{
				NodeID:  "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
				Address: "1.2.3.4",
			}
			service := &models.Service{
				ServiceID: "/service_id/014647c3-b2f5-44eb-94f4-d943260a968c",
				NodeID:    "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
				Address:   pointer.ToString("5.6.7.8"),
			}
			agent := &models.Agent{
				AgentID:      "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
				AgentType:    models.MySQLdExporterType,
				RunsOnNodeID: "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
				CustomLabels: []byte(`{"_some_agent_label": "baz"}`),
				ListenPort:   pointer.ToUint16(12345),
			}

			expected := []*config.ScrapeConfig{{
				JobName:        "_agent_id_75bb30d3-ef4a-4147-97a8-621a996611dd_hr",
				ScrapeInterval: model.Duration(time.Second),
				ScrapeTimeout:  model.Duration(time.Second),
				MetricsPath:    "/metrics-hr",
				ServiceDiscoveryConfig: sd_config.ServiceDiscoveryConfig{
					StaticConfigs: []*targetgroup.Group{{
						Targets: []model.LabelSet{{"__address__": "1.2.3.4:12345"}},
						Labels: model.LabelSet{
							"_some_agent_label": "baz",
							"instance":          "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
							"node_id":           "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
							"service_id":        "/service_id/014647c3-b2f5-44eb-94f4-d943260a968c",
						},
					}},
				},
			}, {
				JobName:        "_agent_id_75bb30d3-ef4a-4147-97a8-621a996611dd_mr",
				ScrapeInterval: model.Duration(10 * time.Second),
				ScrapeTimeout:  model.Duration(5 * time.Second),
				MetricsPath:    "/metrics-mr",
				ServiceDiscoveryConfig: sd_config.ServiceDiscoveryConfig{
					StaticConfigs: []*targetgroup.Group{{
						Targets: []model.LabelSet{{"__address__": "1.2.3.4:12345"}},
						Labels: model.LabelSet{
							"_some_agent_label": "baz",
							"instance":          "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
							"node_id":           "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
							"service_id":        "/service_id/014647c3-b2f5-44eb-94f4-d943260a968c",
						},
					}},
				},
			}, {
				JobName:        "_agent_id_75bb30d3-ef4a-4147-97a8-621a996611dd_lr",
				ScrapeInterval: model.Duration(60 * time.Second),
				ScrapeTimeout:  model.Duration(10 * time.Second),
				MetricsPath:    "/metrics-lr",
				ServiceDiscoveryConfig: sd_config.ServiceDiscoveryConfig{
					StaticConfigs: []*targetgroup.Group{{
						Targets: []model.LabelSet{{"__address__": "1.2.3.4:12345"}},
						Labels: model.LabelSet{
							"_some_agent_label": "baz",
							"instance":          "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
							"node_id":           "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
							"service_id":        "/service_id/014647c3-b2f5-44eb-94f4-d943260a968c",
						},
					}},
				},
			}}

			// Exercise scrapeConfigsForMySQLdExporter
			actual, err := scrapeConfigsForMySQLdExporter(node, service, agent)

			// Verify Results
			require.NoError(t, err)
			require.Len(t, actual, len(expected))
			for i := 0; i < len(expected); i++ {
				assertScrappedConfigsEqual(t, expected[i], actual[i])
			}
		})

		t.Run("BadCustomLabels", func(t *testing.T) {
			// Setup models
			node := &models.Node{}
			service := &models.Service{}
			agent := &models.Agent{
				CustomLabels: []byte("{"),
				ListenPort:   pointer.ToUint16(12345),
			}

			// Exercise scrapeConfigsForMySQLdExporter
			_, err := scrapeConfigsForMySQLdExporter(node, service, agent)

			// Verify Results
			require.EqualError(t, err, "failed to decode custom labels: unexpected end of JSON input")
		})
	})

	t.Run("scrapeConfigsForMongoDBExporter", func(t *testing.T) {
		t.Run("Normal", func(t *testing.T) {
			// Setup models
			node := &models.Node{
				NodeID:  "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
				Address: "1.2.3.4",
			}
			service := &models.Service{
				ServiceID: "/service_id/014647c3-b2f5-44eb-94f4-d943260a968c",
				NodeID:    "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
				Address:   pointer.ToString("5.6.7.8"),
			}
			agent := &models.Agent{
				AgentID:      "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
				AgentType:    models.MongoDBExporterType,
				RunsOnNodeID: "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
				CustomLabels: []byte(`{"_some_agent_label": "baz"}`),
				ListenPort:   pointer.ToUint16(12345),
			}

			expected := &config.ScrapeConfig{
				JobName:        "_agent_id_75bb30d3-ef4a-4147-97a8-621a996611dd",
				ScrapeInterval: model.Duration(time.Second),
				ScrapeTimeout:  model.Duration(time.Second),
				MetricsPath:    "/metrics",
				ServiceDiscoveryConfig: sd_config.ServiceDiscoveryConfig{
					StaticConfigs: []*targetgroup.Group{{
						Targets: []model.LabelSet{{"__address__": "1.2.3.4:12345"}},
						Labels: model.LabelSet{
							"_some_agent_label": "baz",
							"instance":          "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
							"node_id":           "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
							"service_id":        "/service_id/014647c3-b2f5-44eb-94f4-d943260a968c",
						},
					}},
				},
			}

			// Exercise scrapeConfigsForMongoDBExporter
			actual, err := scrapeConfigsForMongoDBExporter(node, service, agent)

			// Verify Results
			require.NoError(t, err)
			assertScrappedConfigsEqual(t, expected, actual)
		})

		t.Run("BadCustomLabels", func(t *testing.T) {
			// Setup models
			node := &models.Node{}
			service := &models.Service{}
			agent := &models.Agent{
				CustomLabels: []byte("{"),
				ListenPort:   pointer.ToUint16(12345),
			}

			// Exercise scrapeConfigsForMongoDBExporter
			_, err := scrapeConfigsForMongoDBExporter(node, service, agent)

			// Verify Results
			require.EqualError(t, err, "failed to decode custom labels: unexpected end of JSON input")
		})
	})

	t.Run("commonExporterLabelSet", func(t *testing.T) {
		// Setup models
		node := &models.Node{
			NodeID:              "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
			NodeName:            "test-node",
			Address:             "1.2.3.4",
			MachineID:           pointer.ToString("test-machine-id"),
			DockerContainerID:   pointer.ToString("cc663f36-0000-1111-2222-c6310bb4738d"),
			DockerContainerName: "test-container-name",
		}
		service := &models.Service{
			ServiceID:   "/service_id/014647c3-b2f5-44eb-94f4-d943260a968c",
			ServiceName: "test-service-name",
			NodeID:      "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
			Address:     pointer.ToString("5.6.7.8"),
		}
		agent := &models.Agent{
			AgentID:      "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
			AgentType:    models.MongoDBExporterType,
			RunsOnNodeID: "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
			CustomLabels: []byte(`{"_some_agent_label": "baz"}`),
			ListenPort:   pointer.ToUint16(12345),
		}
		expected := model.LabelSet{
			model.LabelName("node_id"):               model.LabelValue("/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d"),
			model.LabelName("node_name"):             model.LabelValue("test-node"),
			model.LabelName("machine_id"):            model.LabelValue("test-machine-id"),
			model.LabelName("docker_container_id"):   model.LabelValue("cc663f36-0000-1111-2222-c6310bb4738d"),
			model.LabelName("docker_container_name"): model.LabelValue("test-container-name"),

			model.LabelName("service_id"):   model.LabelValue("/service_id/014647c3-b2f5-44eb-94f4-d943260a968c"),
			model.LabelName("service_name"): model.LabelValue("test-service-name"),

			model.LabelName("instance"): model.LabelValue("/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd"),
		}

		// Exercise commonExporterLabelSet
		actual := commonExporterLabelSet(node, service, agent)

		// Verify Results
		assert.Equal(t, expected, actual, "Common labels is not Equal")
	})
}

func assertScrappedConfigsEqual(t *testing.T, expected, actual *config.ScrapeConfig) bool {
	if !assert.Equal(t, expected, actual) {
		e, err := yaml.Marshal(expected)
		require.NoError(t, err)
		a, err := yaml.Marshal(actual)
		require.NoError(t, err)

		diff, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
			A:        difflib.SplitLines(string(e)),
			FromFile: "Expected",
			B:        difflib.SplitLines(string(a)),
			ToFile:   "Actual",
			Context:  3,
		})
		require.NoError(t, err)
		t.Logf("Diff:\n%s", diff)
		return false
	}
	return true
}
