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
	"regexp"
	"sync"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/yaml.v2"

	"github.com/percona/pmm-managed/services/consul"
	"github.com/percona/pmm-managed/services/prometheus/internal"
	"github.com/percona/pmm-managed/utils/logger"
)

var checkFailedRE = regexp.MustCompile(`FAILED: (.+)\n`)

// Service is responsible for interactions with Prometheus.
// It assumes the following:
//   * Prometheus API is accessible;
//   * Prometheus configuration and rule files are accessible;
//   * promtool is available.
type Service struct {
	ConfigPath   string
	baseURL      *url.URL
	client       *http.Client
	promtoolPath string
	consul       *consul.Client
	lock         sync.RWMutex // for Prometheus configuration file and, by extension, for most methods
}

func NewService(config string, baseURL string, promtool string, consul *consul.Client) (*Service, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &Service{
		ConfigPath:   config,
		baseURL:      u,
		client:       new(http.Client),
		promtoolPath: promtool,
		consul:       consul,
	}, nil
}

// loadConfig loads current Prometheus configuration from file.
func (svc *Service) loadConfig() (*internal.Config, error) {
	cfg, err := internal.LoadFile(svc.ConfigPath)
	if err != nil {
		return nil, errors.Wrap(err, "can't load Prometheus configuration file")
	}
	return cfg, nil
}

// saveConfigAndReload saves given Prometheus configuration to file and reloads Prometheus.
// If configuration can't be reloaded for some reason, old file is restored, and configuration is reloaded again.
func (svc *Service) saveConfigAndReload(ctx context.Context, cfg *internal.Config) error {
	// read existing content
	old, err := ioutil.ReadFile(svc.ConfigPath)
	if err != nil {
		return errors.WithStack(err)
	}
	fi, err := os.Stat(svc.ConfigPath)
	if err != nil {
		return errors.WithStack(err)
	}

	// restore old content and reload in case of error
	var restore bool
	defer func() {
		if restore {
			if err = ioutil.WriteFile(svc.ConfigPath, old, fi.Mode()); err != nil {
				logger.Get(ctx).Error(err)
			}
			if err = svc.reload(); err != nil {
				logger.Get(ctx).Error(err)
			}
		}
	}()

	// marshal new content
	new, err := yaml.Marshal(cfg)
	if err != nil {
		return errors.Wrap(err, "can't marshal Prometheus configuration file")
	}
	new = append([]byte("# Managed by pmm-managed. DO NOT EDIT.\n---\n"), new...)

	// write new content to temporary file, check it
	f, err := ioutil.TempFile("", "pmm-managed-config-")
	if err != nil {
		return errors.WithStack(err)
	}
	if _, err = f.Write(new); err != nil {
		return errors.WithStack(err)
	}
	defer func() {
		f.Close()
		os.Remove(f.Name())
	}()
	b, err := exec.Command(svc.promtoolPath, "check-config", f.Name()).CombinedOutput()
	if err != nil {
		logger.Get(ctx).Errorf("%s", b)

		// return typed error if possible
		s := string(b)
		if m := checkFailedRE.FindStringSubmatch(s); len(m) == 2 {
			return status.Error(codes.Aborted, m[1])
		}
		return errors.Wrap(err, s)
	}
	logger.Get(ctx).Debugf("%s", b)

	// write to permanent location and reload
	restore = true
	if err = ioutil.WriteFile(svc.ConfigPath, new, fi.Mode()); err != nil {
		return errors.WithStack(err)
	}
	if err = svc.reload(); err != nil {
		return err
	}
	restore = false
	return nil
}

// reload causes Prometheus to reload configuration.
func (svc *Service) reload() error {
	u := *svc.baseURL
	u.Path = path.Join(u.Path, "-", "reload")
	resp, err := svc.client.Post(u.String(), "", nil)
	if err != nil {
		return errors.WithStack(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return nil
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.WithStack(err)
	}
	return errors.Errorf("%d: %s", resp.StatusCode, b)
}

// Check updates Prometehus configuration using information from Consul KV.
// (During PMM update prometheus.yml is overwritten, but Consul data directory is kept.)
// It returns error if configuration is not right or Prometheus is not available.
func (svc *Service) Check(ctx context.Context) error {
	l := logger.Get(ctx)

	config, err := svc.loadConfig()
	if err != nil {
		return err
	}

	if svc.baseURL == nil {
		return errors.New("URL is not set")
	}
	u := *svc.baseURL
	u.Path = path.Join(u.Path, "version")
	resp, err := svc.client.Get(u.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	l.Debugf("Prometheus: %s", b)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return errors.Errorf("expected 200, got %d", resp.StatusCode)
	}

	b, err = exec.Command(svc.promtoolPath, "version").CombinedOutput()
	if err != nil {
		return errors.Wrap(err, string(b))
	}
	l.Debugf("%s", b)

	scs, err := svc.getFromConsul()
	if err != nil {
		return err
	}
	var changed bool
	for _, sc := range scs {
		var found bool
		for _, configSC := range config.ScrapeConfigs {
			if configSC.JobName == sc.JobName {
				found = true
				break
			}
		}

		if !found {
			scrapeConfig, err := convertScrapeConfig(&sc)
			if err != nil {
				return err
			}
			config.ScrapeConfigs = append(config.ScrapeConfigs, scrapeConfig)
			l.Infof("Scrape configuration restored from Consul: %+v", sc)
			changed = true
		}
	}

	if changed {
		l.Info("Prometheus configuration updated.")
		return svc.saveConfigAndReload(ctx, config)
	}
	l.Info("Prometheus configuration not changed.")
	return nil
}
