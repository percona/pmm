// Copyright (C) 2023 Percona LLC
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
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"regexp"
	"time"

	"github.com/AlekSi/pointer"
	config "github.com/percona/promconfig"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/yaml.v3"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/dir"
	"github.com/percona/pmm/utils/pdeathsig"
)

const (
	updateBatchDelay           = time.Second
	configurationUpdateTimeout = 3 * time.Second

	victoriametricsDir     = "/srv/victoriametrics"
	victoriametricsDataDir = "/srv/victoriametrics/data"
	dirPerm                = os.FileMode(0o775)
)

var checkFailedRE = regexp.MustCompile(`(?s)cannot unmarshal data: (.+)`)

// Service is responsible for interactions with VictoriaMetrics.
type Service struct {
	scrapeConfigPath string
	db               *reform.DB
	baseURL          *url.URL
	client           *http.Client

	params *models.VictoriaMetricsParams

	l        *logrus.Entry
	reloadCh chan struct{}
}

// NewVictoriaMetrics creates new VictoriaMetrics service.
func NewVictoriaMetrics(scrapeConfigPath string, db *reform.DB, params *models.VictoriaMetricsParams) (*Service, error) {
	u, err := url.Parse(params.URL())
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &Service{
		scrapeConfigPath: scrapeConfigPath,
		db:               db,
		baseURL:          u,
		client:           &http.Client{}, // TODO instrument with utils/irt; see vmalert package https://jira.percona.com/browse/PMM-7229
		params:           params,
		l:                logrus.WithField("component", "victoriametrics"),
		reloadCh:         make(chan struct{}, 1),
	}, nil
}

// Run runs VictoriaMetrics configuration update loop until ctx is canceled.
func (svc *Service) Run(ctx context.Context) {
	// If you change this and related methods,
	// please do similar changes in vmalert package.

	svc.l.Info("Starting...")
	defer svc.l.Info("Done.")

	if err := dir.CreateDataDir(victoriametricsDir, "pmm", "pmm", dirPerm); err != nil {
		svc.l.Error(err)
	}
	if err := dir.CreateDataDir(victoriametricsDataDir, "pmm", "pmm", dirPerm); err != nil {
		svc.l.Error(err)
	}

	// reloadCh, configuration update loop, and RequestConfigurationUpdate method ensure that configuration
	// is reloaded when requested, but several requests are batched together to avoid too often reloads.
	// That allows the caller to just call RequestConfigurationUpdate when it seems fit.
	if cap(svc.reloadCh) != 1 {
		panic("reloadCh should have capacity 1")
	}

	for {
		select {
		case <-ctx.Done():
			return

		case <-svc.reloadCh:
			// batch several update requests together by delaying the first one
			sleepCtx, sleepCancel := context.WithTimeout(ctx, updateBatchDelay)
			<-sleepCtx.Done()
			sleepCancel()

			if ctx.Err() != nil {
				return
			}

			nCtx, cancel := context.WithTimeout(ctx, configurationUpdateTimeout)
			if err := svc.updateConfiguration(nCtx); err != nil {
				svc.l.Errorf("Failed to update configuration, will retry: %+v.", err)
				svc.RequestConfigurationUpdate()
			}
			cancel()
		}
	}
}

// RequestConfigurationUpdate requests VictoriaMetrics configuration update.
func (svc *Service) RequestConfigurationUpdate() {
	select {
	case svc.reloadCh <- struct{}{}:
	default:
	}
}

// updateConfiguration updates VictoriaMetrics configuration.
func (svc *Service) updateConfiguration(ctx context.Context) error {
	if svc.params.ExternalVM() {
		return nil
	}
	start := time.Now()
	defer func() {
		if dur := time.Since(start); dur > time.Second {
			svc.l.Warnf("updateConfiguration took %s.", dur)
		}
	}()

	cfg, err := svc.buildVMConfig()
	if err != nil {
		return err
	}

	return svc.configAndReload(ctx, cfg)
}

func (svc *Service) buildVMConfig() ([]byte, error) {
	base := svc.loadBaseConfig()
	return svc.marshalConfig(base)
}

