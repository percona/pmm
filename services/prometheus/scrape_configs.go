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
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// store scrape configs in Consul under that key
	ConsulKey = "prometheus/scrape_configs"
)

type LabelPair struct {
	Name  string
	Value string
}

type StaticConfig struct {
	Targets []string
	Labels  []LabelPair
}

type RelabelConfig struct {
	TargetLabel string
	Replacement string
}

type BasicAuth struct {
	Username string
	Password string
}

type TLSConfig struct {
	InsecureSkipVerify bool
}

type ScrapeConfig struct {
	JobName        string
	ScrapeInterval string
	ScrapeTimeout  string
	MetricsPath    string
	HonorLabels    bool
	Scheme         string
	BasicAuth      *BasicAuth
	TLSConfig      TLSConfig
	StaticConfigs  []StaticConfig
	RelabelConfigs []RelabelConfig
}

type consulData struct {
	ScrapeConfigs []ScrapeConfig
}

func (svc *Service) getFromConsul() ([]ScrapeConfig, error) {
	b, err := svc.consul.GetKV(ConsulKey)
	if err != nil {
		return nil, err
	}
	if len(b) == 0 {
		return nil, nil
	}
	var cd consulData
	if err = json.Unmarshal(b, &cd); err != nil {
		return nil, errors.WithStack(err)
	}
	return cd.ScrapeConfigs, nil
}

func (svc *Service) putToConsul(scs []ScrapeConfig) error {
	cd := consulData{
		ScrapeConfigs: scs,
	}
	b, err := json.Marshal(cd)
	if err != nil {
		return errors.WithStack(err)
	}
	return svc.consul.PutKV(ConsulKey, b)
}

// ListScrapeConfigs returns all scrape configs.
func (svc *Service) ListScrapeConfigs(ctx context.Context) ([]ScrapeConfig, error) {
	svc.lock.RLock()
	defer svc.lock.RUnlock()

	consulData, err := svc.getFromConsul()
	if err != nil {
		return nil, err
	}
	config, err := svc.loadConfig()
	if err != nil {
		return nil, err
	}

	// return data from Prometheus config to fill default values
	res := make([]ScrapeConfig, len(consulData))
	for i, consulCfg := range consulData {
		var found bool
		for _, configCfg := range config.ScrapeConfigs {
			if consulCfg.JobName == configCfg.JobName {
				res[i] = *convertInternalScrapeConfig(configCfg)
				found = true
				break
			}
		}
		if !found {
			return nil, status.Errorf(codes.FailedPrecondition, "scrape config with job name %q not found in configuration file", consulCfg.JobName)
		}
	}
	return res, nil
}

// GetScrapeConfig returns a scrape config by job name.
// Errors: NotFound(5) if no such scrape config is present.
func (svc *Service) GetScrapeConfig(ctx context.Context, jobName string) (*ScrapeConfig, error) {
	// lock is held by ListScrapeConfigs
	cfgs, err := svc.ListScrapeConfigs(ctx)
	if err != nil {
		return nil, err
	}

	for _, cfg := range cfgs {
		if cfg.JobName == jobName {
			return &cfg, nil
		}
	}
	return nil, status.Errorf(codes.NotFound, "scrape config with job name %q not found", jobName)
}

// CreateScrapeConfig creates a new scrape config.
// Errors: InvalidArgument(3) if some argument is not valid,
// AlreadyExists(6) if scrape config with that job name is already present.
func (svc *Service) CreateScrapeConfig(ctx context.Context, cfg *ScrapeConfig) error {
	svc.lock.Lock()
	defer svc.lock.Unlock()

	consulData, err := svc.getFromConsul()
	if err != nil {
		return err
	}
	config, err := svc.loadConfig()
	if err != nil {
		return err
	}

	updater := &configUpdater{consulData, config.ScrapeConfigs}
	if err = updater.addScrapeConfig(cfg); err != nil {
		return err
	}

	config.ScrapeConfigs = updater.fileData
	if err = svc.saveConfigAndReload(ctx, config); err != nil {
		return err
	}
	return svc.putToConsul(updater.consulData)
}

// SetScrapeConfigs creates new or completely replaces existing scrape configs with a given names.
// Errors: InvalidArgument(3) if some argument is not valid.
func (svc *Service) SetScrapeConfigs(ctx context.Context, useConsul bool, configs ...*ScrapeConfig) error {
	// That method is implemented for RDS and Inventory API. It does not uses Consul.
	// The only reason for useConsul argument existence is to draw attention to that fact, to make it harder to misuse.
	if useConsul {
		panic("Consul is not used")
	}

	svc.lock.Lock()
	defer svc.lock.Unlock()

	config, err := svc.loadConfig()
	if err != nil {
		return err
	}

	for _, cfg := range configs {
		scrapeConfig, err := convertScrapeConfig(cfg)
		if err != nil {
			return err
		}

		var found bool
		for i, sc := range config.ScrapeConfigs {
			if sc.JobName == cfg.JobName {
				config.ScrapeConfigs[i] = scrapeConfig
				found = true
				break
			}
		}
		if !found {
			config.ScrapeConfigs = append(config.ScrapeConfigs, scrapeConfig)
		}
	}

	return svc.saveConfigAndReload(ctx, config)
}

// DeleteScrapeConfig removes existing scrape config by job name.
// Errors: NotFound(5) if no such scrape config is present.
func (svc *Service) DeleteScrapeConfig(ctx context.Context, jobName string) error {
	svc.lock.Lock()
	defer svc.lock.Unlock()

	consulData, err := svc.getFromConsul()
	if err != nil {
		return err
	}
	config, err := svc.loadConfig()
	if err != nil {
		return err
	}

	updater := &configUpdater{consulData, config.ScrapeConfigs}
	if err = updater.removeScrapeConfig(jobName); err != nil {
		return err
	}

	config.ScrapeConfigs = updater.fileData
	if err = svc.saveConfigAndReload(ctx, config); err != nil {
		return err
	}
	return svc.putToConsul(updater.consulData)
}

// AddStaticTargets adds static targets to existing scrape config.
// Errors: NotFound(5) if no such scrape config is present.
func (svc *Service) AddStaticTargets(ctx context.Context, jobName string, targets []string) error {
	svc.lock.Lock()
	defer svc.lock.Unlock()

	consulData, err := svc.getFromConsul()
	if err != nil {
		return err
	}
	config, err := svc.loadConfig()
	if err != nil {
		return err
	}

	updater := &configUpdater{consulData, config.ScrapeConfigs}
	if err = updater.addStaticTargets(jobName, targets); err != nil {
		return err
	}

	config.ScrapeConfigs = updater.fileData
	if err = svc.saveConfigAndReload(ctx, config); err != nil {
		return err
	}
	return svc.putToConsul(updater.consulData)
}

// RemoveStaticTargets removes static targets from existing scrape config.
// Errors: NotFound(5) if no such scrape config is present.
func (svc *Service) RemoveStaticTargets(ctx context.Context, jobName string, targets []string) error {
	svc.lock.Lock()
	defer svc.lock.Unlock()

	consulData, err := svc.getFromConsul()
	if err != nil {
		return err
	}
	config, err := svc.loadConfig()
	if err != nil {
		return err
	}

	updater := &configUpdater{consulData, config.ScrapeConfigs}
	if err = updater.removeStaticTargets(jobName, targets); err != nil {
		return err
	}

	config.ScrapeConfigs = updater.fileData
	if err = svc.saveConfigAndReload(ctx, config); err != nil {
		return err
	}
	return svc.putToConsul(updater.consulData)
}
