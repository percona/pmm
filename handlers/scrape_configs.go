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

package handlers

import (
	"golang.org/x/net/context"

	"github.com/percona/pmm-managed/api"
	"github.com/percona/pmm-managed/services/prometheus"
)

type ScrapeConfigsServer struct {
	Prometheus *prometheus.Service
}

func convertServiceScrapeConfig(cfg *prometheus.ScrapeConfig) *api.ScrapeConfig {
	var basicAuth *api.BasicAuth
	if cfg.BasicAuth != nil {
		basicAuth = &api.BasicAuth{
			Username: cfg.BasicAuth.Username,
			Password: cfg.BasicAuth.Password,
		}
	}

	staticConfigs := make([]*api.StaticConfig, len(cfg.StaticConfigs))
	for i, sc := range cfg.StaticConfigs {
		labels := make([]*api.LabelPair, len(sc.Labels))
		for j, l := range sc.Labels {
			labels[j] = &api.LabelPair{
				Name:  l.Name,
				Value: l.Value,
			}
		}
		staticConfigs[i] = &api.StaticConfig{
			Targets: sc.Targets,
			Labels:  labels,
		}
	}

	return &api.ScrapeConfig{
		JobName:        cfg.JobName,
		ScrapeInterval: cfg.ScrapeInterval,
		ScrapeTimeout:  cfg.ScrapeTimeout,
		MetricsPath:    cfg.MetricsPath,
		Scheme:         cfg.Scheme,
		BasicAuth:      basicAuth,
		TlsConfig: &api.TLSConfig{
			InsecureSkipVerify: cfg.TLSConfig.InsecureSkipVerify,
		},
		StaticConfigs: staticConfigs,
	}
}

// TODO validate
func convertAPIScrapeConfig(cfg *api.ScrapeConfig) (*prometheus.ScrapeConfig, error) {
	var basicAuth *prometheus.BasicAuth
	if cfg.BasicAuth != nil {
		basicAuth = &prometheus.BasicAuth{
			Username: cfg.BasicAuth.Username,
			Password: cfg.BasicAuth.Password,
		}
	}

	staticConfigs := make([]prometheus.StaticConfig, len(cfg.StaticConfigs))
	for i, sc := range cfg.StaticConfigs {
		labels := make([]prometheus.LabelPair, len(sc.Labels))
		for j, l := range sc.Labels {
			labels[j] = prometheus.LabelPair{
				Name:  l.Name,
				Value: l.Value,
			}
		}
		staticConfigs[i] = prometheus.StaticConfig{
			Targets: sc.Targets,
			Labels:  labels,
		}
	}

	return &prometheus.ScrapeConfig{
		JobName:        cfg.JobName,
		ScrapeInterval: cfg.ScrapeInterval,
		ScrapeTimeout:  cfg.ScrapeTimeout,
		MetricsPath:    cfg.MetricsPath,
		Scheme:         cfg.Scheme,
		BasicAuth:      basicAuth,
		TLSConfig: prometheus.TLSConfig{
			InsecureSkipVerify: cfg.GetTlsConfig().GetInsecureSkipVerify(),
		},
		StaticConfigs: staticConfigs,
	}, nil
}

func convertServiceScrapeTargetHealth(health *prometheus.ScrapeTargetHealth) *api.ScrapeTargetHealth {
	h := api.ScrapeTargetHealth_UNKNOWN
	switch health.Health {
	case prometheus.HealthDown:
		h = api.ScrapeTargetHealth_DOWN
	case prometheus.HealthUp:
		h = api.ScrapeTargetHealth_UP
	}
	return &api.ScrapeTargetHealth{
		JobName:  health.JobName,
		Job:      health.Job,
		Target:   health.Target,
		Instance: health.Instance,
		Health:   h,
	}
}

// List returns all scrape configs.
func (s *ScrapeConfigsServer) List(ctx context.Context, req *api.ScrapeConfigsListRequest) (*api.ScrapeConfigsListResponse, error) {
	cfgs, health, err := s.Prometheus.ListScrapeConfigs(ctx)
	if err != nil {
		return nil, err
	}
	res := &api.ScrapeConfigsListResponse{
		ScrapeConfigs:       make([]*api.ScrapeConfig, len(cfgs)),
		ScrapeTargetsHealth: make([]*api.ScrapeTargetHealth, len(health)),
	}
	for i, cfg := range cfgs {
		res.ScrapeConfigs[i] = convertServiceScrapeConfig(&cfg)
	}
	for i, h := range health {
		res.ScrapeTargetsHealth[i] = convertServiceScrapeTargetHealth(&h)
	}
	return res, nil
}

// Get returns a scrape config by job name.
// Errors: NotFound(5) if no such scrape config is present.
func (s *ScrapeConfigsServer) Get(ctx context.Context, req *api.ScrapeConfigsGetRequest) (*api.ScrapeConfigsGetResponse, error) {
	cfg, health, err := s.Prometheus.GetScrapeConfig(ctx, req.JobName)
	if err != nil {
		return nil, err
	}
	res := &api.ScrapeConfigsGetResponse{
		ScrapeConfig:        convertServiceScrapeConfig(cfg),
		ScrapeTargetsHealth: make([]*api.ScrapeTargetHealth, len(health)),
	}
	for i, h := range health {
		res.ScrapeTargetsHealth[i] = convertServiceScrapeTargetHealth(&h)
	}
	return res, nil
}

// Create creates a new scrape config.
// Errors: InvalidArgument(3) if some argument is not valid,
// AlreadyExists(6) if scrape config with that job name is already present.
func (s *ScrapeConfigsServer) Create(ctx context.Context, req *api.ScrapeConfigsCreateRequest) (*api.ScrapeConfigsCreateResponse, error) {
	cfg, err := convertAPIScrapeConfig(req.ScrapeConfig)
	if err != nil {
		return nil, err
	}
	if err := s.Prometheus.CreateScrapeConfig(ctx, cfg, req.CheckReachability); err != nil {
		return nil, err
	}
	return &api.ScrapeConfigsCreateResponse{}, nil
}

// Update updates existing scrape config by job name.
// Errors: InvalidArgument(3) if some argument is not valid,
// NotFound(5) if no such scrape config is present,
// FailedPrecondition(9) if reachability check was requested and some scrape target can't be reached.
func (s *ScrapeConfigsServer) Update(ctx context.Context, req *api.ScrapeConfigsUpdateRequest) (*api.ScrapeConfigsUpdateResponse, error) {
	cfg, err := convertAPIScrapeConfig(req.ScrapeConfig)
	if err != nil {
		return nil, err
	}
	if err := s.Prometheus.UpdateScrapeConfig(ctx, cfg, req.CheckReachability); err != nil {
		return nil, err
	}
	return &api.ScrapeConfigsUpdateResponse{}, nil
}

// Delete removes existing scrape config by job name.
// Errors: NotFound(5) if no such scrape config is present.
func (s *ScrapeConfigsServer) Delete(ctx context.Context, req *api.ScrapeConfigsDeleteRequest) (*api.ScrapeConfigsDeleteResponse, error) {
	if err := s.Prometheus.DeleteScrapeConfig(ctx, req.JobName); err != nil {
		return nil, err
	}
	return &api.ScrapeConfigsDeleteResponse{}, nil
}

// check interfaces
var (
	_ api.ScrapeConfigsServer = (*ScrapeConfigsServer)(nil)
)
