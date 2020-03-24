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

	config "github.com/percona/promconfig"

	"github.com/percona/pmm-managed/models"
)

// scrapeTimeout returns default scrape timeout for given scrape interval.
//nolint:gomnd
func scrapeTimeout(interval time.Duration) config.Duration {
	switch {
	case interval <= 2*time.Second:
		return config.Duration(time.Second)
	case interval <= 10*time.Second:
		return config.Duration(interval - time.Second)
	default:
		return config.Duration(10 * time.Second)
	}
}

func scrapeConfigForPrometheus(interval time.Duration) *config.ScrapeConfig {
	return &config.ScrapeConfig{
		JobName:        "prometheus",
		ScrapeInterval: config.Duration(interval),
		ScrapeTimeout:  scrapeTimeout(interval),
		MetricsPath:    "/prometheus/metrics",
		ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
			StaticConfigs: []*config.Group{{
				Targets: []string{"127.0.0.1:9090"},
				Labels:  map[string]string{"instance": "pmm-server"},
			}},
		},
	}
}

func scrapeConfigForGrafana(interval time.Duration) *config.ScrapeConfig {
	return &config.ScrapeConfig{
		JobName:        "grafana",
		ScrapeInterval: config.Duration(interval),
		ScrapeTimeout:  scrapeTimeout(interval),
		MetricsPath:    "/metrics",
		ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
			StaticConfigs: []*config.Group{{
				Targets: []string{"127.0.0.1:3000"},
				Labels:  map[string]string{"instance": "pmm-server"},
			}},
		},
	}
}

func scrapeConfigForPMMManaged(interval time.Duration) *config.ScrapeConfig {
	return &config.ScrapeConfig{
		JobName:        "pmm-managed",
		ScrapeInterval: config.Duration(interval),
		ScrapeTimeout:  scrapeTimeout(interval),
		MetricsPath:    "/debug/metrics",
		ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
			StaticConfigs: []*config.Group{{
				Targets: []string{"127.0.0.1:7773"},
				Labels:  map[string]string{"instance": "pmm-server"},
			}},
		},
	}
}

func scrapeConfigForQANAPI2(interval time.Duration) *config.ScrapeConfig {
	return &config.ScrapeConfig{
		JobName:        "qan-api2",
		ScrapeInterval: config.Duration(interval),
		ScrapeTimeout:  scrapeTimeout(interval),
		MetricsPath:    "/debug/metrics",
		ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
			StaticConfigs: []*config.Group{{
				Targets: []string{"127.0.0.1:9933"},
				Labels:  map[string]string{"instance": "pmm-server"},
			}},
		},
	}
}

func mergeLabels(node *models.Node, service *models.Service, agent *models.Agent) (map[string]string, error) {
	res := make(map[string]string, 16)

	labels, err := node.UnifiedLabels()
	if err != nil {
		return nil, err
	}
	for name, value := range labels {
		res[name] = value
	}

	if service != nil {
		labels, err = service.UnifiedLabels()
		if err != nil {
			return nil, err
		}
		for name, value := range labels {
			res[name] = value
		}
	}

	labels, err = agent.UnifiedLabels()
	if err != nil {
		return nil, err
	}
	for name, value := range labels {
		res[name] = value
	}

	res["instance"] = agent.AgentID

	return res, nil
}

// jobNameMapping replaces runes that can't be present in Prometheus job name with '_'.
func jobNameMapping(r rune) rune {
	switch r {
	case '/', ':', '.':
		return '_'
	default:
		return r
	}
}

func jobName(agent *models.Agent, intervalName string, interval time.Duration) string {
	return fmt.Sprintf("%s%s_%s-%s", agent.AgentType, strings.Map(jobNameMapping, agent.AgentID), intervalName, interval)
}

func httpClientConfig(agent *models.Agent) config.HTTPClientConfig {
	return config.HTTPClientConfig{
		BasicAuth: &config.BasicAuth{
			Username: "pmm",
			Password: agent.AgentID,
		},
	}
}

type scrapeConfigParams struct {
	host    string // Node address where pmm-agent runs
	node    *models.Node
	service *models.Service
	agent   *models.Agent
}

