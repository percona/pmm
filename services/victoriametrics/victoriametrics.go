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

// Package victoriametrics provides facilities for working with VictoriaMetrics.
package victoriametrics

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"regexp"
	"time"

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
	"github.com/percona/pmm-managed/services/prometheus"
)

const (
	updateBatchDelay           = 3 * time.Second
	configurationUpdateTimeout = 2 * time.Second
	// BasePrometheusConfigPath - basic path with prometheus config,
	// that user can mount to container.
	BasePrometheusConfigPath = "/srv/prometheus/prometheus.base.yml"
)

var checkFailedRE = regexp.MustCompile(`FAILED: parsing YAML file \S+: (.+)\n`)

// Service is responsible for interactions with victoria metrics.
type Service struct {
	scrapeConfigPath string
	db               *reform.DB
	baseURL          *url.URL
	client           *http.Client

	baseConfigPath string // for testing

	l    *logrus.Entry
	sema chan struct{}
}

// NewVictoriaMetrics creates new Victoria Metrics service.
func NewVictoriaMetrics(scrapeConfigPath string, db *reform.DB, baseURL string, params *models.VictoriaMetricsParams) (*Service, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &Service{
		scrapeConfigPath: scrapeConfigPath,
		db:               db,
		baseURL:          u,
		client:           new(http.Client),
		baseConfigPath:   params.BaseConfigPath,
		l:                logrus.WithField("component", "victoriametrics"),
		sema:             make(chan struct{}, 1),
	}, nil
}

// Run runs VictoriaMetrics configuration update loop until ctx is canceled.
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

			if err := svc.updateConfiguration(ctx); err != nil {
				svc.l.Errorf("Failed to update configuration, will retry: %+v.", err)
				svc.RequestConfigurationUpdate()
			}
		}
	}
}

// updateConfiguration updates Prometheus configuration.
func (svc *Service) updateConfiguration(ctx context.Context) error {
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

	return svc.configAndReload(ctx, cfg)
}

// RequestConfigurationUpdate requests VictoriaMetrics configuration update.
func (svc *Service) RequestConfigurationUpdate() {
	select {
	case svc.sema <- struct{}{}:
		ctx, cancel := context.WithTimeout(context.Background(), configurationUpdateTimeout)
		defer cancel()
		err := svc.updateConfiguration(ctx)
		if err != nil {
			svc.l.WithError(err).Errorf("cannot reload configuration")
		}

	default:
	}
}

// IsReady verifies that VictoriaMetrics works.
func (svc *Service) IsReady(ctx context.Context) error {
	// check VictoriaMetrics /health API
	u := *svc.baseURL
	u.Path = path.Join(u.Path, "health")
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return errors.WithStack(err)
	}
	resp, err := svc.client.Do(req)
	if err != nil {
		return errors.WithStack(err)
	}
	defer resp.Body.Close() //nolint:errcheck
	b, err := ioutil.ReadAll(resp.Body)
	svc.l.Debugf("VictoriaMetrics: %s", b)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return errors.Errorf("expected 200, got %d", resp.StatusCode)
	}

	return nil
}

// reload asks VictoriaMetrics to reload configuration.
func (svc *Service) reload(ctx context.Context) error {
	u := *svc.baseURL
	u.Path = path.Join(u.Path, "-", "reload")
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return errors.WithStack(err)
	}

	resp, err := svc.client.Do(req)
	if err != nil {
		return errors.WithStack(err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode == http.StatusNoContent {
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
			svc.l.Errorf("Failed to load base victoriametrics config %s: %s", svc.baseConfigPath, err)
		}

		return &cfg
	}
	if err := yaml.Unmarshal(buf, &cfg); err != nil {
		svc.l.Errorf("Failed to parse base victoriametrics config %s: %s.", svc.baseConfigPath, err)

		return &config.Config{}
	}

	return &cfg
}

