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
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path"
	"sort"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/common/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm-managed/services/prometheus/internal"
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

type Health int32

const (
	HealthUnknown Health = 0
	HealthDown    Health = 1
	HealthUp      Health = 2
)

// ScrapeTargetHealth represents Prometheus scrape target health: unknown, down, or up.
type ScrapeTargetHealth struct {
	JobName  string
	Job      string
	Target   string
	Instance string
	Health   Health
}

// ScrapeTargetReachability represents a single reachability check result.
type ScrapeTargetReachability struct {
	Target string
	Error  string
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

// getTargetsHealth gets all targets from Prometheus and converts response to job -> instance -> health map.
func (svc *Service) getTargetsHealth(ctx context.Context) (map[string]map[string]Health, error) {
	u := *svc.baseURL
	u.Path = path.Join(u.Path, "api/v1/targets")
	resp, err := svc.client.Get(u.String())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer resp.Body.Close()

	// Copied from vendor/github.com/prometheus/prometheus/web/api/v1/api.go to avoid a ton of dependencies
	type target struct {
		DiscoveredLabels model.LabelSet `json:"discoveredLabels"`
		Labels           model.LabelSet `json:"labels"`
		ScrapeURL        string         `json:"scrapeUrl"`
		LastError        string         `json:"lastError"`
		LastScrape       time.Time      `json:"lastScrape"`
		Health           string         `json:"health"` // 	"unknown", "up", or "down"
	}
	type result struct {
		Status string `json:"status"`
		Data   struct {
			ActiveTargets []target `json:"activeTargets"`
		} `json:"data"`
	}
	var res result
	if err = json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, errors.WithStack(err)
	}

	health := make(map[string]map[string]Health)
	for _, target := range res.Data.ActiveTargets {
		job := string(target.Labels["job"])
		instance := string(target.Labels["instance"])
		if health[job] == nil {
			health[job] = make(map[string]Health)
		}

		health[job][instance] = HealthUnknown
		switch target.Health {
		case "down":
			health[job][instance] = HealthDown
		case "up":
			health[job][instance] = HealthUp
		}
	}
	return health, nil
}

// checkReachability checks that given targets can be reached from PMM Server.
// reachabilityCh is closed when this method returns.
func (svc *Service) checkReachability(ctx context.Context, cfg *ScrapeConfig, targets []string, reachabilityCh chan<- ScrapeTargetReachability) {
	// use ephemeral transport with small timeouts and disabled HTTP keep-alive
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			DualStack: true,
		}).DialContext,
		TLSHandshakeTimeout: 5 * time.Second,
		DisableKeepAlives:   true,
		MaxIdleConns:        1,
		MaxIdleConnsPerHost: 1,
		IdleConnTimeout:     time.Second,

		// Prometheus does not uses HTTP/2, so we disable it too
		TLSNextProto: map[string]func(string, *tls.Conn) http.RoundTripper{},
	}
	if cfg.TLSConfig.InsecureSkipVerify {
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}
	defer transport.CloseIdleConnections()

	defer close(reachabilityCh)

	// create client with specified scrape timeout
	var scrapeTimeout model.Duration
	var err error
	if cfg.ScrapeTimeout == "" {
		scrapeTimeout = internal.DefaultGlobalConfig.ScrapeTimeout
	} else {
		scrapeTimeout, err = model.ParseDuration(cfg.ScrapeTimeout)
		if err != nil {
			reachabilityCh <- ScrapeTargetReachability{"", err.Error()}
			return
		}
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(scrapeTimeout),
	}

	var wg sync.WaitGroup
	for _, target := range targets {
		wg.Add(1)
		go func(target string) {
			defer wg.Done()

			// fill request default values
			u := &url.URL{
				Scheme: cfg.Scheme,
				Host:   target,
				Path:   cfg.MetricsPath,
			}
			if cfg.Scheme == "" {
				u.Scheme = internal.DefaultScrapeConfig.Scheme
			}
			if cfg.MetricsPath == "" {
				u.Path = internal.DefaultScrapeConfig.MetricsPath
			}
			if cfg.BasicAuth != nil {
				u.User = url.UserPassword(cfg.BasicAuth.Username, cfg.BasicAuth.Password)
			}
			req, err := http.NewRequest("GET", u.String(), nil)
			if err != nil {
				reachabilityCh <- ScrapeTargetReachability{target, err.Error()}
				return
			}
			req = req.WithContext(ctx)

			// only HTTP 200 is ok
			resp, err := client.Do(req)
			if err != nil {
				reachabilityCh <- ScrapeTargetReachability{target, err.Error()}
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				reachabilityCh <- ScrapeTargetReachability{target, fmt.Sprintf("unexpected response status code %d", resp.StatusCode)}
			}
			reachabilityCh <- ScrapeTargetReachability{target, ""}
		}(target)
	}
	wg.Wait()
}

