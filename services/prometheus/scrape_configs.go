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
	"fmt"

	"github.com/pkg/errors"
	"github.com/prometheus/common/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm-managed/services/prometheus/internal"
)

const (
	// store scrape configs in Consul under that key
	consulKey = "prometheus/scrape_configs"
)

type LabelPair struct {
	Name  string
	Value string
}

type StaticConfig struct {
	Targets []string
	Labels  []LabelPair
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
	Scheme         string
	BasicAuth      *BasicAuth
	TLSConfig      TLSConfig
	StaticConfigs  []StaticConfig
}

type consulData struct {
	ScrapeConfigs []ScrapeConfig
}

func (svc *Service) getFromConsul() ([]ScrapeConfig, error) {
	b, err := svc.consul.GetKV(consulKey)
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
	return svc.consul.PutKV(consulKey, b)
}

func convertInternalScrapeConfig(cfg *internal.ScrapeConfig) *ScrapeConfig {
	var basicAuth *BasicAuth
	if cfg.HTTPClientConfig.BasicAuth != nil {
		basicAuth = &BasicAuth{
			Username: cfg.HTTPClientConfig.BasicAuth.Username,
			Password: cfg.HTTPClientConfig.BasicAuth.Password,
		}
	}

	staticConfigs := make([]StaticConfig, len(cfg.ServiceDiscoveryConfig.StaticConfigs))
	for scI, sc := range cfg.ServiceDiscoveryConfig.StaticConfigs {
		for _, t := range sc.Targets {
			staticConfigs[scI].Targets = append(staticConfigs[scI].Targets, string(t[model.AddressLabel]))
		}
		for n, v := range sc.Labels {
			staticConfigs[scI].Labels = append(staticConfigs[scI].Labels, LabelPair{
				Name:  string(n),
				Value: string(v),
			})
		}
	}

	return &ScrapeConfig{
		JobName:        cfg.JobName,
		ScrapeInterval: cfg.ScrapeInterval.String(),
		ScrapeTimeout:  cfg.ScrapeTimeout.String(),
		MetricsPath:    cfg.MetricsPath,
		Scheme:         cfg.Scheme,
		BasicAuth:      basicAuth,
		TLSConfig: TLSConfig{
			InsecureSkipVerify: cfg.HTTPClientConfig.TLSConfig.InsecureSkipVerify,
		},
		StaticConfigs: staticConfigs,
	}
}

func convertScrapeConfig(cfg *ScrapeConfig) (*internal.ScrapeConfig, error) {
	var err error
	var interval, timeout model.Duration
	if cfg.ScrapeInterval != "" {
		interval, err = model.ParseDuration(cfg.ScrapeInterval)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "interval: %s", err)
		}
	}
	if cfg.ScrapeTimeout != "" {
		timeout, err = model.ParseDuration(cfg.ScrapeTimeout)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "timeout: %s", err)
		}
	}

	var basicAuth *internal.BasicAuth
	if cfg.BasicAuth != nil {
		basicAuth = &internal.BasicAuth{
			Username: cfg.BasicAuth.Username,
			Password: cfg.BasicAuth.Password,
		}
	}

	tg := make([]*internal.TargetGroup, len(cfg.StaticConfigs))
	for i, sc := range cfg.StaticConfigs {
		tg[i] = new(internal.TargetGroup)

		for _, t := range sc.Targets {
			ls := model.LabelSet{model.AddressLabel: model.LabelValue(t)}
			if err = ls.Validate(); err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "static_configs.targets: %s", err)
			}
			tg[i].Targets = append(tg[i].Targets, ls)
		}

		ls := make(model.LabelSet)
		for _, lp := range sc.Labels {
			ls[model.LabelName(lp.Name)] = model.LabelValue(lp.Value)
		}
		if err = ls.Validate(); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "static_configs.labels: %s", err)
		}
		tg[i].Labels = ls
	}

	return &internal.ScrapeConfig{
		JobName:        cfg.JobName,
		ScrapeInterval: interval,
		ScrapeTimeout:  timeout,
		MetricsPath:    cfg.MetricsPath,
		Scheme:         cfg.Scheme,
		HTTPClientConfig: internal.HTTPClientConfig{
			BasicAuth: basicAuth,
			TLSConfig: internal.TLSConfig{
				InsecureSkipVerify: cfg.TLSConfig.InsecureSkipVerify,
			},
		},
		ServiceDiscoveryConfig: internal.ServiceDiscoveryConfig{
			StaticConfigs: tg,
		},
	}, nil
}

// ListScrapeConfigs returns all scrape configs.
func (svc *Service) ListScrapeConfigs(ctx context.Context) ([]ScrapeConfig, error) {
	svc.lock.RLock()
	defer svc.lock.RUnlock()

	return svc.getFromConsul()
}

