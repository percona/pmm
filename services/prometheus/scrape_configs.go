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
	"fmt"
	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"
	"github.com/prometheus/common/model"

	"github.com/percona/pmm-managed/models"
	config_util "github.com/percona/pmm-managed/services/prometheus/internal/common/config"
	"github.com/percona/pmm-managed/services/prometheus/internal/prometheus/config"
	sd_config "github.com/percona/pmm-managed/services/prometheus/internal/prometheus/discovery/config"
	"github.com/percona/pmm-managed/services/prometheus/internal/prometheus/discovery/targetgroup"
)

const addressLabel = model.LabelName(model.AddressLabel)

// scrapeTimeout returns default scrape timeout for given scrape interval.
func scrapeTimeout(interval time.Duration) model.Duration {
	switch {
	case interval <= 2*time.Second:
		return model.Duration(time.Second)
	case interval <= 10*time.Second:
		return model.Duration(interval - time.Second)
	default:
		return model.Duration(10 * time.Second)
	}
}

func scrapeConfigForPrometheus(interval time.Duration) *config.ScrapeConfig {
	return &config.ScrapeConfig{
		JobName:        "prometheus",
		ScrapeInterval: model.Duration(interval),
		ScrapeTimeout:  scrapeTimeout(interval),
		MetricsPath:    "/prometheus/metrics",
		ServiceDiscoveryConfig: sd_config.ServiceDiscoveryConfig{
			StaticConfigs: []*targetgroup.Group{{
				Targets: []model.LabelSet{{addressLabel: "127.0.0.1:9090"}},
				Labels:  model.LabelSet{"instance": "pmm-server"},
			}},
		},
	}
}

func scrapeConfigForGrafana(interval time.Duration) *config.ScrapeConfig {
	return &config.ScrapeConfig{
		JobName:        "grafana",
		ScrapeInterval: model.Duration(interval),
		ScrapeTimeout:  scrapeTimeout(interval),
		MetricsPath:    "/metrics",
		ServiceDiscoveryConfig: sd_config.ServiceDiscoveryConfig{
			StaticConfigs: []*targetgroup.Group{{
				Targets: []model.LabelSet{{addressLabel: "127.0.0.1:3000"}},
				Labels:  model.LabelSet{"instance": "pmm-server"},
			}},
		},
	}
}

func scrapeConfigForPMMManaged(interval time.Duration) *config.ScrapeConfig {
	return &config.ScrapeConfig{
		JobName:        "pmm-managed",
		ScrapeInterval: model.Duration(interval),
		ScrapeTimeout:  scrapeTimeout(interval),
		MetricsPath:    "/debug/metrics",
		ServiceDiscoveryConfig: sd_config.ServiceDiscoveryConfig{
			StaticConfigs: []*targetgroup.Group{{
				Targets: []model.LabelSet{{addressLabel: "127.0.0.1:7773"}},
				Labels:  model.LabelSet{"instance": "pmm-server"},
			}},
		},
	}
}

func mergeLabels(node *models.Node, service *models.Service, agent *models.Agent) (model.LabelSet, error) {
	res := make(model.LabelSet, 16)

	labels, err := node.UnifiedLabels()
	if err != nil {
		return nil, err
	}
	for name, value := range labels {
		res[model.LabelName(name)] = model.LabelValue(value)
	}

	if service != nil {
		labels, err = service.UnifiedLabels()
		if err != nil {
			return nil, err
		}
		for name, value := range labels {
			res[model.LabelName(name)] = model.LabelValue(value)
		}
	}

	labels, err = agent.UnifiedLabels()
	if err != nil {
		return nil, err
	}
	for name, value := range labels {
		res[model.LabelName(name)] = model.LabelValue(value)
	}

	res[model.LabelName("instance")] = model.LabelValue(agent.AgentID)

	if err = res.Validate(); err != nil {
		return nil, errors.Wrap(err, "failed to merge labels")
	}
	return res, nil
}

func jobName(agent *models.Agent, intervalName string, interval time.Duration) string {
	return fmt.Sprintf("%s%s_%s-%s", agent.AgentType, strings.Replace(agent.AgentID, "/", "_", -1), intervalName, interval)
}

func httpClientConfig(agent *models.Agent) config_util.HTTPClientConfig {
	return config_util.HTTPClientConfig{
		BasicAuth: &config_util.BasicAuth{
			Username: "pmm",
			Password: agent.AgentID,
		},
	}
}

type scrapeConfigParams struct {
	host    string
	node    *models.Node
	service *models.Service
	agent   *models.Agent
}

