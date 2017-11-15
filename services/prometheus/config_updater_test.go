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
	"strings"
	"testing"

	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/yaml.v2"

	"github.com/percona/pmm-managed/services/prometheus/internal"
	"github.com/percona/pmm-managed/utils/tests"
)

func assertYAMLEqual(t *testing.T, expected interface{}, actual interface{}) {
	e, err := yaml.Marshal(expected)
	require.NoError(t, err)
	a, err := yaml.Marshal(actual)
	require.NoError(t, err)
	assert.Equal(t, strings.Split(string(e), "\n"), strings.Split(string(a), "\n"))
}

func getConfigUpdater() *configUpdater {
	consulData := []ScrapeConfig{{
		JobName: "postgresql",
		StaticConfigs: []StaticConfig{{
			Targets: []string{"1.2.3.4:12345"},
		}},
	}}
	fileData := []*internal.ScrapeConfig{{
		JobName: "prometheus",
		ServiceDiscoveryConfig: internal.ServiceDiscoveryConfig{
			StaticConfigs: []*internal.TargetGroup{{
				Targets: []model.LabelSet{{"__address__": "127.0.0.1:9090"}},
			}},
		},
	}, {
		JobName: "postgresql",
		ServiceDiscoveryConfig: internal.ServiceDiscoveryConfig{
			StaticConfigs: []*internal.TargetGroup{{
				Targets: []model.LabelSet{{"__address__": "1.2.3.4:12345"}},
			}},
		},
	}}
	return &configUpdater{consulData, fileData}
}

func TestConfigUpdaterTestConfigUpdaterAddRemoveScrapeConfig(t *testing.T) {
	configUpdater := getConfigUpdater()

	add := &ScrapeConfig{
		JobName: "TestConfigUpdaterAddRemoveScrapeConfig",
		StaticConfigs: []StaticConfig{{
			Targets: []string{"5.6.7.8:12345"},
		}},
	}
	err := configUpdater.addScrapeConfig(add)
	assert.NoError(t, err)

	assertYAMLEqual(t, []ScrapeConfig{{
		JobName: "postgresql",
		StaticConfigs: []StaticConfig{{
			Targets: []string{"1.2.3.4:12345"},
		}},
	}, {
		JobName: "TestConfigUpdaterAddRemoveScrapeConfig",
		StaticConfigs: []StaticConfig{{
			Targets: []string{"5.6.7.8:12345"},
		}},
	}}, configUpdater.consulData)
	assertYAMLEqual(t, []*internal.ScrapeConfig{{
		JobName: "prometheus",
		ServiceDiscoveryConfig: internal.ServiceDiscoveryConfig{
			StaticConfigs: []*internal.TargetGroup{{
				Targets: []model.LabelSet{{"__address__": "127.0.0.1:9090"}},
			}},
		},
	}, {
		JobName: "postgresql",
		ServiceDiscoveryConfig: internal.ServiceDiscoveryConfig{
			StaticConfigs: []*internal.TargetGroup{{
				Targets: []model.LabelSet{{"__address__": "1.2.3.4:12345"}},
			}},
		},
	}, {
		JobName: "TestConfigUpdaterAddRemoveScrapeConfig",
		ServiceDiscoveryConfig: internal.ServiceDiscoveryConfig{
			StaticConfigs: []*internal.TargetGroup{{
				Targets: []model.LabelSet{{"__address__": "5.6.7.8:12345"}},
			}},
		},
	}}, configUpdater.fileData)

	err = configUpdater.addScrapeConfig(add)
	tests.AssertGRPCError(t, status.New(codes.AlreadyExists, `scrape config with job name "TestConfigUpdaterAddRemoveScrapeConfig" already exist`), err)

	add.JobName = "prometheus"
	err = configUpdater.addScrapeConfig(add)
	tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, `scrape config with job name "prometheus" is built-in`), err)

	err = configUpdater.removeScrapeConfig("TestConfigUpdaterAddRemoveScrapeConfig")
	assert.NoError(t, err)

	assertYAMLEqual(t, []ScrapeConfig{{
		JobName: "postgresql",
		StaticConfigs: []StaticConfig{{
			Targets: []string{"1.2.3.4:12345"},
		}},
	}}, configUpdater.consulData)
	assertYAMLEqual(t, []*internal.ScrapeConfig{{
		JobName: "prometheus",
		ServiceDiscoveryConfig: internal.ServiceDiscoveryConfig{
			StaticConfigs: []*internal.TargetGroup{{
				Targets: []model.LabelSet{{"__address__": "127.0.0.1:9090"}},
			}},
		},
	}, {
		JobName: "postgresql",
		ServiceDiscoveryConfig: internal.ServiceDiscoveryConfig{
			StaticConfigs: []*internal.TargetGroup{{
				Targets: []model.LabelSet{{"__address__": "1.2.3.4:12345"}},
			}},
		},
	}}, configUpdater.fileData)

	err = configUpdater.removeScrapeConfig("TestConfigUpdaterAddRemoveScrapeConfig")
	tests.AssertGRPCError(t, status.New(codes.NotFound, `scrape config with job name "TestConfigUpdaterAddRemoveScrapeConfig" not found`), err)

	err = configUpdater.removeScrapeConfig("prometheus")
	tests.AssertGRPCError(t, status.New(codes.NotFound, `scrape config with job name "prometheus" not found`), err)
}

