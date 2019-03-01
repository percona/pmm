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
	config_util "github.com/percona/pmm-managed/services/prometheus/internal/common/config"
	"github.com/percona/pmm-managed/services/prometheus/internal/prometheus/config"
	sd_config "github.com/percona/pmm-managed/services/prometheus/internal/prometheus/discovery/config"
	"github.com/percona/pmm-managed/services/prometheus/internal/prometheus/discovery/targetgroup"
)

func TestScrapeConfig(t *testing.T) {
	t.Run("scrapeConfigsForMySQLdExporter", func(t *testing.T) {
		t.Run("Normal", func(t *testing.T) {
			node := &models.Node{
				NodeID:  "/node_id/cc663f36-18ca-40a1-aea9-c6310bb4738d",
				Address: pointer.ToString("1.2.3.4"),
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
				Username:     pointer.ToString("username"),
				Password:     pointer.ToString("password"),
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
				HTTPClientConfig: config_util.HTTPClientConfig{
					BasicAuth: &config_util.BasicAuth{
						Username: "username",
						Password: "password",
					},
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
				HTTPClientConfig: config_util.HTTPClientConfig{
					BasicAuth: &config_util.BasicAuth{
						Username: "username",
						Password: "password",
					},
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
				HTTPClientConfig: config_util.HTTPClientConfig{
					BasicAuth: &config_util.BasicAuth{
						Username: "username",
						Password: "password",
					},
				},
			}}
			actual, err := scrapeConfigsForMySQLdExporter(node, service, agent)
			require.NoError(t, err)
			require.Len(t, actual, len(expected))
			for i := 0; i < len(expected); i++ {
				if !assert.Equal(t, expected[i], actual[i]) {
					e, err := yaml.Marshal(expected[i])
					require.NoError(t, err)
					a, err := yaml.Marshal(actual[i])
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
		})

		t.Run("BadCustomLabels", func(t *testing.T) {
			node := &models.Node{}
			service := &models.Service{}
			agent := &models.Agent{
				CustomLabels: []byte("{"),
				ListenPort:   pointer.ToUint16(12345),
			}

			_, err := scrapeConfigsForMySQLdExporter(node, service, agent)
			require.EqualError(t, err, "failed to decode custom labels: unexpected end of JSON input")
		})
	})
}