// scrapeConfigForStandardExporter returns scrape config for endpoint with given parameters.
func scrapeConfigForStandardExporter(intervalName string, interval time.Duration, params *scrapeConfigParams, collect []string) (*config.ScrapeConfig, error) {
	labels, err := mergeLabels(params.node, params.service, params.agent)
	if err != nil {
		return nil, err
	}

	cfg := &config.ScrapeConfig{
		JobName:          jobName(params.agent, intervalName, interval),
		ScrapeInterval:   config.Duration(interval),
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

	port := int(*params.agent.ListenPort)
	hostport := net.JoinHostPort(params.host, strconv.Itoa(port))

	cfg.ServiceDiscoveryConfig = config.ServiceDiscoveryConfig{
		StaticConfigs: []*config.Group{{
			Targets: []string{hostport},
			Labels:  labels,
		}},
	}

	return cfg, nil
}

// scrapeConfigForRDSExporter returns scrape config for single rds_exporter configuration.
func scrapeConfigForRDSExporter(intervalName string, interval time.Duration, hostport string, metricsPath string) (*config.ScrapeConfig, error) {
	jobName := fmt.Sprintf("rds_exporter_%s_%s-%s", strings.Map(jobNameMapping, hostport), intervalName, interval)
	cfg := &config.ScrapeConfig{
		JobName:        jobName,
		ScrapeInterval: config.Duration(interval),
		ScrapeTimeout:  scrapeTimeout(interval),
		MetricsPath:    metricsPath,
		HonorLabels:    true,
	}

	cfg.ServiceDiscoveryConfig = config.ServiceDiscoveryConfig{
		StaticConfigs: []*config.Group{{
			Targets: []string{hostport},
		}},
	}

	return cfg, nil
}

func scrapeConfigsForNodeExporter(s *models.MetricsResolutions, params *scrapeConfigParams) ([]*config.ScrapeConfig, error) {
	var hr, mr, lr *config.ScrapeConfig
	var err error
	var hrCollect []string

	if params.node.Distro != "darwin" {
		mr, err = scrapeConfigForStandardExporter("mr", s.MR, params, []string{
			"hwmon",
			"textfile.mr",
		})
		if err != nil {
			return nil, err
		}

		lr, err = scrapeConfigForStandardExporter("lr", s.LR, params, []string{
			"bonding",
			"entropy",
			"textfile.lr",
			"uname",
		})
		if err != nil {
			return nil, err
		}

		hrCollect = append(hrCollect,
			"buddyinfo",
			"filefd",
			"meminfo_numa",
			"netstat",
			"processes",
			"standard.go",
			"standard.process",
			"stat",
			"textfile.hr",
			"vmstat",
		)
	}

	hr, err = scrapeConfigForStandardExporter("hr", s.HR, params, append(hrCollect,
		"cpu",
		"diskstats",
		"filesystem",
		"loadavg",
		"meminfo",
		"netdev",
		"time",
	))
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

func scrapeConfigsForRDSExporter(s *models.MetricsResolutions, params []*scrapeConfigParams) ([]*config.ScrapeConfig, error) {
	hostportSet := make(map[string]struct{}, len(params))
	for _, p := range params {
		port := int(*p.agent.ListenPort)
		hostport := net.JoinHostPort(p.host, strconv.Itoa(port))
		hostportSet[hostport] = struct{}{}
	}

	hostports := make([]string, 0, len(hostportSet))
	for hostport := range hostportSet {
		hostports = append(hostports, hostport)
	}
	sort.Strings(hostports)

	var r []*config.ScrapeConfig
	for _, hostport := range hostports {
		mr, err := scrapeConfigForRDSExporter("mr", s.MR, hostport, "/enhanced")
		if err != nil {
			return nil, err
		}
		if mr != nil {
			r = append(r, mr)
		}

		lr, err := scrapeConfigForRDSExporter("lr", s.LR, hostport, "/basic")
		if err != nil {
			return nil, err
		}
		if lr != nil {
			r = append(r, lr)
		}
	}

	return r, nil
}
