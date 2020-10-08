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

// Package prometheus contains business logic of working with Prometheus.
package prometheus

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"reflect"
	"regexp"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/percona/pmm/utils/pdeathsig"
	config "github.com/percona/promconfig"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/yaml.v2"

	"github.com/percona/pmm-managed/models"
)

const (
	updateBatchDelay = 3 * time.Second
	// BasePrometheusConfigPath - basic path with prometheus config,
	// that user can mount to container.
	BasePrometheusConfigPath = "/srv/prometheus/prometheus.base.yml"
)

var checkFailedRE = regexp.MustCompile(`FAILED: parsing YAML file \S+: (.+)\n`)

// Service is responsible for interactions with Prometheus.
// It assumes the following:
//   * Prometheus APIs (including lifecycle) are accessible;
//   * Prometheus configuration and rule files are accessible;
//   * promtool is available.
type Service struct {
	alertingRules *AlertingRules
	configPath    string
	db            *reform.DB
	baseURL       *url.URL
	client        *http.Client

	baseConfigPath string // for testing

	l    *logrus.Entry
	sema chan struct{}

	cachedAlertingRules string
}

// NewService creates new service.
func NewService(alertingRules *AlertingRules, configPath string, db *reform.DB, baseURL string) (*Service, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &Service{
		alertingRules:  alertingRules,
		configPath:     configPath,
		db:             db,
		baseURL:        u,
		client:         new(http.Client),
		baseConfigPath: BasePrometheusConfigPath,
		l:              logrus.WithField("component", "prometheus"),
		sema:           make(chan struct{}, 1),
	}, nil
}

// Run runs Prometheus configuration update loop until ctx is canceled.
func (svc *Service) Run(ctx context.Context) {
	svc.l.Info("Starting...")
	defer svc.l.Info("Done.")

	alertingRules, err := svc.alertingRules.ReadRules()
	if err != nil {
		svc.l.Warnf("Cannot load alerting rules: %s", err)
	}
	svc.cachedAlertingRules = alertingRules

	for {
		select {
		case <-ctx.Done():
			return

		case <-svc.sema:
			// batch several update requests together by delaying the first one
			sleepCtx, sleepCancel := context.WithTimeout(ctx, updateBatchDelay)
			<-sleepCtx.Done()
			sleepCancel()

			if ctx.Err() != nil {
				return
			}

			if err := svc.updateConfiguration(); err != nil {
				svc.l.Errorf("Failed to update configuration, will retry: %+v.", err)
				svc.RequestConfigurationUpdate()
			}
		}
	}
}