// jobInstanceValues returns "job" and "instance" label values for given ScrapeConfig and static target.
// Relabeling is considered.
func jobInstanceValues(cfg *ScrapeConfig, target string) (job string, instance string) {
	job = cfg.JobName
	instance = target

	for _, rl := range cfg.RelabelConfigs {
		if rl.TargetLabel == "job" {
			job = rl.Replacement
		}
		if rl.TargetLabel == "instance" {
			instance = rl.Replacement
		}
	}

	for _, sc := range cfg.StaticConfigs {
		for _, t := range sc.Targets {
			if t == target {
				for _, lp := range sc.Labels {
					if lp.Name == "job" {
						job = lp.Value
					}
					if lp.Name == "instance" {
						instance = lp.Value
					}
				}
				return
			}
		}
	}
	return
}

// ListScrapeConfigs returns all scrape configs.
func (svc *Service) ListScrapeConfigs(ctx context.Context) ([]ScrapeConfig, []ScrapeTargetHealth, error) {
	svc.lock.RLock()
	defer svc.lock.RUnlock()

	// start getting targets health from Prometheus early
	type targetsHealth struct {
		data map[string]map[string]Health
		err  error
	}
	targetsHealthCh := make(chan targetsHealth)
	go func() {
		d, e := svc.getTargetsHealth(ctx)
		targetsHealthCh <- targetsHealth{d, e}
	}()

	consulData, err := svc.getFromConsul()
	if err != nil {
		return nil, nil, err
	}
	config, err := svc.loadConfig()
	if err != nil {
		return nil, nil, err
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
			err = status.Errorf(codes.FailedPrecondition, "scrape config with job name %q not found in configuration file", consulCfg.JobName)
			return nil, nil, err
		}
	}

	health := <-targetsHealthCh
	if health.err != nil {
		return nil, nil, health.err
	}

	// return only health of managed scrape targets, not all of them
	var healthRes []ScrapeTargetHealth
	for _, cfg := range res {
		for _, sc := range cfg.StaticConfigs {
			for _, t := range sc.Targets {
				// default health is unknown
				jobValue, instanceValue := jobInstanceValues(&cfg, t)
				st := ScrapeTargetHealth{
					JobName:  cfg.JobName,
					Job:      jobValue,
					Target:   t,
					Instance: instanceValue,
					Health:   HealthUnknown,
				}

				// check we know real health from Prometheus
				for job, instances := range health.data {
					if jobValue != job {
						continue
					}
					for instance, h := range instances {
						if instanceValue != instance {
							continue
						}
						st.Health = h
					}
				}

				healthRes = append(healthRes, st)
			}
		}
	}

	sort.Slice(healthRes, func(i, j int) bool {
		ri, rj := healthRes[i], healthRes[j]
		if ri.JobName != rj.JobName {
			return ri.JobName < rj.JobName
		}
		return ri.Instance < rj.Instance
	})

	return res, healthRes, nil
}

// GetScrapeConfig returns a scrape config by job name.
// Errors: NotFound(5) if no such scrape config is present.
func (svc *Service) GetScrapeConfig(ctx context.Context, jobName string) (*ScrapeConfig, []ScrapeTargetHealth, error) {
	// lock is held by ListScrapeConfigs
	cfgs, health, err := svc.ListScrapeConfigs(ctx)
	if err != nil {
		return nil, nil, err
	}

	for _, cfg := range cfgs {
		if cfg.JobName == jobName {
			var healthRes []ScrapeTargetHealth
			for _, h := range health {
				if h.JobName == jobName {
					healthRes = append(healthRes, h)
				}
			}
			return &cfg, healthRes, nil
		}
	}
	return nil, nil, status.Errorf(codes.NotFound, "scrape config with job name %q not found", jobName)
}

// CreateScrapeConfig creates a new scrape config.
// Errors: InvalidArgument(3) if some argument is not valid,
// AlreadyExists(6) if scrape config with that job name is already present,
// FailedPrecondition(9) if reachability check was requested and some scrape target can't be reached.
func (svc *Service) CreateScrapeConfig(ctx context.Context, cfg *ScrapeConfig, checkReachability bool) error {
	svc.lock.Lock()
	defer svc.lock.Unlock()

	// start scraping targets early
	var reachabilityCh chan ScrapeTargetReachability
	if checkReachability {
		var targets []string
		for _, sc := range cfg.StaticConfigs {
			for _, t := range sc.Targets {
				targets = append(targets, t)
			}
		}
		reachabilityCh = make(chan ScrapeTargetReachability, len(targets)) // set cap so checkReachability always exits
		svc.checkReachability(ctx, cfg, targets, reachabilityCh)
	}

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

	if checkReachability {
		for r := range reachabilityCh {
			if msg := r.Error; msg != "" {
				if r.Target != "" {
					msg = fmt.Sprintf("%s: %s", r.Target, msg)
				}
				return status.Error(codes.FailedPrecondition, msg)
			}
		}
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

	// do not check that targets are reachable - we do that only for external exporters

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
