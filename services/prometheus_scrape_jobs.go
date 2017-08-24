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

package services

import (
	"context"
	"os"

	"github.com/pkg/errors"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/config"
)

type ScrapeJob struct {
	Name          string
	Interval      string
	Timeout       string
	Path          string
	Scheme        string
	StatisTargets []string
}

func convertScrapeConfig(cfg *config.ScrapeConfig) *ScrapeJob {
	targets := make([]string, len(cfg.ServiceDiscoveryConfig.StaticConfigs))
	for i, sc := range cfg.ServiceDiscoveryConfig.StaticConfigs {
		for _, t := range sc.Targets {
			targets[i] = string(t[model.AddressLabel])
		}
	}
	return &ScrapeJob{
		Name:          cfg.JobName,
		Interval:      cfg.ScrapeInterval.String(),
		Timeout:       cfg.ScrapeTimeout.String(),
		Path:          cfg.MetricsPath,
		Scheme:        cfg.Scheme,
		StatisTargets: targets,
	}
}

// ListScrapeJobs returns all scrape jobs.
func (p *Prometheus) ListScrapeJobs(ctx context.Context) ([]ScrapeJob, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	cfg, err := p.loadConfig()
	if err != nil {
		return nil, err
	}

	res := make([]ScrapeJob, len(cfg.ScrapeConfigs))
	for i, sc := range cfg.ScrapeConfigs {
		res[i] = *convertScrapeConfig(sc)
	}
	return res, nil
}

// GetScrapeJob return scrape job by name, or error if no such scrape job is present.
func (p *Prometheus) GetScrapeJob(ctx context.Context, name string) (*ScrapeJob, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	cfg, err := p.loadConfig()
	if err != nil {
		return nil, err
	}

	for _, sc := range cfg.ScrapeConfigs {
		if sc.JobName == name {
			return convertScrapeConfig(sc), nil
		}
	}
	return nil, errors.WithStack(os.ErrNotExist)
}

// PutScrapeJob creates or replaces existing scrape job.
func (p *Prometheus) PutScrapeJob(ctx context.Context, job *ScrapeJob) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	cfg, err := p.loadConfig()
	if err != nil {
		return err
	}

	var interval, timeout model.Duration
	if job.Interval != "" {
		interval, err = model.ParseDuration(job.Interval)
		if err != nil {
			return err
		}
	}
	if job.Timeout != "" {
		timeout, err = model.ParseDuration(job.Timeout)
		if err != nil {
			return err
		}
	}

	tg := make([]*config.TargetGroup, len(job.StatisTargets))
	for i, t := range job.StatisTargets {
		tg[i] = &config.TargetGroup{
			Targets: []model.LabelSet{{model.AddressLabel: model.LabelValue(t)}},
		}
	}
	scrapeConfig := &config.ScrapeConfig{
		JobName:        job.Name,
		ScrapeInterval: interval,
		ScrapeTimeout:  timeout,
		MetricsPath:    job.Path,
		Scheme:         job.Scheme,
		ServiceDiscoveryConfig: config.ServiceDiscoveryConfig{
			StaticConfigs: tg,
		},
	}

	var found bool
	for i, sc := range cfg.ScrapeConfigs {
		if sc.JobName == job.Name {
			cfg.ScrapeConfigs[i] = scrapeConfig
			found = true
			break
		}
	}
	if !found {
		cfg.ScrapeConfigs = append(cfg.ScrapeConfigs, scrapeConfig)
	}

	if err = p.saveConfig(cfg); err != nil {
		return err
	}
	return p.reload()
}

// DeleteScrapeJob removes existing scrape job by name, or error if no such scrape job is present.
func (p *Prometheus) DeleteScrapeJob(ctx context.Context, name string) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	cfg, err := p.loadConfig()
	if err != nil {
		return err
	}

	var found bool
	for i, j := range cfg.ScrapeConfigs {
		if j.JobName == name {
			cfg.ScrapeConfigs = append(cfg.ScrapeConfigs[:i], cfg.ScrapeConfigs[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		return errors.WithStack(os.ErrNotExist)
	}

	if err = p.saveConfig(cfg); err != nil {
		return err
	}
	return p.reload()
}