// scrapeConfigForStandardExporter returns scrape config for endpoint with given parameters.
// If listen port is not known yet, it returns (nil, nil).
func scrapeConfigForStandardExporter(intervalName string, interval time.Duration, params *scrapeConfigParams, collect []string) (*config.ScrapeConfig, error) {
	labels, err := mergeLabels(params.node, params.service, params.agent)
	if err != nil {
		return nil, err
	}

	cfg := &config.ScrapeConfig{
		JobName:          jobName(params.agent, intervalName, interval),
		ScrapeInterval:   model.Duration(interval),
		ScrapeTimeout:    scrapeTimeout(interval),
		MetricsPath:      "/metrics",
		HTTPClientConfig: httpClientConfig(params.agent),
	}

	if len(collect) > 0 {
		sort.Strings(collect)
		cfg.Params = url.Values{
			"collect[]": collect,
		}
	}

	port := pointer.GetUint16(params.agent.ListenPort)
	if port == 0 {
		return nil, nil
	}
	hostport := net.JoinHostPort(params.host, strconv.Itoa(int(port)))
	target := model.LabelSet{addressLabel: model.LabelValue(hostport)}
	if err = target.Validate(); err != nil {
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

func scrapeConfigsForNodeExporter(s *models.MetricsResolutions, params *scrapeConfigParams) ([]*config.ScrapeConfig, error) {
	hr, err := scrapeConfigForStandardExporter("hr", s.HR, params, []string{
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
	})
	if err != nil {
		return nil, err
	}

	mr, err := scrapeConfigForStandardExporter("mr", s.MR, params, []string{
		"textfile.mr",
	})
	if err != nil {
		return nil, err
	}

	lr, err := scrapeConfigForStandardExporter("lr", s.LR, params, []string{
		"bonding",
		"entropy",
		"textfile.lr",
		"uname",
	})
	if err != nil {
		return nil, err
	}

	var r []*config.ScrapeConfig
	if hr != nil {
		r = append(r, hr)
	}
	if mr != nil {
		r = append(r, mr)
	}
	if lr != nil {
		r = append(r, lr)
	}
	return r, nil
}

// scrapeConfigsForMySQLdExporter returns scrape config for mysqld_exporter.
// If listen port is not known yet, it returns (nil, nil).
func scrapeConfigsForMySQLdExporter(s *models.MetricsResolutions, params *scrapeConfigParams) ([]*config.ScrapeConfig, error) {
	// keep in sync with mysqld_exporter Agent flags generator

	hr, err := scrapeConfigForStandardExporter("hr", s.HR, params, []string{
		"global_status",
		"info_schema.innodb_metrics",
		"custom_query.hr",
		"standard.go",
		"standard.process",
	})
	if err != nil {
		return nil, err
	}

	mrOptions := []string{
		"engine_innodb_status",
		"info_schema.innodb_cmp",
		"info_schema.innodb_cmpmem",
		"info_schema.processlist",
		"info_schema.query_response_time",
		"perf_schema.eventswaits",
		"perf_schema.file_events",
		"slave_status",
		"custom_query.mr",
	}
	if params.agent.IsMySQLTablestatsGroupEnabled() {
		mrOptions = append(mrOptions, "perf_schema.tablelocks")
	}

	mr, err := scrapeConfigForStandardExporter("mr", s.MR, params, mrOptions)
	if err != nil {
		return nil, err
	}

	lrOptions := []string{
		"binlog_size",
		"engine_tokudb_status",
		"global_variables",
		"heartbeat",
		"info_schema.clientstats",
		"info_schema.innodb_tablespaces",
		"info_schema.userstats",
		"perf_schema.eventsstatements",
		"perf_schema.file_instances",
		"custom_query.lr",
	}
	if params.agent.IsMySQLTablestatsGroupEnabled() {
		lrOptions = append(lrOptions,
			"auto_increment.columns",
			"info_schema.tables",
			"info_schema.tablestats",
			"perf_schema.indexiowaits",
			"perf_schema.tableiowaits",
		)
	}

	lr, err := scrapeConfigForStandardExporter("lr", s.LR, params, lrOptions)
	if err != nil {
		return nil, err
	}

	var r []*config.ScrapeConfig
	if hr != nil {
		r = append(r, hr)
	}
	if mr != nil {
		r = append(r, mr)
	}
	if lr != nil {
		r = append(r, lr)
	}
	return r, nil
}

func scrapeConfigsForMongoDBExporter(s *models.MetricsResolutions, params *scrapeConfigParams) ([]*config.ScrapeConfig, error) {
	hr, err := scrapeConfigForStandardExporter("hr", s.HR, params, nil)
	if err != nil {
		return nil, err
	}

	var r []*config.ScrapeConfig
	if hr != nil {
		r = append(r, hr)
	}
	return r, nil
}

func scrapeConfigsForPostgresExporter(s *models.MetricsResolutions, params *scrapeConfigParams) ([]*config.ScrapeConfig, error) {
	hr, err := scrapeConfigForStandardExporter("hr", s.HR, params, []string{
		"exporter",
		"custom_query.hr",
		"standard.go",
		"standard.process",
	})
	if err != nil {
		return nil, err
	}

	mr, err := scrapeConfigForStandardExporter("mr", s.MR, params, []string{
		"custom_query.mr",
	})
	if err != nil {
		return nil, err
	}

	lr, err := scrapeConfigForStandardExporter("lr", s.LR, params, []string{
		"custom_query.lr",
	})
	if err != nil {
		return nil, err
	}

	var r []*config.ScrapeConfig
	if hr != nil {
		r = append(r, hr)
	}
	if mr != nil {
		r = append(r, mr)
	}
	if lr != nil {
		r = append(r, lr)
	}
	return r, nil
}

func scrapeConfigsForProxySQLExporter(s *models.MetricsResolutions, params *scrapeConfigParams) ([]*config.ScrapeConfig, error) {
	hr, err := scrapeConfigForStandardExporter("hr", s.HR, params, nil) // TODO https://jira.percona.com/browse/PMM-4619
	if err != nil {
		return nil, err
	}

	var r []*config.ScrapeConfig
	if hr != nil {
		r = append(r, hr)
	}
	return r, nil
}