// reload asks Prometheus to reload configuration.
func (svc *Service) reload() error {
	u := *svc.baseURL
	u.Path = path.Join(u.Path, "-", "reload")
	resp, err := svc.client.Post(u.String(), "", nil)
	if err != nil {
		return errors.WithStack(err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode == 200 {
		return nil
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.WithStack(err)
	}
	return errors.Errorf("%d: %s", resp.StatusCode, b)
}

func (svc *Service) loadBaseConfig() *config.Config {
	var cfg config.Config

	buf, err := ioutil.ReadFile(svc.baseConfigPath)
	if err != nil {
		if !os.IsNotExist(err) {
			svc.l.Errorf("Failed to load base prometheus config %s: %s", svc.baseConfigPath, err)
		}
		return &cfg
	}

	if err := yaml.Unmarshal(buf, &cfg); err != nil {
		svc.l.Errorf("Failed to parse base prometheus config %s: %s.", svc.baseConfigPath, err)
		return &config.Config{}
	}

	return &cfg
}

// AddScrapeConfigs wraps addScrapeConfigs for victoriametrics package.
func AddScrapeConfigs(l *logrus.Entry, cfg *config.Config, q *reform.Querier, s *models.MetricsResolutions) error {
	return addScrapeConfigs(l, cfg, q, s)
}

// addScrapeConfigs adds Prometheus scrape configs to cfg for all Agents.
func addScrapeConfigs(l *logrus.Entry, cfg *config.Config, q *reform.Querier, s *models.MetricsResolutions) error {
	agents, err := q.SelectAllFrom(models.AgentTable, "WHERE NOT disabled AND listen_port IS NOT NULL ORDER BY agent_type, agent_id")
	if err != nil {
		return errors.WithStack(err)
	}

	var rdsParams []*scrapeConfigParams
	for _, str := range agents {
		agent := str.(*models.Agent)

		if agent.AgentType == models.PMMAgentType {
			// TODO https://jira.percona.com/browse/PMM-4087
			continue
		}

		// sanity check
		if (agent.NodeID != nil) && (agent.ServiceID != nil) {
			l.Panicf("Both agent.NodeID and agent.ServiceID are present: %s", agent)
		}

		// find Service for this Agent
		var paramsService *models.Service
		if agent.ServiceID != nil {
			paramsService, err = models.FindServiceByID(q, pointer.GetString(agent.ServiceID))
			if err != nil {
				return err
			}
		}

		// find Node for this Agent or Service
		var paramsNode *models.Node
		switch {
		case agent.NodeID != nil:
			paramsNode, err = models.FindNodeByID(q, pointer.GetString(agent.NodeID))
		case paramsService != nil:
			paramsNode, err = models.FindNodeByID(q, paramsService.NodeID)
		}
		if err != nil {
			return err
		}

		// find Node address where the agent runs
		var paramsHost string
		switch {
		case agent.PMMAgentID != nil:
			// extract node address through pmm-agent
			pmmAgent, err := models.FindAgentByID(q, *agent.PMMAgentID)
			if err != nil {
				return errors.WithStack(err)
			}
			pmmAgentNode := &models.Node{NodeID: pointer.GetString(pmmAgent.RunsOnNodeID)}
			if err = q.Reload(pmmAgentNode); err != nil {
				return errors.WithStack(err)
			}
			paramsHost = pmmAgentNode.Address
		case agent.RunsOnNodeID != nil:
			externalExporterNode := &models.Node{NodeID: pointer.GetString(agent.RunsOnNodeID)}
			if err = q.Reload(externalExporterNode); err != nil {
				return errors.WithStack(err)
			}
			paramsHost = externalExporterNode.Address
		default:
			l.Warnf("It's not possible to get host, skipping scrape config for %s.", agent)

			continue
		}

		var scfgs []*config.ScrapeConfig
		switch agent.AgentType {
		case models.NodeExporterType:
			scfgs, err = scrapeConfigsForNodeExporter(s, &scrapeConfigParams{
				host:    paramsHost,
				node:    paramsNode,
				service: nil,
				agent:   agent,
			})

		case models.MySQLdExporterType:
			scfgs, err = scrapeConfigsForMySQLdExporter(s, &scrapeConfigParams{
				host:    paramsHost,
				node:    paramsNode,
				service: paramsService,
				agent:   agent,
			})

		case models.MongoDBExporterType:
			scfgs, err = scrapeConfigsForMongoDBExporter(s, &scrapeConfigParams{
				host:    paramsHost,
				node:    paramsNode,
				service: paramsService,
				agent:   agent,
			})

		case models.PostgresExporterType:
			scfgs, err = scrapeConfigsForPostgresExporter(s, &scrapeConfigParams{
				host:    paramsHost,
				node:    paramsNode,
				service: paramsService,
				agent:   agent,
			})

		case models.ProxySQLExporterType:
			scfgs, err = scrapeConfigsForProxySQLExporter(s, &scrapeConfigParams{
				host:    paramsHost,
				node:    paramsNode,
				service: paramsService,
				agent:   agent,
			})

		case models.QANMySQLPerfSchemaAgentType, models.QANMySQLSlowlogAgentType:
			continue
		case models.QANMongoDBProfilerAgentType:
			continue
		case models.QANPostgreSQLPgStatementsAgentType, models.QANPostgreSQLPgStatMonitorAgentType:
			continue

		case models.RDSExporterType:
			rdsParams = append(rdsParams, &scrapeConfigParams{
				host:    paramsHost,
				node:    paramsNode,
				service: paramsService,
				agent:   agent,
			})
			continue

		case models.ExternalExporterType:
			scfgs, err = scrapeConfigsForExternalExporter(s, &scrapeConfigParams{
				host:    paramsHost,
				node:    paramsNode,
				service: paramsService,
				agent:   agent,
			})

		default:
			l.Warnf("Skipping scrape config for %s.", agent)
			continue
		}

		if err != nil {
			l.Warnf("Failed to add %s %q, skipping: %s.", agent.AgentType, agent.AgentID, err)
		}
		cfg.ScrapeConfigs = append(cfg.ScrapeConfigs, scfgs...)
	}

	scfgs := scrapeConfigsForRDSExporter(s, rdsParams)
	cfg.ScrapeConfigs = append(cfg.ScrapeConfigs, scfgs...)

	return nil
}

// marshalConfig marshals Prometheus configuration.
func (svc *Service) marshalConfig() ([]byte, error) {
	cfg := svc.loadBaseConfig()

	e := svc.db.InTransaction(func(tx *reform.TX) error {
		settings, err := models.GetSettings(tx)
		if err != nil {
			return err
		}
		s := settings.MetricsResolutions

		if cfg.GlobalConfig.ScrapeInterval == 0 {
			cfg.GlobalConfig.ScrapeInterval = config.Duration(s.LR)
		}
		if cfg.GlobalConfig.ScrapeTimeout == 0 {
			cfg.GlobalConfig.ScrapeTimeout = scrapeTimeout(s.LR)
		}
		if cfg.GlobalConfig.EvaluationInterval == 0 {
			cfg.GlobalConfig.EvaluationInterval = config.Duration(s.LR)
		}

		cfg.RuleFiles = append(
			cfg.RuleFiles,

			// That covers all .yml files, including:
			// pmm.rules.yml managed by pmm-managed;
			// user-supplied files.
			"/srv/prometheus/rules/*.yml",
		)

		AddInternalServicesToScrape(cfg, s, settings.DBaaS.Enabled)

		cfg.AlertingConfig.AlertmanagerConfigs = append(cfg.AlertingConfig.AlertmanagerConfigs, &config.AlertmanagerConfig{
			ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
				StaticConfigs: []*config.Group{{
					Targets: []string{"127.0.0.1:9093"},
				}},
			},
			Scheme:     "http",
			PathPrefix: "/alertmanager/",
			APIVersion: config.AlertmanagerAPIVersionV2,
		})

		if settings.AlertManagerURL != "" {
			u, err := url.Parse(settings.AlertManagerURL)
			if err == nil && (u.Opaque != "" || u.Host == "") {
				err = errors.Errorf("parsed incorrectly as %#v", u)
			}

			if err == nil {
				var httpClientConfig config.HTTPClientConfig
				if username := u.User.Username(); username != "" {
					password, _ := u.User.Password()
					httpClientConfig = config.HTTPClientConfig{
						BasicAuth: &config.BasicAuth{
							Username: u.User.Username(),
							Password: password,
						},
					}
				}

				cfg.AlertingConfig.AlertmanagerConfigs = append(cfg.AlertingConfig.AlertmanagerConfigs, &config.AlertmanagerConfig{
					ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
						StaticConfigs: []*config.Group{{
							Targets: []string{u.Host},
						}},
					},
					HTTPClientConfig: httpClientConfig,
					Scheme:           u.Scheme,
					PathPrefix:       u.Path,
					APIVersion:       config.AlertmanagerAPIVersionV2,
				})
			} else {
				svc.l.Errorf("Failed to parse Alert Manager URL %q: %s.", settings.AlertManagerURL, err)
			}
		}

		return addScrapeConfigs(svc.l, cfg, tx.Querier, &s)
	})
	if e != nil {
		return nil, e
	}

	// TODO Add comments to each cfg.ScrapeConfigs element.
	// https://jira.percona.com/browse/PMM-3601

	b, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "can't marshal Prometheus configuration file")
	}

	b = append([]byte("# Managed by pmm-managed. DO NOT EDIT.\n---\n"), b...)
	return b, nil
}

