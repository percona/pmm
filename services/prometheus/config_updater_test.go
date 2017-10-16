// pmm-managed
// Copyright (C) 2017 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package prometheus

import (
	"testing"

	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/yaml.v2"

	"github.com/percona/pmm-managed/services/prometheus/internal"
)

func assertYAMLEqual(t *testing.T, expected interface{}, actual interface{}) {
	e, err := yaml.Marshal(expected)
	require.NoError(t, err)
	a, err := yaml.Marshal(actual)
	require.NoError(t, err)
	assert.Equal(t, e, a)
}

func TestConfigUpdater(t *testing.T) {
	consulData := []ScrapeConfig{
		{
			JobName: "postgresql",
			StaticConfigs: []StaticConfig{{
				Targets: []string{"1.2.3.4:12345"},
			}},
		},
	}
	fileData := []*internal.ScrapeConfig{
		{
			JobName: "prometheus",
			ServiceDiscoveryConfig: internal.ServiceDiscoveryConfig{
				StaticConfigs: []*internal.TargetGroup{{
					Targets: []model.LabelSet{{"__address__": "127.0.0.1:9090"}},
				}},
			},
		},
		{
			JobName: "postgresql",
			ServiceDiscoveryConfig: internal.ServiceDiscoveryConfig{
				StaticConfigs: []*internal.TargetGroup{{
					Targets: []model.LabelSet{{"__address__": "1.2.3.4:12345"}},
				}},
			},
		},
	}
	configUpdater := &configUpdater{consulData, fileData}

	t.Run("AddRemoveScrapeConfig", func(t *testing.T) {
		add := &ScrapeConfig{
			JobName: "AddRemoveScrapeConfig",
			StaticConfigs: []StaticConfig{{
				Targets: []string{"5.6.7.8:12345"},
			}},
		}
		err := configUpdater.addScrapeConfig(add)
		assert.NoError(t, err)

		expectedC := []ScrapeConfig{
			{
				JobName: "postgresql",
				StaticConfigs: []StaticConfig{{
					Targets: []string{"1.2.3.4:12345"},
				}},
			},
			{
				JobName: "AddRemoveScrapeConfig",
				StaticConfigs: []StaticConfig{{
					Targets: []string{"5.6.7.8:12345"},
				}},
			},
		}
		assertYAMLEqual(t, expectedC, configUpdater.consulData)

		expectedF := []*internal.ScrapeConfig{
			{
				JobName: "prometheus",
				ServiceDiscoveryConfig: internal.ServiceDiscoveryConfig{
					StaticConfigs: []*internal.TargetGroup{{
						Targets: []model.LabelSet{{"__address__": "127.0.0.1:9090"}},
					}},
				},
			},
			{
				JobName: "postgresql",
				ServiceDiscoveryConfig: internal.ServiceDiscoveryConfig{
					StaticConfigs: []*internal.TargetGroup{{
						Targets: []model.LabelSet{{"__address__": "1.2.3.4:12345"}},
					}},
				},
			},
			{
				JobName: "AddRemoveScrapeConfig",
				ServiceDiscoveryConfig: internal.ServiceDiscoveryConfig{
					StaticConfigs: []*internal.TargetGroup{{
						Targets: []model.LabelSet{{"__address__": "5.6.7.8:12345"}},
					}},
				},
			},
		}
		assertYAMLEqual(t, expectedF, configUpdater.fileData)

		err = configUpdater.addScrapeConfig(add)
		assertGRPCError(t, status.New(codes.AlreadyExists, `scrape config with job name "AddRemoveScrapeConfig" already exist`), err)
	})
}
