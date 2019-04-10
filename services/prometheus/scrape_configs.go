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
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"
	"github.com/prometheus/common/model"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/services/prometheus/internal/prometheus/config"
	sd_config "github.com/percona/pmm-managed/services/prometheus/internal/prometheus/discovery/config"
	"github.com/percona/pmm-managed/services/prometheus/internal/prometheus/discovery/targetgroup"
)

const addressLabel = model.LabelName(model.AddressLabel)

func scrapeConfigForPrometheus() *config.ScrapeConfig {
	return &config.ScrapeConfig{
		JobName:        "prometheus",
		ScrapeInterval: model.Duration(time.Second),
		ScrapeTimeout:  model.Duration(time.Second),
		MetricsPath:    "/prometheus/metrics",
		ServiceDiscoveryConfig: sd_config.ServiceDiscoveryConfig{
			StaticConfigs: []*targetgroup.Group{{
				Targets: []model.LabelSet{{addressLabel: "127.0.0.1:9090"}},
				Labels:  model.LabelSet{"instance": "pmm-server"},
			}},
		},
	}
}

func scrapeConfigForGrafana() *config.ScrapeConfig {
	return &config.ScrapeConfig{
		JobName:        "grafana",
		ScrapeInterval: model.Duration(5 * time.Second),
		ScrapeTimeout:  model.Duration(4 * time.Second),
		MetricsPath:    "/metrics",
		ServiceDiscoveryConfig: sd_config.ServiceDiscoveryConfig{
			StaticConfigs: []*targetgroup.Group{{
				Targets: []model.LabelSet{{addressLabel: "127.0.0.1:3000"}},
				Labels:  model.LabelSet{"instance": "pmm-server"},
			}},
		},
	}
}

func scrapeConfigForPMMManaged() *config.ScrapeConfig {
	return &config.ScrapeConfig{
		JobName:        "pmm-managed",
		ScrapeInterval: model.Duration(10 * time.Second),
		ScrapeTimeout:  model.Duration(5 * time.Second),
		MetricsPath:    "/debug/metrics",
		ServiceDiscoveryConfig: sd_config.ServiceDiscoveryConfig{
			StaticConfigs: []*targetgroup.Group{{
				Targets: []model.LabelSet{{addressLabel: "127.0.0.1:7773"}},
				Labels:  model.LabelSet{"instance": "pmm-server"},
			}},
		},
	}
}

func commonExporterLabelSet(node *models.Node, service *models.Service, agent *models.Agent) model.LabelSet {
	res := make(model.LabelSet)

	res[model.LabelName("node_id")] = model.LabelValue(node.NodeID)
	res[model.LabelName("node_type")] = model.LabelValue(node.NodeType)
	res[model.LabelName("node_name")] = model.LabelValue(node.NodeName)
	res[model.LabelName("machine_id")] = model.LabelValue(pointer.GetString(node.MachineID))
	res[model.LabelName("container_id")] = model.LabelValue(pointer.GetString(node.DockerContainerID))
	res[model.LabelName("container_name")] = model.LabelValue(pointer.GetString(node.DockerContainerName))
	res[model.LabelName("region")] = model.LabelValue(pointer.GetString(node.Region)) // TODO https://jira.percona.com/browse/PMM-3786
	// TODO node_model, az

	if service != nil {
		res[model.LabelName("service_id")] = model.LabelValue(service.ServiceID)
		res[model.LabelName("service_type")] = model.LabelValue(service.ServiceType)
		res[model.LabelName("service_name")] = model.LabelValue(service.ServiceName)
	}

	res[model.LabelName("instance")] = model.LabelValue(agent.AgentID)
	res[model.LabelName("agent_type")] = model.LabelValue(agent.AgentType)

	return res
}

func mergeLabels(labels model.LabelSet, node *models.Node, service *models.Service, agent *models.Agent) error {
	var nLabels, sLabels, aLabels map[string]string
	var err error
	if nLabels, err = node.GetCustomLabels(); err != nil {
		return err
	}
	if service != nil {
		if sLabels, err = service.GetCustomLabels(); err != nil {
			return err
		}
	}
	if aLabels, err = agent.GetCustomLabels(); err != nil {
		return err
	}

	for k, v := range nLabels {
		labels[model.LabelName(k)] = model.LabelValue(v)
	}
	for k, v := range sLabels {
		labels[model.LabelName(k)] = model.LabelValue(v)
	}
	for k, v := range aLabels {
		labels[model.LabelName(k)] = model.LabelValue(v)
	}

	var toDelete []model.LabelName
	for k, v := range labels {
		if v == "" {
			toDelete = append(toDelete, k)
		}
	}
	for _, k := range toDelete {
		delete(labels, k)
	}

	return errors.Wrap(labels.Validate(), "failed to merge labels")
}

func jobName(agent *models.Agent) string {
	return string(agent.AgentType) + strings.Replace(agent.AgentID, "/", "_", -1)
}