// reload asks VictoriaMetrics to reload configuration.
func (svc *Service) reload(ctx context.Context) error {
	u := *svc.baseURL
	u.Path = path.Join(u.Path, "-", "reload")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return errors.WithStack(err)
	}
	resp, err := svc.client.Do(req)
	if err != nil {
		return errors.WithStack(err)
	}
	defer resp.Body.Close() //nolint:errcheck,gosec,nolintlint

	b, err := io.ReadAll(resp.Body)
	svc.l.Debugf("VM reload: %s", b)
	if err != nil {
		return errors.WithStack(err)
	}

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return errors.Errorf("expected 204, got %d", resp.StatusCode)
	}
	return nil
}

// loadBaseConfig returns parsed base configuration file, or empty configuration on error.
func (svc *Service) loadBaseConfig() *config.Config {
	buf, err := os.ReadFile(svc.params.BaseConfigPath)
	if err != nil {
		if !os.IsNotExist(err) {
			svc.l.Errorf("Failed to load base VictoriaMetrics config %s: %s", svc.params.BaseConfigPath, err)
		}

		return &config.Config{}
	}

	var cfg config.Config
	if err := yaml.Unmarshal(buf, &cfg); err != nil {
		svc.l.Errorf("Failed to parse base VictoriaMetrics config %s: %s.", svc.params.BaseConfigPath, err)

		return &config.Config{}
	}

	return &cfg
}

// marshalConfig marshals VictoriaMetrics configuration.
func (svc *Service) marshalConfig(base *config.Config) ([]byte, error) {
	cfg := base
	if err := svc.populateConfig(cfg); err != nil {
		return nil, err
	}

	b, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "can't marshal VictoriaMetrics configuration file")
	}

	b = append([]byte("# Managed by pmm-managed. DO NOT EDIT.\n---\n"), b...)

	return b, nil
}

// validateConfig validates given configuration with `victoriametrics -dryRun`.
func (svc *Service) validateConfig(ctx context.Context, cfg []byte) error {
	f, err := os.CreateTemp("", "pmm-managed-config-victoriametrics-")
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

	args := []string{"-dryRun", "-promscrape.config", f.Name()}
	cmd := exec.CommandContext(ctx, "victoriametrics", args...) //nolint:gosec
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

	args = append(args, "-promscrape.config.strictParse=true")
	cmd = exec.CommandContext(ctx, "victoriametrics", args...) //nolint:gosec
	pdeathsig.Set(cmd, unix.SIGKILL)

	b, err = cmd.CombinedOutput()
	if err != nil {
		s := string(b)
		if m := checkFailedRE.FindStringSubmatch(s); len(m) == 2 {
			svc.l.Warnf("VictoriaMetrics scrape configuration contains unsupported params: %s", m[1])
		} else {
			svc.l.Warnf("VictoriaMetrics scrape configuration contains unsupported params: %s", b)
		}
	}
	svc.l.Debugf("%s", b)

	return nil
}

// configAndReload saves given VictoriaMetrics configuration to file and reloads VictoriaMetrics.
// If configuration can't be reloaded for some reason, old file is restored, and configuration is reloaded again.
func (svc *Service) configAndReload(ctx context.Context, b []byte) error {
	oldCfg, err := os.ReadFile(svc.scrapeConfigPath)
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
			if err = os.WriteFile(svc.scrapeConfigPath, oldCfg, fi.Mode()); err != nil {
				svc.l.Error(err)
			}
			if err = svc.reload(ctx); err != nil {
				svc.l.Error(err)
			}
		}
	}()

	if err = svc.validateConfig(ctx, b); err != nil {
		return err
	}

	restore = true
	if err = os.WriteFile(svc.scrapeConfigPath, b, fi.Mode()); err != nil {
		return errors.WithStack(err)
	}
	if err = svc.reload(ctx); err != nil {
		return err
	}
	svc.l.Infof("Configuration reloaded.")
	restore = false

	return nil
}

