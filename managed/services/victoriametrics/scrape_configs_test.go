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

package victoriametrics

import (
	"net/url"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	config "github.com/percona/promconfig"
	"github.com/pmezard/go-difflib/difflib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/version"
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
				NodeID:       "cc663f36-18ca-40a1-aea9-c6310bb4738d",
				NodeName:     "node_name",
				Address:      "1.2.3.4",
				CustomLabels: []byte(`{"_some_node_label": "foo"}`),
			}
			agent := &models.Agent{
				AgentID:      "75bb30d3-ef4a-4147-97a8-621a996611dd",
				AgentType:    models.NodeExporterType,
				CustomLabels: []byte(`{"_some_agent_label": "baz"}`),
				ListenPort:   pointer.ToUint16(12345),
				ExporterOptions: &models.ExporterOptions{
					DisabledCollectors: []string{"cpu", "entropy"},
				},
			}

			expected := []*config.ScrapeConfig{{
				JobName:        "node_exporter75bb30d3-ef4a-4147-97a8-621a996611dd_hr",
				ScrapeInterval: config.Duration(s.HR),
				ScrapeTimeout:  scrapeTimeout(s.HR),
				MetricsPath:    "/metrics",
				HTTPClientConfig: config.HTTPClientConfig{
					BasicAuth: &config.BasicAuth{
						Username: "pmm",
						Password: "75bb30d3-ef4a-4147-97a8-621a996611dd",
					},
				},
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.Group{{
						Targets: []string{"1.2.3.4:12345"},
						Labels: map[string]string{
							"_some_agent_label": "baz",
							"_some_node_label":  "foo",
							"agent_id":          "75bb30d3-ef4a-4147-97a8-621a996611dd",
							"agent_type":        "node_exporter",
							"instance":          "75bb30d3-ef4a-4147-97a8-621a996611dd",
							"node_id":           "cc663f36-18ca-40a1-aea9-c6310bb4738d",
							"node_name":         "node_name",
						},
					}},
				},
				Params: url.Values{"collect[]": []string{
					"buddyinfo",
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
				JobName:        "node_exporter75bb30d3-ef4a-4147-97a8-621a996611dd_mr",
				ScrapeInterval: config.Duration(s.MR),
				ScrapeTimeout:  scrapeTimeout(s.MR),
				MetricsPath:    "/metrics",
				HTTPClientConfig: config.HTTPClientConfig{
					BasicAuth: &config.BasicAuth{
						Username: "pmm",
						Password: "75bb30d3-ef4a-4147-97a8-621a996611dd",
					},
				},
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.Group{{
						Targets: []string{"1.2.3.4:12345"},
						Labels: map[string]string{
							"_some_agent_label": "baz",
							"_some_node_label":  "foo",
							"agent_id":          "75bb30d3-ef4a-4147-97a8-621a996611dd",
							"agent_type":        "node_exporter",
							"instance":          "75bb30d3-ef4a-4147-97a8-621a996611dd",
							"node_id":           "cc663f36-18ca-40a1-aea9-c6310bb4738d",
							"node_name":         "node_name",
						},
					}},
				},
				Params: url.Values{"collect[]": []string{
					"hwmon",
					"textfile.mr",
				}},
			}, {
				JobName:        "node_exporter75bb30d3-ef4a-4147-97a8-621a996611dd_lr",
				ScrapeInterval: config.Duration(s.LR),
				ScrapeTimeout:  scrapeTimeout(s.LR),
				MetricsPath:    "/metrics",
				HTTPClientConfig: config.HTTPClientConfig{
					BasicAuth: &config.BasicAuth{
						Username: "pmm",
						Password: "75bb30d3-ef4a-4147-97a8-621a996611dd",
					},
				},
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.Group{{
						Targets: []string{"1.2.3.4:12345"},
						Labels: map[string]string{
							"_some_agent_label": "baz",
							"_some_node_label":  "foo",
							"agent_id":          "75bb30d3-ef4a-4147-97a8-621a996611dd",
							"agent_type":        "node_exporter",
							"instance":          "75bb30d3-ef4a-4147-97a8-621a996611dd",
							"node_id":           "cc663f36-18ca-40a1-aea9-c6310bb4738d",
							"node_name":         "node_name",
						},
					}},
				},
				Params: url.Values{"collect[]": []string{
					"bonding",
					"os",
					"textfile.lr",
					"uname",
				}},
			}}

			actual, err := scrapeConfigsForNodeExporter(&scrapeConfigParams{
				host:              "1.2.3.4",
				node:              node,
				agent:             agent,
				metricsResolution: s,
			})

			require.NoError(t, err)
			require.Len(t, actual, len(expected))
			for i := 0; i < len(expected); i++ {
				assertScrapeConfigsEqual(t, expected[i], actual[i])
			}
		})

		t.Run("MacOS", func(t *testing.T) {
			node := &models.Node{
				NodeID:       "cc663f36-18ca-40a1-aea9-c6310bb4738d",
				NodeName:     "node_name",
				Distro:       "darwin",
				Address:      "1.2.3.4",
				CustomLabels: []byte(`{"_some_node_label": "foo"}`),
			}
			agent := &models.Agent{
				AgentID:      "75bb30d3-ef4a-4147-97a8-621a996611dd",
				AgentType:    models.NodeExporterType,
				CustomLabels: []byte(`{"_some_agent_label": "baz"}`),
				ListenPort:   pointer.ToUint16(12345),
				ExporterOptions: &models.ExporterOptions{
					DisabledCollectors: []string{"cpu", "time"},
				},
			}

			expected := []*config.ScrapeConfig{{
				JobName:        "node_exporter75bb30d3-ef4a-4147-97a8-621a996611dd_hr",
				ScrapeInterval: config.Duration(s.HR),
				ScrapeTimeout:  scrapeTimeout(s.HR),
				MetricsPath:    "/metrics",
				HTTPClientConfig: config.HTTPClientConfig{
					BasicAuth: &config.BasicAuth{
						Username: "pmm",
						Password: "75bb30d3-ef4a-4147-97a8-621a996611dd",
					},
				},
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.Group{{
						Targets: []string{"1.2.3.4:12345"},
						Labels: map[string]string{
							"_some_agent_label": "baz",
							"_some_node_label":  "foo",
							"agent_id":          "75bb30d3-ef4a-4147-97a8-621a996611dd",
							"agent_type":        "node_exporter",
							"instance":          "75bb30d3-ef4a-4147-97a8-621a996611dd",
							"node_id":           "cc663f36-18ca-40a1-aea9-c6310bb4738d",
							"node_name":         "node_name",
						},
					}},
				},
				Params: url.Values{"collect[]": []string{
					"diskstats",
					"filesystem",
					"loadavg",
					"meminfo",
					"netdev",
				}},
			}}

			actual, err := scrapeConfigsForNodeExporter(&scrapeConfigParams{
				host:              "1.2.3.4",
				node:              node,
				agent:             agent,
				metricsResolution: s,
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
				NodeID:       "cc663f36-18ca-40a1-aea9-c6310bb4738d",
				NodeName:     "node_name",
				Address:      "1.2.3.4",
				CustomLabels: []byte(`{"_some_node_label": "foo"}`),
			}
			service := &models.Service{
				ServiceID:    "014647c3-b2f5-44eb-94f4-d943260a968c",
				NodeID:       "cc663f36-18ca-40a1-aea9-c6310bb4738d",
				Address:      pointer.ToString("5.6.7.8"),
				CustomLabels: []byte(`{"_some_service_label": "bar"}`),
			}
			agent := &models.Agent{
				AgentID:      "75bb30d3-ef4a-4147-97a8-621a996611dd",
				AgentType:    models.MySQLdExporterType,
				CustomLabels: []byte(`{"_some_agent_label": "baz"}`),
				ListenPort:   pointer.ToUint16(12345),
			}

			expected := []*config.ScrapeConfig{{
				JobName:        "mysqld_exporter75bb30d3-ef4a-4147-97a8-621a996611dd_hr",
				ScrapeInterval: config.Duration(s.HR),
				ScrapeTimeout:  scrapeTimeout(s.HR),
				MetricsPath:    "/metrics",
				HTTPClientConfig: config.HTTPClientConfig{
					BasicAuth: &config.BasicAuth{
						Username: "pmm",
						Password: "75bb30d3-ef4a-4147-97a8-621a996611dd",
					},
				},
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.Group{{
						Targets: []string{"4.5.6.7:12345"},
						Labels: map[string]string{
							"_some_agent_label":   "baz",
							"_some_node_label":    "foo",
							"_some_service_label": "bar",
							"agent_id":            "75bb30d3-ef4a-4147-97a8-621a996611dd",
							"agent_type":          "mysqld_exporter",
							"instance":            "75bb30d3-ef4a-4147-97a8-621a996611dd",
							"node_id":             "cc663f36-18ca-40a1-aea9-c6310bb4738d",
							"node_name":           "node_name",
							"service_id":          "014647c3-b2f5-44eb-94f4-d943260a968c",
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
				JobName:        "mysqld_exporter75bb30d3-ef4a-4147-97a8-621a996611dd_mr",
				ScrapeInterval: config.Duration(s.MR),
				ScrapeTimeout:  scrapeTimeout(s.MR),
				MetricsPath:    "/metrics",
				HTTPClientConfig: config.HTTPClientConfig{
					BasicAuth: &config.BasicAuth{
						Username: "pmm",
						Password: "75bb30d3-ef4a-4147-97a8-621a996611dd",
					},
				},
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.Group{{
						Targets: []string{"4.5.6.7:12345"},
						Labels: map[string]string{
							"_some_agent_label":   "baz",
							"_some_node_label":    "foo",
							"_some_service_label": "bar",
							"agent_id":            "75bb30d3-ef4a-4147-97a8-621a996611dd",
							"agent_type":          "mysqld_exporter",
							"instance":            "75bb30d3-ef4a-4147-97a8-621a996611dd",
							"node_id":             "cc663f36-18ca-40a1-aea9-c6310bb4738d",
							"node_name":           "node_name",
							"service_id":          "014647c3-b2f5-44eb-94f4-d943260a968c",
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
				JobName:        "mysqld_exporter75bb30d3-ef4a-4147-97a8-621a996611dd_lr",
				ScrapeInterval: config.Duration(s.LR),
				ScrapeTimeout:  scrapeTimeout(s.LR),
				MetricsPath:    "/metrics",
				HTTPClientConfig: config.HTTPClientConfig{
					BasicAuth: &config.BasicAuth{
						Username: "pmm",
						Password: "75bb30d3-ef4a-4147-97a8-621a996611dd",
					},
				},
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.Group{{
						Targets: []string{"4.5.6.7:12345"},
						Labels: map[string]string{
							"_some_agent_label":   "baz",
							"_some_node_label":    "foo",
							"_some_service_label": "bar",
							"agent_id":            "75bb30d3-ef4a-4147-97a8-621a996611dd",
							"agent_type":          "mysqld_exporter",
							"instance":            "75bb30d3-ef4a-4147-97a8-621a996611dd",
							"node_id":             "cc663f36-18ca-40a1-aea9-c6310bb4738d",
							"node_name":           "node_name",
							"service_id":          "014647c3-b2f5-44eb-94f4-d943260a968c",
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
					"plugins",
				}},
			}}

			actual, err := scrapeConfigsForMySQLdExporter(&scrapeConfigParams{
				host:              "4.5.6.7",
				node:              node,
				service:           service,
				agent:             agent,
				metricsResolution: s,
			})
			require.NoError(t, err)
			require.Len(t, actual, len(expected))
			for i := 0; i < len(expected); i++ {
				assertScrapeConfigsEqual(t, expected[i], actual[i])
			}
		})

		t.Run("DisabledCollectors", func(t *testing.T) {
			node := &models.Node{
				NodeID:       "cc663f36-18ca-40a1-aea9-c6310bb4738d",
				NodeName:     "node_name",
				Address:      "1.2.3.4",
				CustomLabels: []byte(`{"_some_node_label": "foo"}`),
			}
			service := &models.Service{
				ServiceID:    "014647c3-b2f5-44eb-94f4-d943260a968c",
				NodeID:       "cc663f36-18ca-40a1-aea9-c6310bb4738d",
				Address:      pointer.ToString("5.6.7.8"),
				CustomLabels: []byte(`{"_some_service_label": "bar"}`),
			}
			agent := &models.Agent{
				AgentID:      "75bb30d3-ef4a-4147-97a8-621a996611dd",
				AgentType:    models.MySQLdExporterType,
				CustomLabels: []byte(`{"_some_agent_label": "baz"}`),
				ListenPort:   pointer.ToUint16(12345),
				ExporterOptions: &models.ExporterOptions{
					DisabledCollectors: []string{"global_status", "info_schema.innodb_cmp", "info_schema.query_response_time", "perf_schema.eventsstatements", "heartbeat"},
				},
			}

			expected := []*config.ScrapeConfig{{
				JobName:        "mysqld_exporter75bb30d3-ef4a-4147-97a8-621a996611dd_hr",
				ScrapeInterval: config.Duration(s.HR),
				ScrapeTimeout:  scrapeTimeout(s.HR),
				MetricsPath:    "/metrics",
				HTTPClientConfig: config.HTTPClientConfig{
					BasicAuth: &config.BasicAuth{
						Username: "pmm",
						Password: "75bb30d3-ef4a-4147-97a8-621a996611dd",
					},
				},
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.Group{{
						Targets: []string{"4.5.6.7:12345"},
						Labels: map[string]string{
							"_some_agent_label":   "baz",
							"_some_node_label":    "foo",
							"_some_service_label": "bar",
							"agent_id":            "75bb30d3-ef4a-4147-97a8-621a996611dd",
							"agent_type":          "mysqld_exporter",
							"instance":            "75bb30d3-ef4a-4147-97a8-621a996611dd",
							"node_id":             "cc663f36-18ca-40a1-aea9-c6310bb4738d",
							"node_name":           "node_name",
							"service_id":          "014647c3-b2f5-44eb-94f4-d943260a968c",
						},
					}},
				},
				Params: url.Values{"collect[]": []string{
					"custom_query.hr",
					"info_schema.innodb_metrics",
					"standard.go",
					"standard.process",
				}},
			}, {
				JobName:        "mysqld_exporter75bb30d3-ef4a-4147-97a8-621a996611dd_mr",
				ScrapeInterval: config.Duration(s.MR),
				ScrapeTimeout:  scrapeTimeout(s.MR),
				MetricsPath:    "/metrics",
				HTTPClientConfig: config.HTTPClientConfig{
					BasicAuth: &config.BasicAuth{
						Username: "pmm",
						Password: "75bb30d3-ef4a-4147-97a8-621a996611dd",
					},
				},
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.Group{{
						Targets: []string{"4.5.6.7:12345"},
						Labels: map[string]string{
							"_some_agent_label":   "baz",
							"_some_node_label":    "foo",
							"_some_service_label": "bar",
							"agent_id":            "75bb30d3-ef4a-4147-97a8-621a996611dd",
							"agent_type":          "mysqld_exporter",
							"instance":            "75bb30d3-ef4a-4147-97a8-621a996611dd",
							"node_id":             "cc663f36-18ca-40a1-aea9-c6310bb4738d",
							"node_name":           "node_name",
							"service_id":          "014647c3-b2f5-44eb-94f4-d943260a968c",
						},
					}},
				},
				Params: url.Values{"collect[]": []string{
					"custom_query.mr",
					"engine_innodb_status",
					"info_schema.innodb_cmpmem",
					"info_schema.processlist",
					"perf_schema.eventswaits",
					"perf_schema.file_events",
					"perf_schema.tablelocks",
					"slave_status",
				}},
			}, {
				JobName:        "mysqld_exporter75bb30d3-ef4a-4147-97a8-621a996611dd_lr",
				ScrapeInterval: config.Duration(s.LR),
				ScrapeTimeout:  scrapeTimeout(s.LR),
				MetricsPath:    "/metrics",
				HTTPClientConfig: config.HTTPClientConfig{
					BasicAuth: &config.BasicAuth{
						Username: "pmm",
						Password: "75bb30d3-ef4a-4147-97a8-621a996611dd",
					},
				},
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.Group{{
						Targets: []string{"4.5.6.7:12345"},
						Labels: map[string]string{
							"_some_agent_label":   "baz",
							"_some_node_label":    "foo",
							"_some_service_label": "bar",
							"agent_id":            "75bb30d3-ef4a-4147-97a8-621a996611dd",
							"agent_type":          "mysqld_exporter",
							"instance":            "75bb30d3-ef4a-4147-97a8-621a996611dd",
							"node_id":             "cc663f36-18ca-40a1-aea9-c6310bb4738d",
							"node_name":           "node_name",
							"service_id":          "014647c3-b2f5-44eb-94f4-d943260a968c",
						},
					}},
				},
				Params: url.Values{"collect[]": []string{
					"auto_increment.columns",
					"binlog_size",
					"custom_query.lr",
					"engine_tokudb_status",
					"global_variables",
					"info_schema.clientstats",
					"info_schema.innodb_tablespaces",
					"info_schema.tables",
					"info_schema.tablestats",
					"info_schema.userstats",
					"perf_schema.file_instances",
					"perf_schema.indexiowaits",
					"perf_schema.tableiowaits",
					"plugins",
				}},
			}}

			actual, err := scrapeConfigsForMySQLdExporter(&scrapeConfigParams{
				host:              "4.5.6.7",
				node:              node,
				service:           service,
				agent:             agent,
				metricsResolution: s,
			})
			require.NoError(t, err)
			require.Len(t, actual, len(expected))
			for i := 0; i < len(expected); i++ {
				assertScrapeConfigsEqual(t, expected[i], actual[i])
			}
		})

		t.Run("ManyTables", func(t *testing.T) {
			node := &models.Node{
				NodeID:   "cc663f36-18ca-40a1-aea9-c6310bb4738d",
				NodeName: "node_name",
				Address:  "1.2.3.4",
			}
			service := &models.Service{
				ServiceID: "014647c3-b2f5-44eb-94f4-d943260a968c",
				NodeID:    "cc663f36-18ca-40a1-aea9-c6310bb4738d",
				Address:   pointer.ToString("5.6.7.8"),
			}
			agent := &models.Agent{
				AgentID:    "75bb30d3-ef4a-4147-97a8-621a996611dd",
				AgentType:  models.MySQLdExporterType,
				ListenPort: pointer.ToUint16(12345),
				MySQLOptions: &models.MySQLOptions{
					TableCount:                     pointer.ToInt32(100500),
					TableCountTablestatsGroupLimit: 1000,
				},
			}

			expected := []*config.ScrapeConfig{{
				JobName:        "mysqld_exporter75bb30d3-ef4a-4147-97a8-621a996611dd_hr",
				ScrapeInterval: config.Duration(s.HR),
				ScrapeTimeout:  scrapeTimeout(s.HR),
				MetricsPath:    "/metrics",
				HTTPClientConfig: config.HTTPClientConfig{
					BasicAuth: &config.BasicAuth{
						Username: "pmm",
						Password: "75bb30d3-ef4a-4147-97a8-621a996611dd",
					},
				},
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.Group{{
						Targets: []string{"4.5.6.7:12345"},
						Labels: map[string]string{
							"agent_id":   "75bb30d3-ef4a-4147-97a8-621a996611dd",
							"agent_type": "mysqld_exporter",
							"instance":   "75bb30d3-ef4a-4147-97a8-621a996611dd",
							"node_id":    "cc663f36-18ca-40a1-aea9-c6310bb4738d",
							"node_name":  "node_name",
							"service_id": "014647c3-b2f5-44eb-94f4-d943260a968c",
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
				JobName:        "mysqld_exporter75bb30d3-ef4a-4147-97a8-621a996611dd_mr",
				ScrapeInterval: config.Duration(s.MR),
				ScrapeTimeout:  scrapeTimeout(s.MR),
				MetricsPath:    "/metrics",
				HTTPClientConfig: config.HTTPClientConfig{
					BasicAuth: &config.BasicAuth{
						Username: "pmm",
						Password: "75bb30d3-ef4a-4147-97a8-621a996611dd",
					},
				},
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.Group{{
						Targets: []string{"4.5.6.7:12345"},
						Labels: map[string]string{
							"agent_id":   "75bb30d3-ef4a-4147-97a8-621a996611dd",
							"agent_type": "mysqld_exporter",
							"instance":   "75bb30d3-ef4a-4147-97a8-621a996611dd",
							"node_id":    "cc663f36-18ca-40a1-aea9-c6310bb4738d",
							"node_name":  "node_name",
							"service_id": "014647c3-b2f5-44eb-94f4-d943260a968c",
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
				JobName:        "mysqld_exporter75bb30d3-ef4a-4147-97a8-621a996611dd_lr",
				ScrapeInterval: config.Duration(s.LR),
				ScrapeTimeout:  scrapeTimeout(s.LR),
				MetricsPath:    "/metrics",
				HTTPClientConfig: config.HTTPClientConfig{
					BasicAuth: &config.BasicAuth{
						Username: "pmm",
						Password: "75bb30d3-ef4a-4147-97a8-621a996611dd",
					},
				},
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.Group{{
						Targets: []string{"4.5.6.7:12345"},
						Labels: map[string]string{
							"agent_id":   "75bb30d3-ef4a-4147-97a8-621a996611dd",
							"agent_type": "mysqld_exporter",
							"instance":   "75bb30d3-ef4a-4147-97a8-621a996611dd",
							"node_id":    "cc663f36-18ca-40a1-aea9-c6310bb4738d",
							"node_name":  "node_name",
							"service_id": "014647c3-b2f5-44eb-94f4-d943260a968c",
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
					"info_schema.userstats",
					"perf_schema.eventsstatements",
					"perf_schema.file_instances",
					"plugins",
				}},
			}}

			actual, err := scrapeConfigsForMySQLdExporter(&scrapeConfigParams{
				host:              "4.5.6.7",
				node:              node,
				service:           service,
				agent:             agent,
				metricsResolution: s,
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

			_, err := scrapeConfigsForMySQLdExporter(&scrapeConfigParams{
				host:              "4.5.6.7",
				node:              node,
				service:           service,
				agent:             agent,
				metricsResolution: s,
			})
			require.EqualError(t, err, "failed to decode custom labels: unexpected end of JSON input")
		})
	})

	t.Run("scrapeConfigsForMongoDBExporter", func(t *testing.T) {
		t.Run("Normal", func(t *testing.T) {
			node := &models.Node{
				NodeID:       "cc663f36-18ca-40a1-aea9-c6310bb4738d",
				NodeName:     "node_name",
				Address:      "1.2.3.4",
				CustomLabels: []byte(`{"_some_node_label": "foo"}`),
			}
			service := &models.Service{
				ServiceID:    "014647c3-b2f5-44eb-94f4-d943260a968c",
				NodeID:       "cc663f36-18ca-40a1-aea9-c6310bb4738d",
				Address:      pointer.ToString("5.6.7.8"),
				CustomLabels: []byte(`{"_some_service_label": "bar"}`),
			}
			agent := &models.Agent{
				AgentID:        "75bb30d3-ef4a-4147-97a8-621a996611dd",
				AgentType:      models.MongoDBExporterType,
				CustomLabels:   []byte(`{"_some_agent_label": "baz"}`),
				ListenPort:     pointer.ToUint16(12345),
				MongoDBOptions: &models.MongoDBOptions{EnableAllCollectors: true},
			}

			expected := []*config.ScrapeConfig{
				{
					JobName:        "mongodb_exporter75bb30d3-ef4a-4147-97a8-621a996611dd_hr",
					ScrapeInterval: config.Duration(s.HR),
					ScrapeTimeout:  scrapeTimeout(s.HR),
					MetricsPath:    "/metrics",
					Params: map[string][]string{
						"collect[]": {"diagnosticdata", "replicasetstatus", "topmetrics"},
					},
					HTTPClientConfig: config.HTTPClientConfig{
						BasicAuth: &config.BasicAuth{
							Username: "pmm",
							Password: "75bb30d3-ef4a-4147-97a8-621a996611dd",
						},
					},
					ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
						StaticConfigs: []*config.Group{{
							Targets: []string{"4.5.6.7:12345"},
							Labels: map[string]string{
								"_some_agent_label":   "baz",
								"_some_node_label":    "foo",
								"_some_service_label": "bar",
								"agent_id":            "75bb30d3-ef4a-4147-97a8-621a996611dd",
								"agent_type":          "mongodb_exporter",
								"instance":            "75bb30d3-ef4a-4147-97a8-621a996611dd",
								"node_id":             "cc663f36-18ca-40a1-aea9-c6310bb4738d",
								"node_name":           "node_name",
								"service_id":          "014647c3-b2f5-44eb-94f4-d943260a968c",
							},
						}},
					},
				}, {
					JobName:        "mongodb_exporter75bb30d3-ef4a-4147-97a8-621a996611dd_lr",
					ScrapeInterval: config.Duration(s.LR),
					ScrapeTimeout:  scrapeTimeout(s.LR),
					MetricsPath:    "/metrics",
					Params: map[string][]string{
						"collect[]": {"collstats", "currentopmetrics", "dbstats", "indexstats", "shards"},
					},
					HTTPClientConfig: config.HTTPClientConfig{
						BasicAuth: &config.BasicAuth{
							Username: "pmm",
							Password: "75bb30d3-ef4a-4147-97a8-621a996611dd",
						},
					},
					ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
						StaticConfigs: []*config.Group{{
							Targets: []string{"4.5.6.7:12345"},
							Labels: map[string]string{
								"_some_agent_label":   "baz",
								"_some_node_label":    "foo",
								"_some_service_label": "bar",
								"agent_id":            "75bb30d3-ef4a-4147-97a8-621a996611dd",
								"agent_type":          "mongodb_exporter",
								"instance":            "75bb30d3-ef4a-4147-97a8-621a996611dd",
								"node_id":             "cc663f36-18ca-40a1-aea9-c6310bb4738d",
								"node_name":           "node_name",
								"service_id":          "014647c3-b2f5-44eb-94f4-d943260a968c",
							},
						}},
					},
				},
			}

			actual, err := scrapeConfigsForMongoDBExporter(&scrapeConfigParams{
				host:              "4.5.6.7",
				node:              node,
				service:           service,
				agent:             agent,
				pmmAgentVersion:   version.MustParse("2.42.0"),
				metricsResolution: s,
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

			_, err := scrapeConfigsForMongoDBExporter(&scrapeConfigParams{
				host:              "4.5.6.7",
				node:              node,
				service:           service,
				agent:             agent,
				pmmAgentVersion:   version.MustParse("2.26.0"),
				metricsResolution: s,
			})
			require.EqualError(t, err, "failed to decode custom labels: unexpected end of JSON input")
		})
	})

	t.Run("scrapeConfigsForPostgresExporter", func(t *testing.T) {
		t.Run("Normal", func(t *testing.T) {
			node := &models.Node{
				NodeID:       "cc663f36-18ca-40a1-aea9-c6310bb4738d",
				NodeName:     "node_name",
				Address:      "1.2.3.4",
				CustomLabels: []byte(`{"_some_node_label": "foo"}`),
			}
			service := &models.Service{
				ServiceID:    "014647c3-b2f5-44eb-94f4-d943260a968c",
				NodeID:       "cc663f36-18ca-40a1-aea9-c6310bb4738d",
				Address:      pointer.ToString("5.6.7.8"),
				CustomLabels: []byte(`{"_some_service_label": "bar"}`),
			}
			agent := &models.Agent{
				AgentID:      "75bb30d3-ef4a-4147-97a8-621a996611dd",
				AgentType:    models.PostgresExporterType,
				CustomLabels: []byte(`{"_some_agent_label": "baz"}`),
				ListenPort:   pointer.ToUint16(12345),
				ExporterOptions: &models.ExporterOptions{
					DisabledCollectors: []string{"standard.process", "custom_query.lr"},
				},
			}

			expected := []*config.ScrapeConfig{{
				JobName:        "postgres_exporter75bb30d3-ef4a-4147-97a8-621a996611dd_hr",
				ScrapeInterval: config.Duration(s.HR),
				ScrapeTimeout:  scrapeTimeout(s.HR),
				MetricsPath:    "/metrics",
				HTTPClientConfig: config.HTTPClientConfig{
					BasicAuth: &config.BasicAuth{
						Username: "pmm",
						Password: "75bb30d3-ef4a-4147-97a8-621a996611dd",
					},
				},
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.Group{{
						Targets: []string{"4.5.6.7:12345"},
						Labels: map[string]string{
							"_some_agent_label":   "baz",
							"_some_node_label":    "foo",
							"_some_service_label": "bar",
							"agent_id":            "75bb30d3-ef4a-4147-97a8-621a996611dd",
							"agent_type":          "postgres_exporter",
							"instance":            "75bb30d3-ef4a-4147-97a8-621a996611dd",
							"node_id":             "cc663f36-18ca-40a1-aea9-c6310bb4738d",
							"node_name":           "node_name",
							"service_id":          "014647c3-b2f5-44eb-94f4-d943260a968c",
						},
					}},
				},
				Params: url.Values{"collect[]": []string{
					"custom_query.hr",
					"exporter",
					"postgres",
					"standard.go",
				}},
			}, {
				JobName:        "postgres_exporter75bb30d3-ef4a-4147-97a8-621a996611dd_mr",
				ScrapeInterval: config.Duration(s.MR),
				ScrapeTimeout:  scrapeTimeout(s.MR),
				MetricsPath:    "/metrics",
				HTTPClientConfig: config.HTTPClientConfig{
					BasicAuth: &config.BasicAuth{
						Username: "pmm",
						Password: "75bb30d3-ef4a-4147-97a8-621a996611dd",
					},
				},
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.Group{{
						Targets: []string{"4.5.6.7:12345"},
						Labels: map[string]string{
							"_some_agent_label":   "baz",
							"_some_node_label":    "foo",
							"_some_service_label": "bar",
							"agent_id":            "75bb30d3-ef4a-4147-97a8-621a996611dd",
							"agent_type":          "postgres_exporter",
							"instance":            "75bb30d3-ef4a-4147-97a8-621a996611dd",
							"node_id":             "cc663f36-18ca-40a1-aea9-c6310bb4738d",
							"node_name":           "node_name",
							"service_id":          "014647c3-b2f5-44eb-94f4-d943260a968c",
						},
					}},
				},
				Params: url.Values{"collect[]": []string{
					"custom_query.mr",
				}},
			}, {
				JobName:        "postgres_exporter75bb30d3-ef4a-4147-97a8-621a996611dd_lr",
				ScrapeInterval: config.Duration(s.LR),
				ScrapeTimeout:  scrapeTimeout(s.LR),
				MetricsPath:    "/metrics",
				HTTPClientConfig: config.HTTPClientConfig{
					BasicAuth: &config.BasicAuth{
						Username: "pmm",
						Password: "75bb30d3-ef4a-4147-97a8-621a996611dd",
					},
				},
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.Group{{
						Targets: []string{"4.5.6.7:12345"},
						Labels: map[string]string{
							"_some_agent_label":   "baz",
							"_some_node_label":    "foo",
							"_some_service_label": "bar",
							"agent_id":            "75bb30d3-ef4a-4147-97a8-621a996611dd",
							"agent_type":          "postgres_exporter",
							"instance":            "75bb30d3-ef4a-4147-97a8-621a996611dd",
							"node_id":             "cc663f36-18ca-40a1-aea9-c6310bb4738d",
							"node_name":           "node_name",
							"service_id":          "014647c3-b2f5-44eb-94f4-d943260a968c",
						},
					}},
				},
				Params: nil,
			}}

			actual, err := scrapeConfigsForPostgresExporter(&scrapeConfigParams{
				host:              "4.5.6.7",
				node:              node,
				service:           service,
				agent:             agent,
				metricsResolution: s,
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

			_, err := scrapeConfigsForPostgresExporter(&scrapeConfigParams{
				host:              "4.5.6.7",
				node:              node,
				service:           service,
				agent:             agent,
				metricsResolution: s,
			})
			require.EqualError(t, err, "failed to decode custom labels: unexpected end of JSON input")
		})
	})

	t.Run("scrapeConfigsForProxySQLExporter", func(t *testing.T) {
		t.Run("Normal", func(t *testing.T) {
			node := &models.Node{
				NodeID:       "cc663f36-18ca-40a1-aea9-c6310bb4738d",
				NodeName:     "node_name",
				Address:      "1.2.3.4",
				CustomLabels: []byte(`{"_some_node_label": "foo"}`),
			}
			service := &models.Service{
				ServiceID:    "014647c3-b2f5-44eb-94f4-d943260a968c",
				NodeID:       "cc663f36-18ca-40a1-aea9-c6310bb4738d",
				Address:      pointer.ToString("5.6.7.8"),
				CustomLabels: []byte(`{"_some_service_label": "bar"}`),
			}
			agent := &models.Agent{
				AgentID:      "75bb30d3-ef4a-4147-97a8-621a996611dd",
				AgentType:    models.ProxySQLExporterType,
				CustomLabels: []byte(`{"_some_agent_label": "baz"}`),
				ListenPort:   pointer.ToUint16(12345),
			}

			expected := []*config.ScrapeConfig{{
				JobName:        "proxysql_exporter75bb30d3-ef4a-4147-97a8-621a996611dd_hr",
				ScrapeInterval: config.Duration(s.HR),
				ScrapeTimeout:  scrapeTimeout(s.HR),
				MetricsPath:    "/metrics",
				HTTPClientConfig: config.HTTPClientConfig{
					BasicAuth: &config.BasicAuth{
						Username: "pmm",
						Password: "75bb30d3-ef4a-4147-97a8-621a996611dd",
					},
				},
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.Group{{
						Targets: []string{"4.5.6.7:12345"},
						Labels: map[string]string{
							"_some_agent_label":   "baz",
							"_some_node_label":    "foo",
							"_some_service_label": "bar",
							"agent_id":            "75bb30d3-ef4a-4147-97a8-621a996611dd",
							"agent_type":          "proxysql_exporter",
							"instance":            "75bb30d3-ef4a-4147-97a8-621a996611dd",
							"node_id":             "cc663f36-18ca-40a1-aea9-c6310bb4738d",
							"node_name":           "node_name",
							"service_id":          "014647c3-b2f5-44eb-94f4-d943260a968c",
						},
					}},
				},
			}}

			actual, err := scrapeConfigsForProxySQLExporter(&scrapeConfigParams{
				host:              "4.5.6.7",
				node:              node,
				service:           service,
				agent:             agent,
				metricsResolution: s,
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

			_, err := scrapeConfigsForProxySQLExporter(&scrapeConfigParams{
				host:              "4.5.6.7",
				node:              node,
				service:           service,
				agent:             agent,
				metricsResolution: s,
			})
			require.EqualError(t, err, "failed to decode custom labels: unexpected end of JSON input")
		})
	})

	t.Run("scrapeConfigsForRDSExporter", func(t *testing.T) {
		t.Run("Normal", func(t *testing.T) {
			params := []*scrapeConfigParams{
				// two RDS configs on the same host/port combination: single pmm-agent, single rds_exporter process
				{
					host:              "1.1.1.1",
					agent:             &models.Agent{ListenPort: pointer.ToUint16(12345)},
					metricsResolution: s,
				},
				{
					host:              "1.1.1.1",
					agent:             &models.Agent{ListenPort: pointer.ToUint16(12345)},
					metricsResolution: s,
				},

				// two RDS configs on the same host, different ports: two pmm-agents, two rds_exporter processes
				{
					host:              "2.2.2.2",
					agent:             &models.Agent{ListenPort: pointer.ToUint16(12345)},
					metricsResolution: s,
				},
				{
					host:              "2.2.2.2",
					agent:             &models.Agent{ListenPort: pointer.ToUint16(12346)},
					metricsResolution: s,
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

			actual := scrapeConfigsForRDSExporter(params)
			require.Len(t, actual, len(expected))
			for i := 0; i < len(expected); i++ {
				assertScrapeConfigsEqual(t, expected[i], actual[i])
			}
		})
	})

	t.Run("scrapeConfigsForExternalExporter", func(t *testing.T) {
		node := &models.Node{
			NodeID:       "cc663f36-18ca-40a1-aea9-c6310bb4738d",
			NodeName:     "node_name",
			Address:      "1.2.3.4",
			CustomLabels: []byte(`{"_some_node_label": "foo"}`),
		}
		service := &models.Service{
			ServiceID:     "014647c3-b2f5-44eb-94f4-d943260a968c",
			NodeID:        "cc663f36-18ca-40a1-aea9-c6310bb4738d",
			Address:       pointer.ToString("5.6.7.8"),
			CustomLabels:  []byte(`{"_some_service_label": "bar"}`),
			ExternalGroup: "rabbitmq",
		}
		t.Run("Normal", func(t *testing.T) {
			agent := &models.Agent{
				AgentID:      "75bb30d3-ef4a-4147-97a8-621a996611dd",
				AgentType:    models.ExternalExporterType,
				CustomLabels: []byte(`{"_some_agent_label": "baz"}`),
				ListenPort:   pointer.ToUint16(12345),
			}

			expected := []*config.ScrapeConfig{{
				JobName:        "external-exporter75bb30d3-ef4a-4147-97a8-621a996611dd_mr",
				ScrapeInterval: config.Duration(s.HR),
				ScrapeTimeout:  scrapeTimeout(s.HR),
				ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
					StaticConfigs: []*config.Group{{
						Targets: []string{"4.5.6.7:12345"},
						Labels: map[string]string{
							"_some_agent_label":   "baz",
							"_some_node_label":    "foo",
							"_some_service_label": "bar",
							"agent_id":            "75bb30d3-ef4a-4147-97a8-621a996611dd",
							"agent_type":          "external-exporter",
							"external_group":      "rabbitmq",
							"instance":            "75bb30d3-ef4a-4147-97a8-621a996611dd",
							"node_id":             "cc663f36-18ca-40a1-aea9-c6310bb4738d",
							"node_name":           "node_name",
							"service_id":          "014647c3-b2f5-44eb-94f4-d943260a968c",
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
				AgentID:      "75bb30d3-ef4a-4147-97a8-621a996611dd",
				AgentType:    models.ExternalExporterType,
				CustomLabels: []byte(`{"_some_agent_label": "baz"}`),
				Username:     pointer.ToString("username"),
				Password:     pointer.ToString("password"),
				ListenPort:   pointer.ToUint16(12345),
				ExporterOptions: &models.ExporterOptions{
					MetricsPath:   pointer.ToString("/some-metric-path"),
					MetricsScheme: pointer.ToString("https"),
				},
			}

			expected := []*config.ScrapeConfig{{
				JobName:        "external-exporter75bb30d3-ef4a-4147-97a8-621a996611dd_mr",
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
							"agent_id":            "75bb30d3-ef4a-4147-97a8-621a996611dd",
							"agent_type":          "external-exporter",
							"external_group":      "rabbitmq",
							"instance":            "75bb30d3-ef4a-4147-97a8-621a996611dd",
							"node_id":             "cc663f36-18ca-40a1-aea9-c6310bb4738d",
							"node_name":           "node_name",
							"service_id":          "014647c3-b2f5-44eb-94f4-d943260a968c",
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

			_, err := scrapeConfigsForMongoDBExporter(&scrapeConfigParams{
				host:              "4.5.6.7",
				node:              node,
				service:           service,
				agent:             agent,
				pmmAgentVersion:   version.MustParse("2.26.0"),
				metricsResolution: s,
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