func scrapeConfigForNodeExporter(node *models.Node, agent *models.Agent) (*config.ScrapeConfig, error) {
	labels := commonExporterLabelSet(node, nil, agent)
	if err := mergeLabels(labels, node, nil, agent); err != nil {
		return nil, err
	}

	port := pointer.GetUint16(agent.ListenPort)
	if port == 0 {
		return nil, errors.New("listen port is not known")
	}
	hostport := net.JoinHostPort(pointer.GetString(node.Address), strconv.Itoa(int(port)))
	target := model.LabelSet{addressLabel: model.LabelValue(hostport)}
	if err := target.Validate(); err != nil {
		return nil, errors.Wrap(err, "failed to set targets")
	}

	res := &config.ScrapeConfig{
		JobName:        jobName(agent),
		ScrapeInterval: model.Duration(time.Second),
		ScrapeTimeout:  model.Duration(time.Second),
		MetricsPath:    "/metrics",
		ServiceDiscoveryConfig: sd_config.ServiceDiscoveryConfig{
			StaticConfigs: []*targetgroup.Group{{
				Targets: []model.LabelSet{target},
				Labels:  labels,
			}},
		},
	}

	return res, nil
}

func scrapeConfigsForMySQLdExporter(node *models.Node, service *models.Service, agent *models.Agent) ([]*config.ScrapeConfig, error) {
	labels := commonExporterLabelSet(node, service, agent)
	if err := mergeLabels(labels, node, service, agent); err != nil {
		return nil, err
	}

	hr := &config.ScrapeConfig{
		JobName:        jobName(agent) + "_hr",
		ScrapeInterval: model.Duration(time.Second),
		ScrapeTimeout:  model.Duration(time.Second),
		MetricsPath:    "/metrics-hr",
	}
	mr := &config.ScrapeConfig{
		JobName:        jobName(agent) + "_mr",
		ScrapeInterval: model.Duration(10 * time.Second),
		ScrapeTimeout:  model.Duration(5 * time.Second),
		MetricsPath:    "/metrics-mr",
	}
	lr := &config.ScrapeConfig{
		JobName:        jobName(agent) + "_lr",
		ScrapeInterval: model.Duration(60 * time.Second),
		ScrapeTimeout:  model.Duration(10 * time.Second),
		MetricsPath:    "/metrics-lr",
	}
	res := []*config.ScrapeConfig{hr, mr, lr}

	port := pointer.GetUint16(agent.ListenPort)
	if port == 0 {
		return nil, errors.New("listen port is not known")
	}
	hostport := net.JoinHostPort(pointer.GetString(node.Address), strconv.Itoa(int(port)))
	target := model.LabelSet{addressLabel: model.LabelValue(hostport)}
	if err := target.Validate(); err != nil {
		return nil, errors.Wrap(err, "failed to set targets")
	}

	for _, cfg := range res {
		cfg.ServiceDiscoveryConfig = sd_config.ServiceDiscoveryConfig{
			StaticConfigs: []*targetgroup.Group{{
				Targets: []model.LabelSet{target},
				Labels:  labels,
			}},
		}
	}

	return res, nil
}

func scrapeConfigForPostgresExporter(node *models.Node, service *models.Service, agent *models.Agent) (*config.ScrapeConfig, error) {
	labels := commonExporterLabelSet(node, service, agent)
	if err := mergeLabels(labels, node, service, agent); err != nil {
		return nil, err
	}

	cfg := &config.ScrapeConfig{
		JobName:        jobName(agent),
		ScrapeInterval: model.Duration(time.Second),
		ScrapeTimeout:  model.Duration(time.Second),
		MetricsPath:    "/metrics",
	}

	port := pointer.GetUint16(agent.ListenPort)
	if port == 0 {
		return nil, errors.New("listen port is not known")
	}
	hostport := net.JoinHostPort(pointer.GetString(node.Address), strconv.Itoa(int(port)))
	target := model.LabelSet{addressLabel: model.LabelValue(hostport)}
	if err := target.Validate(); err != nil {
		return nil, errors.Wrap(err, "failed to set targets")
	}

	cfg.ServiceDiscoveryConfig = sd_config.ServiceDiscoveryConfig{
		StaticConfigs: []*targetgroup.Group{{
			Targets: []model.LabelSet{target},
			Labels:  labels,
		}},
	}

	return cfg, nil
}

func scrapeConfigForMongoDBExporter(node *models.Node, service *models.Service, agent *models.Agent) (*config.ScrapeConfig, error) {
	labels := commonExporterLabelSet(node, service, agent)
	if err := mergeLabels(labels, node, service, agent); err != nil {
		return nil, err
	}

	port := pointer.GetUint16(agent.ListenPort)
	if port == 0 {
		return nil, errors.New("listen port is not known")
	}
	hostport := net.JoinHostPort(pointer.GetString(node.Address), strconv.Itoa(int(port)))
	target := model.LabelSet{addressLabel: model.LabelValue(hostport)}
	if err := target.Validate(); err != nil {
		return nil, errors.Wrap(err, "failed to set targets")
	}

	res := &config.ScrapeConfig{
		JobName:        jobName(agent),
		ScrapeInterval: model.Duration(time.Second),
		ScrapeTimeout:  model.Duration(time.Second),
		MetricsPath:    "/metrics",
		ServiceDiscoveryConfig: sd_config.ServiceDiscoveryConfig{
			StaticConfigs: []*targetgroup.Group{{
				Targets: []model.LabelSet{target},
				Labels:  labels,
			}},
		},
	}

	return res, nil
}