// AddInternalServicesToScrape adds internal services metrics to scrape targets.
func AddInternalServicesToScrape(cfg *config.Config, s models.MetricsResolutions, dbaas bool) {
	cfg.ScrapeConfigs = append(cfg.ScrapeConfigs,
		scrapeConfigForPrometheus(s.HR),
		scrapeConfigForAlertmanager(s.MR),
		scrapeConfigForGrafana(s.MR),
		scrapeConfigForPMMManaged(s.MR),
		scrapeConfigForQANAPI2(s.MR),
	)

	// TODO Refactor to remove boolean positional parameter when Prometheus or PERCONA_TEST_DBAAS is removed
	if dbaas {
		cfg.ScrapeConfigs = append(cfg.ScrapeConfigs, scrapeConfigForDBaaSController(s.MR))
	}
}

// saveConfigAndReload saves given Prometheus configuration to file and reloads Prometheus.
// If configuration can't be reloaded for some reason, old file is restored, and configuration is reloaded again.
func (svc *Service) saveConfigAndReload(cfg []byte) error {
	// read existing content
	oldCfg, err := ioutil.ReadFile(svc.configPath)
	if err != nil {
		return errors.WithStack(err)
	}

	alertingRules, err := svc.alertingRules.ReadRules()
	if err != nil {
		svc.l.Warnf("Cannot load alerting rules: %s", err)
	}
	// compare with new config
	if reflect.DeepEqual(cfg, oldCfg) && alertingRules == svc.cachedAlertingRules {
		svc.l.Infof("Configuration not changed, doing nothing.")
		return nil
	}

	fi, err := os.Stat(svc.configPath)
	if err != nil {
		return errors.WithStack(err)
	}

	// restore old content and reload in case of error
	var restore bool
	defer func() {
		if restore {
			if err = ioutil.WriteFile(svc.configPath, oldCfg, fi.Mode()); err != nil {
				svc.l.Error(err)
			}
			if err = svc.reload(); err != nil {
				svc.l.Error(err)
			}
		}
	}()

	// write new content to temporary file, check it
	f, err := ioutil.TempFile("", "pmm-managed-config-")
	if err != nil {
		return errors.WithStack(err)
	}
	if _, err = f.Write(cfg); err != nil {
		return errors.WithStack(err)
	}
	defer func() {
		_ = f.Close()
		_ = os.Remove(f.Name())
	}()
	args := []string{"check", "config", f.Name()}
	cmd := exec.Command("promtool", args...) //nolint:gosec
	pdeathsig.Set(cmd, unix.SIGKILL)
	b, err := cmd.CombinedOutput()
	if err != nil {
		svc.l.Errorf("%s", b)

		// return typed error if possible
		s := string(b)
		if m := checkFailedRE.FindStringSubmatch(s); len(m) == 2 {
			return status.Error(codes.Aborted, m[1])
		}
		return errors.Wrap(err, s)
	}
	svc.l.Debugf("%s", b)

	// write to permanent location and reload
	restore = true
	if err = ioutil.WriteFile(svc.configPath, cfg, fi.Mode()); err != nil {
		return errors.WithStack(err)
	}
	if err = svc.reload(); err != nil {
		return err
	}
	svc.l.Infof("Configuration reloaded.")
	restore = false
	svc.cachedAlertingRules = alertingRules
	return nil
}

// updateConfiguration updates Prometheus configuration.
func (svc *Service) updateConfiguration() error {
	start := time.Now()
	defer func() {
		if dur := time.Since(start); dur > time.Second {
			svc.l.Warnf("updateConfiguration took %s.", dur)
		}
	}()

	cfg, err := svc.marshalConfig()
	if err != nil {
		return err
	}
	return svc.saveConfigAndReload(cfg)
}

// RequestConfigurationUpdate requests Prometheus configuration update.
func (svc *Service) RequestConfigurationUpdate() {
	select {
	case svc.sema <- struct{}{}:
	default:
	}
}

// IsReady verifies that Prometheus works.
func (svc *Service) IsReady(ctx context.Context) error {
	// check Prometheus /version API and log version
	u := *svc.baseURL
	u.Path = path.Join(u.Path, "version")
	resp, err := svc.client.Get(u.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck
	b, err := ioutil.ReadAll(resp.Body)
	svc.l.Debugf("Prometheus: %s", b)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return errors.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// check promtool version
	b, err = exec.CommandContext(ctx, "promtool", "--version").CombinedOutput()
	if err != nil {
		return errors.Wrap(err, string(b))
	}
	svc.l.Debugf("%s", b)
	return nil
}
