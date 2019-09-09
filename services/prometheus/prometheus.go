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
	"github.com/pkg/errors"
	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/yaml.v2"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/services/prometheus/internal/prometheus/config"
)

const updateBatchDelay = 3 * time.Second

var checkFailedRE = regexp.MustCompile(`FAILED: parsing YAML file \S+: (.+)\n`)

// Service is responsible for interactions with Prometheus.
// It assumes the following:
//   * Prometheus APIs (including lifecycle) are accessible;
//   * Prometheus configuration and rule files are accessible;
//   * promtool is available.
type Service struct {
	configPath   string
	promtoolPath string
	db           *reform.DB
	baseURL      *url.URL
	client       *http.Client

	l    *logrus.Entry
	sema chan struct{}
}

// NewService creates new service.
func NewService(configPath string, promtoolPath string, db *reform.DB, baseURL string) (*Service, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &Service{
		configPath:   configPath,
		promtoolPath: promtoolPath,
		db:           db,
		baseURL:      u,
		client:       new(http.Client),
		l:            logrus.WithField("component", "prometheus"),
		sema:         make(chan struct{}, 1),
	}, nil
}

// Run runs Prometheus configuration update loop until ctx is canceled.
func (svc *Service) Run(ctx context.Context) {
	svc.l.Info("Starting...")
	defer svc.l.Info("Done.")

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

// marshalConfig marshals Prometheus configuration.
func (svc *Service) marshalConfig() ([]byte, error) {
	var cfg *config.Config
	e := svc.db.InTransaction(func(tx *reform.TX) error {
		settings, err := models.GetSettings(tx)
		if err != nil {
			return err
		}
		s := settings.MetricsResolutions

		cfg = &config.Config{
			GlobalConfig: config.GlobalConfig{
				ScrapeInterval:     model.Duration(s.LR),
				ScrapeTimeout:      scrapeTimeout(s.LR),
				EvaluationInterval: model.Duration(s.LR),
			},
			RuleFiles: []string{
				"/srv/prometheus/rules/*.rules.yml",
			},
			ScrapeConfigs: []*config.ScrapeConfig{
				scrapeConfigForPrometheus(s.HR),
				scrapeConfigForGrafana(s.MR),
				scrapeConfigForPMMManaged(s.MR),
			},
		}

		agents, err := tx.SelectAllFrom(models.AgentTable, "ORDER BY agent_type, agent_id")
		if err != nil {
			return errors.WithStack(err)
		}
		for _, str := range agents {
			agent := str.(*models.Agent)
			if agent.Disabled {
				continue
			}

			nodes, err := models.FindNodesForAgentID(tx.Querier, agent.AgentID)
			if err != nil {
				return err
			}
			services, err := models.ServicesForAgent(tx.Querier, agent.AgentID)
			if err != nil {
				return err
			}
			var host string
			if agent.AgentType != models.PMMAgentType {
				pmmAgent, err := models.FindAgentByID(tx.Querier, *agent.PMMAgentID)
				if err != nil {
					return errors.WithStack(err)
				}

				node := &models.Node{NodeID: pointer.GetString(pmmAgent.RunsOnNodeID)}
				if err = tx.Reload(node); err != nil {
					return errors.WithStack(err)
				}
				host = node.Address
			}

			switch agent.AgentType {
			case models.PMMAgentType:
				// TODO https://jira.percona.com/browse/PMM-4087
				continue

			case models.NodeExporterType:
				for _, node := range nodes {
					scfgs, err := scrapeConfigsForNodeExporter(&s, &scrapeConfigParams{
						host:    host,
						node:    node,
						service: nil,
						agent:   agent,
					})
					if err != nil {
						svc.l.Warnf("Failed to add %s %q, skipping: %s.", agent.AgentType, agent.AgentID, err)
						continue
					}
					cfg.ScrapeConfigs = append(cfg.ScrapeConfigs, scfgs...)
				}

			case models.MySQLdExporterType:
				for _, service := range services {
					node := &models.Node{NodeID: service.NodeID}
					if err = tx.Reload(node); err != nil {
						return errors.WithStack(err)
					}

					scfgs, err := scrapeConfigsForMySQLdExporter(&s, &scrapeConfigParams{
						host:    host,
						node:    node,
						service: service,
						agent:   agent,
					})
					if err != nil {
						svc.l.Warnf("Failed to add %s %q, skipping: %s.", agent.AgentType, agent.AgentID, err)
						continue
					}
					cfg.ScrapeConfigs = append(cfg.ScrapeConfigs, scfgs...)
				}

			case models.MongoDBExporterType:
				for _, service := range services {
					node := &models.Node{NodeID: service.NodeID}
					if err = tx.Reload(node); err != nil {
						return errors.WithStack(err)
					}

					scfgs, err := scrapeConfigsForMongoDBExporter(&s, &scrapeConfigParams{
						host:    host,
						node:    node,
						service: service,
						agent:   agent,
					})
					if err != nil {
						svc.l.Warnf("Failed to add %s %q, skipping: %s.", agent.AgentType, agent.AgentID, err)
						continue
					}
					cfg.ScrapeConfigs = append(cfg.ScrapeConfigs, scfgs...)
				}

			case models.PostgresExporterType:
				for _, service := range services {
					node := &models.Node{NodeID: service.NodeID}
					if err = tx.Reload(node); err != nil {
						return errors.WithStack(err)
					}

					scfgs, err := scrapeConfigsForPostgresExporter(&s, &scrapeConfigParams{
						host:    host,
						node:    node,
						service: service,
						agent:   agent,
					})
					if err != nil {
						svc.l.Warnf("Failed to add %s %q, skipping: %s.", agent.AgentType, agent.AgentID, err)
						continue
					}
					cfg.ScrapeConfigs = append(cfg.ScrapeConfigs, scfgs...)
				}

			case models.ProxySQLExporterType:
				for _, service := range services {
					node := &models.Node{NodeID: service.NodeID}
					if err = tx.Reload(node); err != nil {
						return errors.WithStack(err)
					}

					scfgs, err := scrapeConfigsForProxySQLExporter(&s, &scrapeConfigParams{
						host:    host,
						node:    node,
						service: service,
						agent:   agent,
					})
					if err != nil {
						svc.l.Warnf("Failed to add %s %q, skipping: %s.", agent.AgentType, agent.AgentID, err)
						continue
					}
					cfg.ScrapeConfigs = append(cfg.ScrapeConfigs, scfgs...)
				}

			case models.QANMySQLPerfSchemaAgentType, models.QANMySQLSlowlogAgentType:
				continue
			case models.QANMongoDBProfilerAgentType:
				continue
			case models.QANPostgreSQLPgStatementsAgentType:
				continue

			default:
				svc.l.Warnf("Skipping scrape config for %s.", agent)
			}
		}
		return nil
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

// saveConfigAndReload saves given Prometheus configuration to file and reloads Prometheus.
// If configuration can't be reloaded for some reason, old file is restored, and configuration is reloaded again.
func (svc *Service) saveConfigAndReload(cfg []byte) error {
	// read existing content
	oldCfg, err := ioutil.ReadFile(svc.configPath)
	if err != nil {
		return errors.WithStack(err)
	}

	// compare with new config
	if reflect.DeepEqual(cfg, oldCfg) {
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
	cmd := exec.Command(svc.promtoolPath, args...) //nolint:gosec
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

// Check verifies that Prometheus works.
func (svc *Service) Check(ctx context.Context) error {
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
	b, err = exec.CommandContext(ctx, svc.promtoolPath, "--version").CombinedOutput() //nolint:gosec
	if err != nil {
		return errors.Wrap(err, string(b))
	}
	svc.l.Debugf("%s", b)
	return nil
}