// marshalConfig marshals VictoriaMetrics configuration.
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
			cfg.GlobalConfig.ScrapeTimeout = prometheus.ScrapeTimeout(s.LR)
		}
		cfg.ScrapeConfigs = append(cfg.ScrapeConfigs, scrapeConfigForVictoriaMetrics(s.HR), scrapeConfigForVMAlert(s.HR))
		prometheus.AddInternalServicesToScrape(cfg, s, settings.DBaaS.Enabled)

		return prometheus.AddScrapeConfigs(svc.l, cfg, tx.Querier, &s)
	})
	if e != nil {
		return nil, e
	}

	b, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "can't marshal VictoriaMetrics configuration file")
	}

	b = append([]byte("# Managed by pmm-managed. DO NOT EDIT.\n---\n"), b...)

	return b, nil
}

// configAndReload saves given VictoriaMetrics configuration to file and reloads VictoriaMetrics.
// If configuration can't be reloaded for some reason, old file is restored, and configuration is reloaded again.
func (svc *Service) configAndReload(ctx context.Context, cfg []byte) error {
	oldCfg, err := ioutil.ReadFile(svc.scrapeConfigPath)
	if err != nil {
		return errors.WithStack(err)
	}

	fi, err := os.Stat(svc.scrapeConfigPath)
	if err != nil {
		return errors.WithStack(err)
	}

	// restore old content and reload in case of error
	var restore bool
	defer func() {
		if restore {
			if err = ioutil.WriteFile(svc.scrapeConfigPath, oldCfg, fi.Mode()); err != nil {
				svc.l.Error(err)
			}
			if err = svc.reload(ctx); err != nil {
				svc.l.Error(err)
			}
		}
	}()

	// write new content to temporary file, check it
	f, err := ioutil.TempFile("", "pmm-managed-config-victoriametrics-")
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
	cmd := exec.CommandContext(ctx, "promtool", args...) //nolint:gosec
	pdeathsig.Set(cmd, unix.SIGKILL)
	b, err := cmd.CombinedOutput()
	if err != nil {
		svc.l.Errorf("%s", b)
		s := string(b)
		if m := checkFailedRE.FindStringSubmatch(s); len(m) == 2 {
			return status.Error(codes.Aborted, m[1])
		}

		return errors.Wrap(err, s)
	}
	svc.l.Debugf("%s", b)

	restore = true
	if err = ioutil.WriteFile(svc.scrapeConfigPath, cfg, fi.Mode()); err != nil {
		return errors.WithStack(err)
	}
	if err = svc.reload(ctx); err != nil {
		return err
	}
	svc.l.Infof("Configuration reloaded.")
	restore = false

	return nil
}

// scrapeConfigForVictoriaMetrics returns scrape config for Victoria Metrics in Prometheus format.
func scrapeConfigForVictoriaMetrics(interval time.Duration) *config.ScrapeConfig {
	return &config.ScrapeConfig{
		JobName:        "victoriametrics",
		ScrapeInterval: config.Duration(interval),
		ScrapeTimeout:  prometheus.ScrapeTimeout(interval),
		MetricsPath:    "/prometheus/metrics",
		ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
			StaticConfigs: []*config.Group{
				{
					Targets: []string{"127.0.0.1:9090"},
					Labels:  map[string]string{"instance": "pmm-server"},
				},
			},
		},
	}
}

// scrapeConfigForVMAlert returns scrape config for VMAlert in Prometheus format.
func scrapeConfigForVMAlert(interval time.Duration) *config.ScrapeConfig {
	return &config.ScrapeConfig{
		JobName:        "vmalert",
		ScrapeInterval: config.Duration(interval),
		ScrapeTimeout:  prometheus.ScrapeTimeout(interval),
		MetricsPath:    "/metrics",
		ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
			StaticConfigs: []*config.Group{
				{
					Targets: []string{"127.0.0.1:8880"},
					Labels:  map[string]string{"instance": "pmm-server"},
				},
			},
		},
	}
}
