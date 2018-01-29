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
	t.Helper()

	e, err := yaml.Marshal(expected)
	require.NoError(t, err)
	a, err := yaml.Marshal(actual)
	require.NoError(t, err)
	assert.Equal(t, strings.Split(string(e), "\n"), strings.Split(string(a), "\n"), "expected:\n%s\nactual:\n%s", e, a)
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

func TestConfigUpdaterAddSetRemoveScrapeConfig(t *testing.T) {
	configUpdater := getConfigUpdater()

	add := &ScrapeConfig{
		JobName: "AddSetRemoveScrapeConfig",
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
		JobName: "AddSetRemoveScrapeConfig",
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
		JobName: "AddSetRemoveScrapeConfig",
		ServiceDiscoveryConfig: internal.ServiceDiscoveryConfig{
			StaticConfigs: []*internal.TargetGroup{{
				Targets: []model.LabelSet{{"__address__": "5.6.7.8:12345"}},
			}},
		},
	}}, configUpdater.fileData)

	err = configUpdater.addScrapeConfig(add)
	tests.AssertGRPCError(t, status.New(codes.AlreadyExists, `scrape config with job name "AddSetRemoveScrapeConfig" already exist`), err)

	add.JobName = "prometheus"
	err = configUpdater.addScrapeConfig(add)
	tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, `scrape config with job name "prometheus" is built-in`), err)

	set := add
	set.JobName = "AddSetRemoveScrapeConfig"
	set.StaticConfigs = []StaticConfig{{
		Targets: []string{"5.6.7.8:12345"},
		Labels: []LabelPair{{
			Name:  "instance",
			Value: "test_host",
		}},
	}}
	err = configUpdater.setScrapeConfig(set)
	assert.NoError(t, err)

	assertYAMLEqual(t, []ScrapeConfig{{
		JobName: "postgresql",
		StaticConfigs: []StaticConfig{{
			Targets: []string{"1.2.3.4:12345"},
		}},
	}, {
		JobName: "AddSetRemoveScrapeConfig",
		StaticConfigs: []StaticConfig{{
			Targets: []string{"5.6.7.8:12345"},
			Labels: []LabelPair{{
				Name:  "instance",
				Value: "test_host",
			}},
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
		JobName: "AddSetRemoveScrapeConfig",
		ServiceDiscoveryConfig: internal.ServiceDiscoveryConfig{
			StaticConfigs: []*internal.TargetGroup{{
				Targets: []model.LabelSet{{"__address__": "5.6.7.8:12345"}},
				Labels:  model.LabelSet{"instance": "test_host"},
			}},
		},
	}}, configUpdater.fileData)

	set.JobName = "no_such_config"
	err = configUpdater.setScrapeConfig(set)
	tests.AssertGRPCError(t, status.New(codes.NotFound, `scrape config with job name "no_such_config" not found`), err)

	set.JobName = "prometheus"
	err = configUpdater.setScrapeConfig(set)
	tests.AssertGRPCError(t, status.New(codes.NotFound, `scrape config with job name "prometheus" not found`), err)

	err = configUpdater.removeScrapeConfig("AddSetRemoveScrapeConfig")
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

	err = configUpdater.removeScrapeConfig("AddSetRemoveScrapeConfig")
	tests.AssertGRPCError(t, status.New(codes.NotFound, `scrape config with job name "AddSetRemoveScrapeConfig" not found`), err)

	err = configUpdater.removeScrapeConfig("prometheus")
	tests.AssertGRPCError(t, status.New(codes.NotFound, `scrape config with job name "prometheus" not found`), err)
}
