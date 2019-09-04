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
	"net/url"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/pmezard/go-difflib/difflib"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/percona/pmm-managed/models"
	config_util "github.com/percona/pmm-managed/services/prometheus/internal/common/config"
	"github.com/percona/pmm-managed/services/prometheus/internal/prometheus/config"
	sd_config "github.com/percona/pmm-managed/services/prometheus/internal/prometheus/discovery/config"
	"github.com/percona/pmm-managed/services/prometheus/internal/prometheus/discovery/targetgroup"
)

func TestScrapeConfig(t *testing.T) {
	t.Run("scrapeConfigForNodeExporter", func(t *testing.T) {
		t.Run("Normal", func(t *testing.T) {
			node := &models.Node{
				NodeID:       "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
				NodeName:     "node_name",
				Address:      "1.2.3.4",
				CustomLabels: []byte(`{"_some_node_label": "foo"}`),
			}
			agent := &models.Agent{
				AgentID:      "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
				AgentType:    models.NodeExporterType,
				RunsOnNodeID: nil,
				CustomLabels: []byte(`{"_some_agent_label": "baz"}`),
				ListenPort:   pointer.ToUint16(12345),
			}

			expected := &config.ScrapeConfig{
				JobName:        "node_exporter_agent_id_75bb30d3-ef4a-4147-97a8-621a996611dd",
				ScrapeInterval: model.Duration(time.Second),
				ScrapeTimeout:  model.Duration(time.Second),
				MetricsPath:    "/metrics",
				HTTPClientConfig: config_util.HTTPClientConfig{
					BasicAuth: &config_util.BasicAuth{
						Username: "pmm",
						Password: "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
					},
				},
				ServiceDiscoveryConfig: sd_config.ServiceDiscoveryConfig{
					StaticConfigs: []*targetgroup.Group{{
						Targets: []model.LabelSet{{"__address__": "1.2.3.4:12345"}},
						Labels: model.LabelSet{
							"_some_agent_label": "baz",
							"_some_node_label":  "foo",
							"agent_id":          "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
							"agent_type":        "node_exporter",
							"instance":          "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
							"node_id":           "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
							"node_name":         "node_name",
						},
					}},
				},
			}

			actual, err := scrapeConfigForNodeExporter(time.Second, node, agent)
			require.NoError(t, err)
			assertScrappedConfigsEqual(t, expected, actual)
		})
	})

	t.Run("scrapeConfigsForMySQLdExporter", func(t *testing.T) {
		s := &models.MetricsResolutions{
			HR: time.Second,
			MR: 5 * time.Second,
			LR: 60 * time.Second,
		}

		t.Run("Normal", func(t *testing.T) {
			node := &models.Node{
				NodeID:       "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
				NodeName:     "node_name",
				Address:      "1.2.3.4",
				CustomLabels: []byte(`{"_some_node_label": "foo"}`),
			}
			service := &models.Service{
				ServiceID:    "/service_id/014647c3-b2f5-44eb-94f4-d943260a968c",
				NodeID:       "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
				Address:      pointer.ToString("5.6.7.8"),
				CustomLabels: []byte(`{"_some_service_label": "bar"}`),
			}
			agent := &models.Agent{
				AgentID:      "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
				AgentType:    models.MySQLdExporterType,
				RunsOnNodeID: nil,
				CustomLabels: []byte(`{"_some_agent_label": "baz"}`),
				ListenPort:   pointer.ToUint16(12345),
			}

			expected := []*config.ScrapeConfig{{
				JobName:        "mysqld_exporter_agent_id_75bb30d3-ef4a-4147-97a8-621a996611dd_hr",
				ScrapeInterval: model.Duration(time.Second),
				ScrapeTimeout:  model.Duration(time.Second),
				MetricsPath:    "/metrics-hr",
				HTTPClientConfig: config_util.HTTPClientConfig{
					BasicAuth: &config_util.BasicAuth{
						Username: "pmm",
						Password: "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
					},
				},
				ServiceDiscoveryConfig: sd_config.ServiceDiscoveryConfig{
					StaticConfigs: []*targetgroup.Group{{
						Targets: []model.LabelSet{{"__address__": "4.5.6.7:12345"}},
						Labels: model.LabelSet{
							"_some_agent_label":   "baz",
							"_some_node_label":    "foo",
							"_some_service_label": "bar",
							"agent_id":            "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
							"agent_type":          "mysqld_exporter",
							"instance":            "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
							"node_id":             "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
							"node_name":           "node_name",
							"service_id":          "/service_id/014647c3-b2f5-44eb-94f4-d943260a968c",
						},
					}},
				},
			}, {
				JobName:        "mysqld_exporter_agent_id_75bb30d3-ef4a-4147-97a8-621a996611dd_mr",
				ScrapeInterval: model.Duration(5 * time.Second),
				ScrapeTimeout:  model.Duration(4 * time.Second),
				MetricsPath:    "/metrics-mr",
				HTTPClientConfig: config_util.HTTPClientConfig{
					BasicAuth: &config_util.BasicAuth{
						Username: "pmm",
						Password: "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
					},
				},
				ServiceDiscoveryConfig: sd_config.ServiceDiscoveryConfig{
					StaticConfigs: []*targetgroup.Group{{
						Targets: []model.LabelSet{{"__address__": "4.5.6.7:12345"}},
						Labels: model.LabelSet{
							"_some_agent_label":   "baz",
							"_some_node_label":    "foo",
							"_some_service_label": "bar",
							"agent_id":            "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
							"agent_type":          "mysqld_exporter",
							"instance":            "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
							"node_id":             "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
							"node_name":           "node_name",
							"service_id":          "/service_id/014647c3-b2f5-44eb-94f4-d943260a968c",
						},
					}},
				},
			}, {
				JobName:        "mysqld_exporter_agent_id_75bb30d3-ef4a-4147-97a8-621a996611dd_lr",
				ScrapeInterval: model.Duration(60 * time.Second),
				ScrapeTimeout:  model.Duration(10 * time.Second),
				MetricsPath:    "/metrics-lr",
				HTTPClientConfig: config_util.HTTPClientConfig{
					BasicAuth: &config_util.BasicAuth{
						Username: "pmm",
						Password: "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
					},
				},
				ServiceDiscoveryConfig: sd_config.ServiceDiscoveryConfig{
					StaticConfigs: []*targetgroup.Group{{
						Targets: []model.LabelSet{{"__address__": "4.5.6.7:12345"}},
						Labels: model.LabelSet{
							"_some_agent_label":   "baz",
							"_some_node_label":    "foo",
							"_some_service_label": "bar",
							"agent_id":            "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
							"agent_type":          "mysqld_exporter",
							"instance":            "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
							"node_id":             "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
							"node_name":           "node_name",
							"service_id":          "/service_id/014647c3-b2f5-44eb-94f4-d943260a968c",
						},
					}},
				},
			}}

			actual, err := scrapeConfigsForMySQLdExporter(s, "4.5.6.7", node, service, agent)
			require.NoError(t, err)
			require.Len(t, actual, len(expected))
			for i := 0; i < len(expected); i++ {
				assertScrappedConfigsEqual(t, expected[i], actual[i])
			}
		})

		t.Run("BadCustomLabels", func(t *testing.T) {
			node := &models.Node{}
			service := &models.Service{}
			agent := &models.Agent{
				CustomLabels: []byte("{"),
				ListenPort:   pointer.ToUint16(12345),
			}

			_, err := scrapeConfigsForMySQLdExporter(s, "4.5.6.7", node, service, agent)
			require.EqualError(t, err, "failed to decode custom labels: unexpected end of JSON input")
		})
	})

	t.Run("scrapeConfigForMongoDBExporter", func(t *testing.T) {
		t.Run("Normal", func(t *testing.T) {
			node := &models.Node{
				NodeID:       "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
				NodeName:     "node_name",
				Address:      "1.2.3.4",
				CustomLabels: []byte(`{"_some_node_label": "foo"}`),
			}
			service := &models.Service{
				ServiceID:    "/service_id/014647c3-b2f5-44eb-94f4-d943260a968c",
				NodeID:       "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
				Address:      pointer.ToString("5.6.7.8"),
				CustomLabels: []byte(`{"_some_service_label": "bar"}`),
			}
			agent := &models.Agent{
				AgentID:      "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
				AgentType:    models.MongoDBExporterType,
				RunsOnNodeID: nil,
				CustomLabels: []byte(`{"_some_agent_label": "baz"}`),
				ListenPort:   pointer.ToUint16(12345),
			}

			expected := &config.ScrapeConfig{
				JobName:        "mongodb_exporter_agent_id_75bb30d3-ef4a-4147-97a8-621a996611dd",
				ScrapeInterval: model.Duration(time.Second),
				ScrapeTimeout:  model.Duration(time.Second),
				MetricsPath:    "/metrics",
				HTTPClientConfig: config_util.HTTPClientConfig{
					BasicAuth: &config_util.BasicAuth{
						Username: "pmm",
						Password: "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
					},
				},
				ServiceDiscoveryConfig: sd_config.ServiceDiscoveryConfig{
					StaticConfigs: []*targetgroup.Group{{
						Targets: []model.LabelSet{{"__address__": "4.5.6.7:12345"}},
						Labels: model.LabelSet{
							"_some_agent_label":   "baz",
							"_some_node_label":    "foo",
							"_some_service_label": "bar",
							"agent_id":            "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
							"agent_type":          "mongodb_exporter",
							"instance":            "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
							"node_id":             "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
							"node_name":           "node_name",
							"service_id":          "/service_id/014647c3-b2f5-44eb-94f4-d943260a968c",
						},
					}},
				},
			}

			actual, err := scrapeConfigForMongoDBExporter(time.Second, "4.5.6.7", node, service, agent)
			require.NoError(t, err)
			assertScrappedConfigsEqual(t, expected, actual)
		})

		t.Run("BadCustomLabels", func(t *testing.T) {
			node := &models.Node{}
			service := &models.Service{}
			agent := &models.Agent{
				CustomLabels: []byte("{"),
				ListenPort:   pointer.ToUint16(12345),
			}

			_, err := scrapeConfigForMongoDBExporter(time.Second, "4.5.6.7", node, service, agent)
			require.EqualError(t, err, "failed to decode custom labels: unexpected end of JSON input")
		})
	})

	t.Run("scrapeConfigForPostgresExporter", func(t *testing.T) {
		t.Run("Normal", func(t *testing.T) {
			node := &models.Node{
				NodeID:       "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
				NodeName:     "node_name",
				Address:      "1.2.3.4",
				CustomLabels: []byte(`{"_some_node_label": "foo"}`),
			}
			service := &models.Service{
				ServiceID:    "/service_id/014647c3-b2f5-44eb-94f4-d943260a968c",
				NodeID:       "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
				Address:      pointer.ToString("5.6.7.8"),
				CustomLabels: []byte(`{"_some_service_label": "bar"}`),
			}
			agent := &models.Agent{
				AgentID:      "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
				AgentType:    models.PostgresExporterType,
				RunsOnNodeID: nil,
				CustomLabels: []byte(`{"_some_agent_label": "baz"}`),
				ListenPort:   pointer.ToUint16(12345),
			}

			expected := &config.ScrapeConfig{
				JobName:        "postgres_exporter_agent_id_75bb30d3-ef4a-4147-97a8-621a996611dd",
				ScrapeInterval: model.Duration(time.Second),
				ScrapeTimeout:  model.Duration(time.Second),
				MetricsPath:    "/metrics",
				HTTPClientConfig: config_util.HTTPClientConfig{
					BasicAuth: &config_util.BasicAuth{
						Username: "pmm",
						Password: "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
					},
				},
				Params: url.Values{
					"collect[]": []string{"exporter"},
				},
				ServiceDiscoveryConfig: sd_config.ServiceDiscoveryConfig{
					StaticConfigs: []*targetgroup.Group{{
						Targets: []model.LabelSet{{"__address__": "4.5.6.7:12345"}},
						Labels: model.LabelSet{
							"_some_agent_label":   "baz",
							"_some_node_label":    "foo",
							"_some_service_label": "bar",
							"agent_id":            "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
							"agent_type":          "postgres_exporter",
							"instance":            "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
							"node_id":             "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
							"node_name":           "node_name",
							"service_id":          "/service_id/014647c3-b2f5-44eb-94f4-d943260a968c",
						},
					}},
				},
			}

			actual, err := scrapeConfigForPostgresExporter(time.Second, "4.5.6.7", node, service, agent)
			require.NoError(t, err)
			assertScrappedConfigsEqual(t, expected, actual)
		})

		t.Run("BadCustomLabels", func(t *testing.T) {
			node := &models.Node{}
			service := &models.Service{}
			agent := &models.Agent{
				CustomLabels: []byte("{"),
				ListenPort:   pointer.ToUint16(12345),
			}

			_, err := scrapeConfigForPostgresExporter(time.Second, "4.5.6.7", node, service, agent)
			require.EqualError(t, err, "failed to decode custom labels: unexpected end of JSON input")
		})
	})

	t.Run("scrapeConfigForProxySQLExporter", func(t *testing.T) {
		t.Run("Normal", func(t *testing.T) {
			node := &models.Node{
				NodeID:       "/node_id/7cc6ec12-4951-48c6-a4d5-7c3141fa4107",
				NodeName:     "node_name",
				Address:      "1.2.3.4",
				CustomLabels: []byte(`{"_some_node_label": "foo"}`),
			}
			service := &models.Service{
				ServiceID:    "/service_id/56fa3285-4476-49cc-95ff-96b36c24b0b6",
				NodeID:       "/node_id/7cc6ec12-4951-48c6-a4d5-7c3141fa4107",
				Address:      pointer.ToString("5.6.7.8"),
				CustomLabels: []byte(`{"_some_service_label": "bar"}`),
			}
			agent := &models.Agent{
				AgentID:      "/agent_id/782589c6-d3af-45e5-aa20-7f664a690940",
				AgentType:    models.ProxySQLExporterType,
				RunsOnNodeID: nil,
				CustomLabels: []byte(`{"_some_agent_label": "baz"}`),
				ListenPort:   pointer.ToUint16(12345),
			}

			expected := &config.ScrapeConfig{
				JobName:        "proxysql_exporter_agent_id_782589c6-d3af-45e5-aa20-7f664a690940",
				ScrapeInterval: model.Duration(time.Second),
				ScrapeTimeout:  model.Duration(time.Second),
				MetricsPath:    "/metrics",
				HTTPClientConfig: config_util.HTTPClientConfig{
					BasicAuth: &config_util.BasicAuth{
						Username: "pmm",
						Password: "/agent_id/782589c6-d3af-45e5-aa20-7f664a690940",
					},
				},
				ServiceDiscoveryConfig: sd_config.ServiceDiscoveryConfig{
					StaticConfigs: []*targetgroup.Group{{
						Targets: []model.LabelSet{{"__address__": "4.5.6.7:12345"}},
						Labels: model.LabelSet{
							"_some_agent_label":   "baz",
							"_some_node_label":    "foo",
							"_some_service_label": "bar",
							"agent_id":            "/agent_id/782589c6-d3af-45e5-aa20-7f664a690940",
							"agent_type":          "proxysql_exporter",
							"instance":            "/agent_id/782589c6-d3af-45e5-aa20-7f664a690940",
							"node_id":             "/node_id/7cc6ec12-4951-48c6-a4d5-7c3141fa4107",
							"node_name":           "node_name",
							"service_id":          "/service_id/56fa3285-4476-49cc-95ff-96b36c24b0b6",
						},
					}},
				},
			}

			actual, err := scrapeConfigForProxySQLExporter(time.Second, "4.5.6.7", node, service, agent)
			require.NoError(t, err)
			assertScrappedConfigsEqual(t, expected, actual)
		})

		t.Run("BadCustomLabels", func(t *testing.T) {
			node := &models.Node{}
			service := &models.Service{}
			agent := &models.Agent{
				CustomLabels: []byte("{"),
				ListenPort:   pointer.ToUint16(12345),
			}

			_, err := scrapeConfigForProxySQLExporter(time.Second, node.Address, node, service, agent)
			require.EqualError(t, err, "failed to decode custom labels: unexpected end of JSON input")
		})
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