// GetScrapeConfig returns a scrape config by job name.
// Errors: NotFound(5) if no such scrape config is present.
func (svc *Service) GetScrapeConfig(ctx context.Context, jobName string) (*ScrapeConfig, error) {
	svc.lock.RLock()
	defer svc.lock.RUnlock()

	scs, err := svc.getFromConsul()
	if err != nil {
		return nil, err
	}
	for _, sc := range scs {
		if sc.JobName == jobName {
			return &sc, nil
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

	if err = svc.putToConsul(updater.consulData); err != nil {
		return err
	}
	config.ScrapeConfigs = updater.fileData
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

	if err = svc.putToConsul(updater.consulData); err != nil {
		return err
	}
	config.ScrapeConfigs = updater.fileData
	return svc.saveConfigAndReload(ctx, config)
}

// AddStaticTargets adds static targets to existing scrape config.
// Errors: NotFound(5) if no such scrape config is present.
func (svc *Service) AddStaticTargets(ctx context.Context, jobName string, targets []string) error {
	svc.lock.Lock()
	defer svc.lock.Unlock()

	// check Consul
	scs, err := svc.getFromConsul()
	if err != nil {
		return err
	}
	scsI := -1
	for i, sc := range scs {
		if sc.JobName == jobName {
			scsI = i
			break
		}
	}
	if scsI < 0 {
		return status.Errorf(codes.NotFound, "scrape config with job name %q not found", jobName)
	}

	// check file
	config, err := svc.loadConfig()
	if err != nil {
		return err
	}
	if err = ensureNotBuiltIn(config.ScrapeConfigs, scs, jobName); err != nil {
		return err
	}

	// update configuration
	var sc StaticConfig
	switch len(scs[scsI].StaticConfigs) {
	case 0:
		sc = StaticConfig{}
	case 1:
		sc = scs[scsI].StaticConfigs[0]
	default:
		msg := fmt.Sprintf(
			"scrape config with job name %q has %d static configs, that is not supported yet",
			jobName, len(scs[scsI].StaticConfigs),
		)
		return status.Error(codes.Unimplemented, msg)
	}
	for _, add := range targets {
		var found bool
		for _, t := range sc.Targets {
			if t == add {
				found = true
				break
			}
		}
		if found {
			continue
		}

		sc.Targets = append(sc.Targets, add)
	}

	// update Consul
	scs[scsI].StaticConfigs = []StaticConfig{sc}
	if err = svc.putToConsul(scs); err != nil {
		return err
	}

	// update file
	cfgI, cfg := findScrapeConfigByJobName(config.ScrapeConfigs, jobName)
	if cfg == nil {
		return status.Errorf(codes.FailedPrecondition, "scrape config with job name %q not found in configuration file", jobName)
	}
	scrapeConfig, err := convertScrapeConfig(&scs[scsI])
	config.ScrapeConfigs[cfgI] = scrapeConfig
	return svc.saveConfigAndReload(ctx, config)
}

// RemoveStaticTargets removes static targets from existing scrape config.
// Errors: NotFound(5) if no such scrape config is present.
func (svc *Service) RemoveStaticTargets(ctx context.Context, jobName string, targets []string) error {
	svc.lock.Lock()
	defer svc.lock.Unlock()

	// check Consul
	scs, err := svc.getFromConsul()
	if err != nil {
		return err
	}
	scsI := -1
	for i, sc := range scs {
		if sc.JobName == jobName {
			scsI = i
			break
		}
	}
	if scsI < 0 {
		return status.Errorf(codes.NotFound, "scrape config with job name %q not found", jobName)
	}

	// check file
	config, err := svc.loadConfig()
	if err != nil {
		return err
	}
	if err = ensureNotBuiltIn(config.ScrapeConfigs, scs, jobName); err != nil {
		return err
	}

	// update configuration
	var sc StaticConfig
	switch len(scs[scsI].StaticConfigs) {
	case 0:
		sc = StaticConfig{}
	case 1:
		sc = scs[scsI].StaticConfigs[0]
	default:
		msg := fmt.Sprintf(
			"scrape config with job name %q has %d static configs, that is not supported yet",
			jobName, len(scs[scsI].StaticConfigs),
		)
		return status.Error(codes.Unimplemented, msg)
	}
	for _, remove := range targets {
		for i, t := range sc.Targets {
			if t == remove {
				sc.Targets = append(sc.Targets[:i], sc.Targets[i+1:]...)
				break
			}
		}
	}

	// update Consul
	scs[scsI].StaticConfigs = []StaticConfig{sc}
	if err = svc.putToConsul(scs); err != nil {
		return err
	}

	// update file
	cfgI, cfg := findScrapeConfigByJobName(config.ScrapeConfigs, jobName)
	if cfg == nil {
		return status.Errorf(codes.FailedPrecondition, "scrape config with job name %q not found in configuration file", jobName)
	}
	scrapeConfig, err := convertScrapeConfig(&scs[scsI])
	config.ScrapeConfigs[cfgI] = scrapeConfig
	return svc.saveConfigAndReload(ctx, config)
}