// populateConfig adds configuration from the database to cfg.
func (svc *Service) populateConfig(cfg *config.Config) error {
	return svc.db.InTransaction(func(tx *reform.TX) error {
		settings, err := models.GetSettings(tx)
		if err != nil {
			return err
		}
		resolutions := settings.MetricsResolutions
		if cfg.GlobalConfig.ScrapeInterval == 0 {
			cfg.GlobalConfig.ScrapeInterval = config.Duration(resolutions.LR)
		}
		if cfg.GlobalConfig.ScrapeTimeout == 0 {
			cfg.GlobalConfig.ScrapeTimeout = ScrapeTimeout(resolutions.LR)
		}
		cfg.ScrapeConfigs = append(cfg.ScrapeConfigs, scrapeConfigForVictoriaMetrics(svc.l, resolutions.HR, svc.params))
		if svc.params.ExternalVM() {
			cfg.ScrapeConfigs = append(cfg.ScrapeConfigs, scrapeConfigForInternalVMAgent(resolutions.HR, svc.baseURL.Host))
		}
		cfg.ScrapeConfigs = append(cfg.ScrapeConfigs, scrapeConfigForVMAlert(resolutions.HR))
		AddInternalServicesToScrape(cfg, resolutions)
		if pointer.GetBool(settings.Nomad.Enabled) {
			cfg.ScrapeConfigs = append(cfg.ScrapeConfigs,
				scrapeConfigForNomadServer(resolutions.MR))
		}
		return AddScrapeConfigs(svc.l, cfg, tx.Querier, &resolutions, nil, false)
	})
}

// scrapeConfigForVictoriaMetrics returns scrape config for Victoria Metrics in Prometheus format.
func scrapeConfigForVictoriaMetrics(l *logrus.Entry, interval time.Duration, vmParams *models.VictoriaMetricsParams) *config.ScrapeConfig {
	target, err := vmParams.URLFor("metrics")
	if err != nil {
		l.Errorf("couldn't parse relative path to victoria metrics: %q", err)
		return nil
	}

	return &config.ScrapeConfig{
		JobName:        "victoriametrics",
		ScrapeInterval: config.Duration(interval),
		ScrapeTimeout:  ScrapeTimeout(interval),
		MetricsPath:    target.Path,
		ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
			StaticConfigs: []*config.Group{
				{
					Targets: []string{target.Host},
					Labels:  map[string]string{"instance": models.PMMServerAgentID},
				},
			},
		},
	}
}

// scrapeConfigForInternalVMAgent returns scrape config for internal VM Agent in Prometheus format.
func scrapeConfigForInternalVMAgent(interval time.Duration, target string) *config.ScrapeConfig {
	return &config.ScrapeConfig{
		JobName:        "vmagent",
		ScrapeInterval: config.Duration(interval),
		ScrapeTimeout:  ScrapeTimeout(interval),
		MetricsPath:    "/metrics",
		ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
			StaticConfigs: []*config.Group{
				{
					Targets: []string{target},
					Labels:  map[string]string{"instance": models.PMMServerAgentID},
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
		ScrapeTimeout:  ScrapeTimeout(interval),
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

// BuildScrapeConfigForVMAgent builds scrape configuration for given pmm-agent.
func (svc *Service) BuildScrapeConfigForVMAgent(pmmAgentID string) ([]byte, error) {
	if pmmAgentID == models.PMMServerAgentID {
		return svc.buildVMConfig()
	}
	var cfg config.Config
	e := svc.db.InTransaction(func(tx *reform.TX) error {
		settings, err := models.GetSettings(tx)
		if err != nil {
			return err
		}
		s := settings.MetricsResolutions
		return AddScrapeConfigs(svc.l, &cfg, tx.Querier, &s, pointer.ToString(pmmAgentID), true)
	})
	if e != nil {
		return nil, e
	}

	return yaml.Marshal(cfg)
}

// IsReady verifies that VictoriaMetrics works.
func (svc *Service) IsReady(ctx context.Context) error {
	if svc.params.ExternalVM() {
		svc.l.Debugf("External VM is used, VM healthcheck is skipped")
		return nil
	}
	u := *svc.baseURL
	u.Path = path.Join(u.Path, "health")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return errors.WithStack(err)
	}
	resp, err := svc.client.Do(req)
	if err != nil {
		return errors.WithStack(err)
	}
	defer resp.Body.Close() //nolint:gosec,errcheck,nolintlint

	b, err := io.ReadAll(resp.Body)
	svc.l.Debugf("VM health: %s", b)
	if err != nil {
		return errors.WithStack(err)
	}
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("expected 200, got %d", resp.StatusCode)
	}

	return nil
}