func TestConfigUpdaterAddRemoveStaticTargets(t *testing.T) {
	configUpdater := getConfigUpdater()

	// add the same targets twice: no error, no duplicate
	for i := 0; i < 2; i++ {
		err := configUpdater.addStaticTargets("postgresql", []string{"5.6.7.8:12345", "1.2.3.4:12345"})
		assert.NoError(t, err)

		assertYAMLEqual(t, []ScrapeConfig{{
			JobName: "postgresql",
			StaticConfigs: []StaticConfig{{
				Targets: []string{"1.2.3.4:12345", "5.6.7.8:12345"},
			}},
		}}, configUpdater.consulData)
		assertYAMLEqual(t, []*internal.ScrapeConfig{{
			JobName: "prometheus",
			ServiceDiscoveryConfig: internal.ServiceDiscoveryConfig{
				StaticConfigs: []*internal.TargetGroup{{
					Targets: []model.LabelSet{{"__address__": "127.0.0.1:9090"}},
				}},
			},
		}, {
			JobName: "postgresql",
			ServiceDiscoveryConfig: internal.ServiceDiscoveryConfig{
				StaticConfigs: []*internal.TargetGroup{{
					Targets: []model.LabelSet{{"__address__": "1.2.3.4:12345"}, {"__address__": "5.6.7.8:12345"}},
				}},
			},
		}}, configUpdater.fileData)
	}

	err := configUpdater.addStaticTargets("prometheus", []string{"127.0.0.1:9090"})
	tests.AssertGRPCError(t, status.New(codes.NotFound, `scrape config with job name "prometheus" not found`), err)

	err = configUpdater.addStaticTargets("no_such_job", []string{"127.0.0.1:9090"})
	tests.AssertGRPCError(t, status.New(codes.NotFound, `scrape config with job name "no_such_job" not found`), err)

	// remove the same target twice: no error, no duplicate
	for i := 0; i < 2; i++ {
		err := configUpdater.removeStaticTargets("postgresql", []string{"1.2.3.4:12345"})
		assert.NoError(t, err)

		assertYAMLEqual(t, []ScrapeConfig{{
			JobName: "postgresql",
			StaticConfigs: []StaticConfig{{
				Targets: []string{"5.6.7.8:12345"},
			}},
		}}, configUpdater.consulData)
		assertYAMLEqual(t, []*internal.ScrapeConfig{{
			JobName: "prometheus",
			ServiceDiscoveryConfig: internal.ServiceDiscoveryConfig{
				StaticConfigs: []*internal.TargetGroup{{
					Targets: []model.LabelSet{{"__address__": "127.0.0.1:9090"}},
				}},
			},
		}, {
			JobName: "postgresql",
			ServiceDiscoveryConfig: internal.ServiceDiscoveryConfig{
				StaticConfigs: []*internal.TargetGroup{{
					Targets: []model.LabelSet{{"__address__": "5.6.7.8:12345"}},
				}},
			},
		}}, configUpdater.fileData)
	}

	err = configUpdater.removeStaticTargets("postgresql", []string{"5.6.7.8:12345"})
	assert.NoError(t, err)

	assertYAMLEqual(t, []ScrapeConfig{{
		JobName: "postgresql",
	}}, configUpdater.consulData)
	assertYAMLEqual(t, []*internal.ScrapeConfig{{
		JobName: "prometheus",
		ServiceDiscoveryConfig: internal.ServiceDiscoveryConfig{
			StaticConfigs: []*internal.TargetGroup{{
				Targets: []model.LabelSet{{"__address__": "127.0.0.1:9090"}},
			}},
		},
	}, {
		JobName: "postgresql",
	}}, configUpdater.fileData)

	err = configUpdater.removeStaticTargets("prometheus", []string{"127.0.0.1:9090"})
	tests.AssertGRPCError(t, status.New(codes.NotFound, `scrape config with job name "prometheus" not found`), err)

	err = configUpdater.removeStaticTargets("no_such_job", []string{"127.0.0.1:9090"})
	tests.AssertGRPCError(t, status.New(codes.NotFound, `scrape config with job name "no_such_job" not found`), err)
}
