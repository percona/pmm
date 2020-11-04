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
	config "github.com/percona/promconfig"
	"github.com/pmezard/go-difflib/difflib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/percona/pmm-managed/models"
)

func TestScrapeConfig(t *testing.T) {
	s := &models.MetricsResolutions{
		HR: 5 * time.Second,
		MR: 5 * time.Second,
		LR: 60 * time.Second,
	}

	t.Run("scrapeConfigsForNodeExporter", func(t *testing.T) {
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
				CustomLabels: []byte(`{"_some_agent_label": "baz"}`),
				ListenPort:   pointer.ToUint16(12345),
			}

			expected := []*config.ScrapeConfig{{
				JobName:        "node_exporter_agent_id_75bb30d3-ef4a-4147-97a8-621a996611dd_hr-5s",
				ScrapeInterval: config.Duration(s.HR),
				ScrapeTimeout:  scrapeTimeout(s.HR),
				MetricsPath:    "/metrics",
				HTTPClientConfig: config.HTTPClientConfig{
					BasicAuth: &config.BasicAuth{
						Username: "pmm",
						Password: "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
					},
				},
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.Group{{
						Targets: []string{"1.2.3.4:12345"},
						Labels: map[string]string{
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
				Params: url.Values{"collect[]": []string{
					"buddyinfo",
					"cpu",
					"diskstats",
					"filefd",
					"filesystem",
					"loadavg",
					"meminfo",
					"meminfo_numa",
					"netdev",
					"netstat",
					"processes",
					"standard.go",
					"standard.process",
					"stat",
					"textfile.hr",
					"time",
					"vmstat",
				}},
			}, {
				JobName:        "node_exporter_agent_id_75bb30d3-ef4a-4147-97a8-621a996611dd_mr-5s",
				ScrapeInterval: config.Duration(s.MR),
				ScrapeTimeout:  scrapeTimeout(s.MR),
				MetricsPath:    "/metrics",
				HTTPClientConfig: config.HTTPClientConfig{
					BasicAuth: &config.BasicAuth{
						Username: "pmm",
						Password: "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
					},
				},
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.Group{{
						Targets: []string{"1.2.3.4:12345"},
						Labels: map[string]string{
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
				Params: url.Values{"collect[]": []string{
					"hwmon",
					"textfile.mr",
				}},
			}, {
				JobName:        "node_exporter_agent_id_75bb30d3-ef4a-4147-97a8-621a996611dd_lr-1m0s",
				ScrapeInterval: config.Duration(s.LR),
				ScrapeTimeout:  scrapeTimeout(s.LR),
				MetricsPath:    "/metrics",
				HTTPClientConfig: config.HTTPClientConfig{
					BasicAuth: &config.BasicAuth{
						Username: "pmm",
						Password: "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
					},
				},
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.Group{{
						Targets: []string{"1.2.3.4:12345"},
						Labels: map[string]string{
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
				Params: url.Values{"collect[]": []string{
					"bonding",
					"entropy",
					"textfile.lr",
					"uname",
				}},
			}}

			actual, err := scrapeConfigsForNodeExporter(s, &scrapeConfigParams{
				host:  "1.2.3.4",
				node:  node,
				agent: agent,
			})

			require.NoError(t, err)
			require.Len(t, actual, len(expected))
			for i := 0; i < len(expected); i++ {
				assertScrapeConfigsEqual(t, expected[i], actual[i])
			}
		})

		t.Run("MacOS", func(t *testing.T) {
			node := &models.Node{
				NodeID:       "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
				NodeName:     "node_name",
				Distro:       "darwin",
				Address:      "1.2.3.4",
				CustomLabels: []byte(`{"_some_node_label": "foo"}`),
			}
			agent := &models.Agent{
				AgentID:      "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
				AgentType:    models.NodeExporterType,
				CustomLabels: []byte(`{"_some_agent_label": "baz"}`),
				ListenPort:   pointer.ToUint16(12345),
			}

			expected := []*config.ScrapeConfig{{
				JobName:        "node_exporter_agent_id_75bb30d3-ef4a-4147-97a8-621a996611dd_hr-5s",
				ScrapeInterval: config.Duration(s.HR),
				ScrapeTimeout:  scrapeTimeout(s.HR),
				MetricsPath:    "/metrics",
				HTTPClientConfig: config.HTTPClientConfig{
					BasicAuth: &config.BasicAuth{
						Username: "pmm",
						Password: "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
					},
				},
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.Group{{
						Targets: []string{"1.2.3.4:12345"},
						Labels: map[string]string{
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
				Params: url.Values{"collect[]": []string{
					"cpu",
					"diskstats",
					"filesystem",
					"loadavg",
					"meminfo",
					"netdev",
					"time",
				}},
			}}

			actual, err := scrapeConfigsForNodeExporter(s, &scrapeConfigParams{
				host:  "1.2.3.4",
				node:  node,
				agent: agent,
			})

			require.NoError(t, err)
			require.Len(t, actual, len(expected))
			for i := 0; i < len(expected); i++ {
				assertScrapeConfigsEqual(t, expected[i], actual[i])
			}
		})
	})

	t.Run("scrapeConfigsForMySQLdExporter", func(t *testing.T) {
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
				CustomLabels: []byte(`{"_some_agent_label": "baz"}`),
				ListenPort:   pointer.ToUint16(12345),
			}

			expected := []*config.ScrapeConfig{{
				JobName:        "mysqld_exporter_agent_id_75bb30d3-ef4a-4147-97a8-621a996611dd_hr-5s",
				ScrapeInterval: config.Duration(s.HR),
				ScrapeTimeout:  scrapeTimeout(s.HR),
				MetricsPath:    "/metrics",
				HTTPClientConfig: config.HTTPClientConfig{
					BasicAuth: &config.BasicAuth{
						Username: "pmm",
						Password: "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
					},
				},
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.Group{{
						Targets: []string{"4.5.6.7:12345"},
						Labels: map[string]string{
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
				Params: url.Values{"collect[]": []string{
					"custom_query.hr",
					"global_status",
					"info_schema.innodb_metrics",
					"standard.go",
					"standard.process",
				}},
			}, {
				JobName:        "mysqld_exporter_agent_id_75bb30d3-ef4a-4147-97a8-621a996611dd_mr-5s",
				ScrapeInterval: config.Duration(s.MR),
				ScrapeTimeout:  scrapeTimeout(s.MR),
				MetricsPath:    "/metrics",
				HTTPClientConfig: config.HTTPClientConfig{
					BasicAuth: &config.BasicAuth{
						Username: "pmm",
						Password: "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
					},
				},
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.Group{{
						Targets: []string{"4.5.6.7:12345"},
						Labels: map[string]string{
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
				Params: url.Values{"collect[]": []string{
					"custom_query.mr",
					"engine_innodb_status",
					"info_schema.innodb_cmp",
					"info_schema.innodb_cmpmem",
					"info_schema.processlist",
					"info_schema.query_response_time",
					"perf_schema.eventswaits",
					"perf_schema.file_events",
					"perf_schema.tablelocks",
					"slave_status",
				}},
			}, {
				JobName:        "mysqld_exporter_agent_id_75bb30d3-ef4a-4147-97a8-621a996611dd_lr-1m0s",
				ScrapeInterval: config.Duration(s.LR),
				ScrapeTimeout:  scrapeTimeout(s.LR),
				MetricsPath:    "/metrics",
				HTTPClientConfig: config.HTTPClientConfig{
					BasicAuth: &config.BasicAuth{
						Username: "pmm",
						Password: "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
					},
				},
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.Group{{
						Targets: []string{"4.5.6.7:12345"},
						Labels: map[string]string{
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
				Params: url.Values{"collect[]": []string{
					"auto_increment.columns",
					"binlog_size",
					"custom_query.lr",
					"engine_tokudb_status",
					"global_variables",
					"heartbeat",
					"info_schema.clientstats",
					"info_schema.innodb_tablespaces",
					"info_schema.tables",
					"info_schema.tablestats",
					"info_schema.userstats",
					"perf_schema.eventsstatements",
					"perf_schema.file_instances",
					"perf_schema.indexiowaits",
					"perf_schema.tableiowaits",
				}},
			}}

			actual, err := scrapeConfigsForMySQLdExporter(s, &scrapeConfigParams{
				host:    "4.5.6.7",
				node:    node,
				service: service,
				agent:   agent,
			})
			require.NoError(t, err)
			require.Len(t, actual, len(expected))
			for i := 0; i < len(expected); i++ {
				assertScrapeConfigsEqual(t, expected[i], actual[i])
			}
		})

		t.Run("ManyTables", func(t *testing.T) {
			node := &models.Node{
				NodeID:   "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
				NodeName: "node_name",
				Address:  "1.2.3.4",
			}
			service := &models.Service{
				ServiceID: "/service_id/014647c3-b2f5-44eb-94f4-d943260a968c",
				NodeID:    "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
				Address:   pointer.ToString("5.6.7.8"),
			}
			agent := &models.Agent{
				AgentID:                        "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
				AgentType:                      models.MySQLdExporterType,
				ListenPort:                     pointer.ToUint16(12345),
				TableCount:                     pointer.ToInt32(100500),
				TableCountTablestatsGroupLimit: 1000,
			}

			expected := []*config.ScrapeConfig{{
				JobName:        "mysqld_exporter_agent_id_75bb30d3-ef4a-4147-97a8-621a996611dd_hr-5s",
				ScrapeInterval: config.Duration(s.HR),
				ScrapeTimeout:  scrapeTimeout(s.HR),
				MetricsPath:    "/metrics",
				HTTPClientConfig: config.HTTPClientConfig{
					BasicAuth: &config.BasicAuth{
						Username: "pmm",
						Password: "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
					},
				},
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.Group{{
						Targets: []string{"4.5.6.7:12345"},
						Labels: map[string]string{
							"agent_id":   "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
							"agent_type": "mysqld_exporter",
							"instance":   "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
							"node_id":    "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
							"node_name":  "node_name",
							"service_id": "/service_id/014647c3-b2f5-44eb-94f4-d943260a968c",
						},
					}},
				},
				Params: url.Values{"collect[]": []string{
					"custom_query.hr",
					"global_status",
					"info_schema.innodb_metrics",
					"standard.go",
					"standard.process",
				}},
			}, {
				JobName:        "mysqld_exporter_agent_id_75bb30d3-ef4a-4147-97a8-621a996611dd_mr-5s",
				ScrapeInterval: config.Duration(s.MR),
				ScrapeTimeout:  scrapeTimeout(s.MR),
				MetricsPath:    "/metrics",
				HTTPClientConfig: config.HTTPClientConfig{
					BasicAuth: &config.BasicAuth{
						Username: "pmm",
						Password: "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
					},
				},
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.Group{{
						Targets: []string{"4.5.6.7:12345"},
						Labels: map[string]string{
							"agent_id":   "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
							"agent_type": "mysqld_exporter",
							"instance":   "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
							"node_id":    "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
							"node_name":  "node_name",
							"service_id": "/service_id/014647c3-b2f5-44eb-94f4-d943260a968c",
						},
					}},
				},
				Params: url.Values{"collect[]": []string{
					"custom_query.mr",
					"engine_innodb_status",
					"info_schema.innodb_cmp",
					"info_schema.innodb_cmpmem",
					"info_schema.processlist",
					"info_schema.query_response_time",
					"perf_schema.eventswaits",
					"perf_schema.file_events",
					"slave_status",
				}},
			}, {
				JobName:        "mysqld_exporter_agent_id_75bb30d3-ef4a-4147-97a8-621a996611dd_lr-1m0s",
				ScrapeInterval: config.Duration(s.LR),
				ScrapeTimeout:  scrapeTimeout(s.LR),
				MetricsPath:    "/metrics",
				HTTPClientConfig: config.HTTPClientConfig{
					BasicAuth: &config.BasicAuth{
						Username: "pmm",
						Password: "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
					},
				},
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.Group{{
						Targets: []string{"4.5.6.7:12345"},
						Labels: map[string]string{
							"agent_id":   "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
							"agent_type": "mysqld_exporter",
							"instance":   "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
							"node_id":    "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
							"node_name":  "node_name",
							"service_id": "/service_id/014647c3-b2f5-44eb-94f4-d943260a968c",
						},
					}},
				},
				Params: url.Values{"collect[]": []string{
					"binlog_size",
					"custom_query.lr",
					"engine_tokudb_status",
					"global_variables",
					"heartbeat",
					"info_schema.clientstats",
					"info_schema.innodb_tablespaces",
					"info_schema.userstats",
					"perf_schema.eventsstatements",
					"perf_schema.file_instances",
				}},
			}}

			actual, err := scrapeConfigsForMySQLdExporter(s, &scrapeConfigParams{
				host:    "4.5.6.7",
				node:    node,
				service: service,
				agent:   agent,
			})
			require.NoError(t, err)
			require.Len(t, actual, len(expected))
			for i := 0; i < len(expected); i++ {
				assertScrapeConfigsEqual(t, expected[i], actual[i])
			}
		})

		t.Run("BadCustomLabels", func(t *testing.T) {
			node := &models.Node{}
			service := &models.Service{}
			agent := &models.Agent{
				CustomLabels: []byte("{"),
				ListenPort:   pointer.ToUint16(12345),
			}

			_, err := scrapeConfigsForMySQLdExporter(s, &scrapeConfigParams{
				host:    "4.5.6.7",
				node:    node,
				service: service,
				agent:   agent,
			})
			require.EqualError(t, err, "failed to decode custom labels: unexpected end of JSON input")
		})
	})

	t.Run("scrapeConfigsForMongoDBExporter", func(t *testing.T) {
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
				CustomLabels: []byte(`{"_some_agent_label": "baz"}`),
				ListenPort:   pointer.ToUint16(12345),
			}

			expected := []*config.ScrapeConfig{{
				JobName:        "mongodb_exporter_agent_id_75bb30d3-ef4a-4147-97a8-621a996611dd_hr-5s",
				ScrapeInterval: config.Duration(s.HR),
				ScrapeTimeout:  scrapeTimeout(s.HR),
				MetricsPath:    "/metrics",
				HTTPClientConfig: config.HTTPClientConfig{
					BasicAuth: &config.BasicAuth{
						Username: "pmm",
						Password: "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
					},
				},
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.Group{{
						Targets: []string{"4.5.6.7:12345"},
						Labels: map[string]string{
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
			}}

			actual, err := scrapeConfigsForMongoDBExporter(s, &scrapeConfigParams{
				host:    "4.5.6.7",
				node:    node,
				service: service,
				agent:   agent,
			})
			require.NoError(t, err)
			require.Len(t, actual, len(expected))
			for i := 0; i < len(expected); i++ {
				assertScrapeConfigsEqual(t, expected[i], actual[i])
			}
		})

		t.Run("BadCustomLabels", func(t *testing.T) {
			node := &models.Node{}
			service := &models.Service{}
			agent := &models.Agent{
				CustomLabels: []byte("{"),
				ListenPort:   pointer.ToUint16(12345),
			}

			_, err := scrapeConfigsForMongoDBExporter(s, &scrapeConfigParams{
				host:    "4.5.6.7",
				node:    node,
				service: service,
				agent:   agent,
			})
			require.EqualError(t, err, "failed to decode custom labels: unexpected end of JSON input")
		})
	})

	t.Run("scrapeConfigsForPostgresExporter", func(t *testing.T) {
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
				CustomLabels: []byte(`{"_some_agent_label": "baz"}`),
				ListenPort:   pointer.ToUint16(12345),
			}

			expected := []*config.ScrapeConfig{{
				JobName:        "postgres_exporter_agent_id_75bb30d3-ef4a-4147-97a8-621a996611dd_hr-5s",
				ScrapeInterval: config.Duration(s.HR),
				ScrapeTimeout:  scrapeTimeout(s.HR),
				MetricsPath:    "/metrics",
				HTTPClientConfig: config.HTTPClientConfig{
					BasicAuth: &config.BasicAuth{
						Username: "pmm",
						Password: "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
					},
				},
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.Group{{
						Targets: []string{"4.5.6.7:12345"},
						Labels: map[string]string{
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
				Params: url.Values{"collect[]": []string{
					"custom_query.hr",
					"exporter",
					"standard.go",
					"standard.process",
				}},
			}, {
				JobName:        "postgres_exporter_agent_id_75bb30d3-ef4a-4147-97a8-621a996611dd_mr-5s",
				ScrapeInterval: config.Duration(s.MR),
				ScrapeTimeout:  scrapeTimeout(s.MR),
				MetricsPath:    "/metrics",
				HTTPClientConfig: config.HTTPClientConfig{
					BasicAuth: &config.BasicAuth{
						Username: "pmm",
						Password: "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
					},
				},
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.Group{{
						Targets: []string{"4.5.6.7:12345"},
						Labels: map[string]string{
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
				Params: url.Values{"collect[]": []string{
					"custom_query.mr",
				}},
			}, {
				JobName:        "postgres_exporter_agent_id_75bb30d3-ef4a-4147-97a8-621a996611dd_lr-1m0s",
				ScrapeInterval: config.Duration(s.LR),
				ScrapeTimeout:  scrapeTimeout(s.LR),
				MetricsPath:    "/metrics",
				HTTPClientConfig: config.HTTPClientConfig{
					BasicAuth: &config.BasicAuth{
						Username: "pmm",
						Password: "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
					},
				},
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.Group{{
						Targets: []string{"4.5.6.7:12345"},
						Labels: map[string]string{
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
				Params: url.Values{"collect[]": []string{
					"custom_query.lr",
				}},
			}}

			actual, err := scrapeConfigsForPostgresExporter(s, &scrapeConfigParams{
				host:    "4.5.6.7",
				node:    node,
				service: service,
				agent:   agent,
			})
			require.NoError(t, err)
			require.Len(t, actual, len(expected))
			for i := 0; i < len(expected); i++ {
				assertScrapeConfigsEqual(t, expected[i], actual[i])
			}
		})

		t.Run("BadCustomLabels", func(t *testing.T) {
			node := &models.Node{}
			service := &models.Service{}
			agent := &models.Agent{
				CustomLabels: []byte("{"),
				ListenPort:   pointer.ToUint16(12345),
			}

			_, err := scrapeConfigsForPostgresExporter(s, &scrapeConfigParams{
				host:    "4.5.6.7",
				node:    node,
				service: service,
				agent:   agent,
			})
			require.EqualError(t, err, "failed to decode custom labels: unexpected end of JSON input")
		})
	})

	t.Run("scrapeConfigsForProxySQLExporter", func(t *testing.T) {
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
				AgentType:    models.ProxySQLExporterType,
				CustomLabels: []byte(`{"_some_agent_label": "baz"}`),
				ListenPort:   pointer.ToUint16(12345),
			}

			expected := []*config.ScrapeConfig{{
				JobName:        "proxysql_exporter_agent_id_75bb30d3-ef4a-4147-97a8-621a996611dd_hr-5s",
				ScrapeInterval: config.Duration(s.HR),
				ScrapeTimeout:  scrapeTimeout(s.HR),
				MetricsPath:    "/metrics",
				HTTPClientConfig: config.HTTPClientConfig{
					BasicAuth: &config.BasicAuth{
						Username: "pmm",
						Password: "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
					},
				},
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.Group{{
						Targets: []string{"4.5.6.7:12345"},
						Labels: map[string]string{
							"_some_agent_label":   "baz",
							"_some_node_label":    "foo",
							"_some_service_label": "bar",
							"agent_id":            "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
							"agent_type":          "proxysql_exporter",
							"instance":            "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
							"node_id":             "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
							"node_name":           "node_name",
							"service_id":          "/service_id/014647c3-b2f5-44eb-94f4-d943260a968c",
						},
					}},
				},
			}}

			actual, err := scrapeConfigsForProxySQLExporter(s, &scrapeConfigParams{
				host:    "4.5.6.7",
				node:    node,
				service: service,
				agent:   agent,
			})
			require.NoError(t, err)
			require.Len(t, actual, len(expected))
			for i := 0; i < len(expected); i++ {
				assertScrapeConfigsEqual(t, expected[i], actual[i])
			}
		})

		t.Run("BadCustomLabels", func(t *testing.T) {
			node := &models.Node{}
			service := &models.Service{}
			agent := &models.Agent{
				CustomLabels: []byte("{"),
				ListenPort:   pointer.ToUint16(12345),
			}

			_, err := scrapeConfigsForProxySQLExporter(s, &scrapeConfigParams{
				host:    "4.5.6.7",
				node:    node,
				service: service,
				agent:   agent,
			})
			require.EqualError(t, err, "failed to decode custom labels: unexpected end of JSON input")
		})
	})

	t.Run("scrapeConfigsForRDSExporter", func(t *testing.T) {
		t.Run("Normal", func(t *testing.T) {
			params := []*scrapeConfigParams{
				// two RDS configs on the same host/port combination: single pmm-agent, single rds_exporter process
				{
					host:  "1.1.1.1",
					agent: &models.Agent{ListenPort: pointer.ToUint16(12345)},
				},
				{
					host:  "1.1.1.1",
					agent: &models.Agent{ListenPort: pointer.ToUint16(12345)},
				},

				// two RDS configs on the same host, different ports: two pmm-agents, two rds_exporter processes
				{
					host:  "2.2.2.2",
					agent: &models.Agent{ListenPort: pointer.ToUint16(12345)},
				},
				{
					host:  "2.2.2.2",
					agent: &models.Agent{ListenPort: pointer.ToUint16(12346)},
				},
			}

			expected := []*config.ScrapeConfig{{
				JobName:        "rds_exporter_1_1_1_1_12345_mr-5s",
				ScrapeInterval: config.Duration(s.MR),
				ScrapeTimeout:  scrapeTimeout(s.MR),
				MetricsPath:    "/enhanced",
				HonorLabels:    true,
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.Group{{
						Targets: []string{"1.1.1.1:12345"},
					}},
				},
			}, {
				JobName:        "rds_exporter_1_1_1_1_12345_lr-1m0s",
				ScrapeInterval: config.Duration(s.LR),
				ScrapeTimeout:  scrapeTimeout(s.LR),
				MetricsPath:    "/basic",
				HonorLabels:    true,
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.Group{{
						Targets: []string{"1.1.1.1:12345"},
					}},
				},
			}, {
				JobName:        "rds_exporter_2_2_2_2_12345_mr-5s",
				ScrapeInterval: config.Duration(s.MR),
				ScrapeTimeout:  scrapeTimeout(s.MR),
				MetricsPath:    "/enhanced",
				HonorLabels:    true,
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.Group{{
						Targets: []string{"2.2.2.2:12345"},
					}},
				},
			}, {
				JobName:        "rds_exporter_2_2_2_2_12345_lr-1m0s",
				ScrapeInterval: config.Duration(s.LR),
				ScrapeTimeout:  scrapeTimeout(s.LR),
				MetricsPath:    "/basic",
				HonorLabels:    true,
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.Group{{
						Targets: []string{"2.2.2.2:12345"},
					}},
				},
			}, {
				JobName:        "rds_exporter_2_2_2_2_12346_mr-5s",
				ScrapeInterval: config.Duration(s.MR),
				ScrapeTimeout:  scrapeTimeout(s.MR),
				MetricsPath:    "/enhanced",
				HonorLabels:    true,
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.Group{{
						Targets: []string{"2.2.2.2:12346"},
					}},
				},
			}, {
				JobName:        "rds_exporter_2_2_2_2_12346_lr-1m0s",
				ScrapeInterval: config.Duration(s.LR),
				ScrapeTimeout:  scrapeTimeout(s.LR),
				MetricsPath:    "/basic",
				HonorLabels:    true,
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.Group{{
						Targets: []string{"2.2.2.2:12346"},
					}},
				},
			}}

			actual := scrapeConfigsForRDSExporter(s, params)
			require.Len(t, actual, len(expected))
			for i := 0; i < len(expected); i++ {
				assertScrapeConfigsEqual(t, expected[i], actual[i])
			}
		})
	})

	t.Run("scrapeConfigsForExternalExporter", func(t *testing.T) {
		node := &models.Node{
			NodeID:       "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
			NodeName:     "node_name",
			Address:      "1.2.3.4",
			CustomLabels: []byte(`{"_some_node_label": "foo"}`),
		}
		service := &models.Service{
			ServiceID:     "/service_id/014647c3-b2f5-44eb-94f4-d943260a968c",
			NodeID:        "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
			Address:       pointer.ToString("5.6.7.8"),
			CustomLabels:  []byte(`{"_some_service_label": "bar"}`),
			ExternalGroup: "rabbitmq",
		}
		t.Run("Normal", func(t *testing.T) {
			agent := &models.Agent{
				AgentID:      "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
				AgentType:    models.ExternalExporterType,
				CustomLabels: []byte(`{"_some_agent_label": "baz"}`),
				ListenPort:   pointer.ToUint16(12345),
			}

			expected := []*config.ScrapeConfig{{
				JobName:        "external-exporter_agent_id_75bb30d3-ef4a-4147-97a8-621a996611dd_mr-5s",
				ScrapeInterval: config.Duration(s.HR),
				ScrapeTimeout:  scrapeTimeout(s.HR),
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.Group{{
						Targets: []string{"4.5.6.7:12345"},
						Labels: map[string]string{
							"_some_agent_label":   "baz",
							"_some_node_label":    "foo",
							"_some_service_label": "bar",
							"agent_id":            "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
							"agent_type":          "external-exporter",
							"external_group":      "rabbitmq",
							"instance":            "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
							"node_id":             "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
							"node_name":           "node_name",
							"service_id":          "/service_id/014647c3-b2f5-44eb-94f4-d943260a968c",
						},
					}},
				},
			}}

			actual, err := scrapeConfigsForExternalExporter(s, &scrapeConfigParams{
				host:    "4.5.6.7",
				node:    node,
				service: service,
				agent:   agent,
			})
			require.NoError(t, err)
			require.Len(t, actual, len(expected))
			for i := 0; i < len(expected); i++ {
				assertScrapeConfigsEqual(t, expected[i], actual[i])
			}
		})

		t.Run("WithExtraParams", func(t *testing.T) {
			agent := &models.Agent{
				AgentID:       "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
				AgentType:     models.ExternalExporterType,
				CustomLabels:  []byte(`{"_some_agent_label": "baz"}`),
				Username:      pointer.ToString("username"),
				Password:      pointer.ToString("password"),
				ListenPort:    pointer.ToUint16(12345),
				MetricsPath:   pointer.ToString("/some-metric-path"),
				MetricsScheme: pointer.ToString("https"),
			}

			expected := []*config.ScrapeConfig{{
				JobName:        "external-exporter_agent_id_75bb30d3-ef4a-4147-97a8-621a996611dd_mr-5s",
				ScrapeInterval: config.Duration(s.HR),
				ScrapeTimeout:  scrapeTimeout(s.HR),
				MetricsPath:    "/some-metric-path",
				Scheme:         "https",
				HTTPClientConfig: config.HTTPClientConfig{
					BasicAuth: &config.BasicAuth{
						Username: "username",
						Password: "password",
					},
				},
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.Group{{
						Targets: []string{"4.5.6.7:12345"},
						Labels: map[string]string{
							"_some_agent_label":   "baz",
							"_some_node_label":    "foo",
							"_some_service_label": "bar",
							"agent_id":            "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
							"agent_type":          "external-exporter",
							"external_group":      "rabbitmq",
							"instance":            "/agent_id/75bb30d3-ef4a-4147-97a8-621a996611dd",
							"node_id":             "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
							"node_name":           "node_name",
							"service_id":          "/service_id/014647c3-b2f5-44eb-94f4-d943260a968c",
						},
					}},
				},
			}}

			actual, err := scrapeConfigsForExternalExporter(s, &scrapeConfigParams{
				host:    "4.5.6.7",
				node:    node,
				service: service,
				agent:   agent,
			})
			require.NoError(t, err)
			require.Len(t, actual, len(expected))
			for i := 0; i < len(expected); i++ {
				assertScrapeConfigsEqual(t, expected[i], actual[i])
			}
		})

		t.Run("BadCustomLabels", func(t *testing.T) {
			agent := &models.Agent{
				CustomLabels: []byte("{"),
				ListenPort:   pointer.ToUint16(12345),
			}

			_, err := scrapeConfigsForMongoDBExporter(s, &scrapeConfigParams{
				host:    "4.5.6.7",
				node:    node,
				service: service,
				agent:   agent,
			})
			require.EqualError(t, err, "failed to decode custom labels: unexpected end of JSON input")
		})
	})
}

func assertScrapeConfigsEqual(t *testing.T, expected, actual *config.ScrapeConfig) {
	t.Helper()

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
	}
}
