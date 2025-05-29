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

// Package vmalert provides facilities for working with VMAlert.
package vmalert

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/pkg/errors"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/managed/utils/irt"
)

const (
	updateBatchDelay             = time.Second
	configurationUpdateTimeout   = 3 * time.Second
	defaultDialTimeout           = 3 * time.Second
	defaultKeepAliveTimeout      = 30 * time.Second
	defaultIdleConnTimeout       = 90 * time.Second
	defaultExpectContinueTimeout = 1 * time.Second
	defaultMaxIdleConns          = 1
)

// Service is responsible for interactions with victoria metrics.
type Service struct {
	baseURL       *url.URL
	client        *http.Client
	externalRules *ExternalRules
	irtm          prom.Collector

	l        *logrus.Entry
	reloadCh chan struct{}
}

// NewVMAlert creates new Victoria Metrics Alert service.
func NewVMAlert(externalRules *ExternalRules, baseURL string) (*Service, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var t http.RoundTripper = &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   defaultDialTimeout,
			KeepAlive: defaultKeepAliveTimeout,
		}).DialContext,
		MaxIdleConns:          defaultMaxIdleConns,
		IdleConnTimeout:       defaultIdleConnTimeout,
		ExpectContinueTimeout: defaultExpectContinueTimeout,
	}
	if logrus.GetLevel() >= logrus.TraceLevel {
		t = irt.WithLogger(t, logrus.WithField("component", "vmalert/client").Tracef)
	}
	t, irtm := irt.WithMetrics(t, "vmalert")

	return &Service{
		externalRules: externalRules,
		baseURL:       u,
		client: &http.Client{
			Transport: t,
		},
		l:        logrus.WithField("component", "vmalert"),
		irtm:     irtm,
		reloadCh: make(chan struct{}, 1),
	}, nil
}

// Run runs VMAlert configuration update loop until ctx is canceled.
func (svc *Service) Run(ctx context.Context) {
	// If you change this and related methods,
	// please do similar changes in victoriametrics packages.

	svc.l.Info("Starting...")
	defer svc.l.Info("Done.")

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

// RequestConfigurationUpdate requests VMAlert configuration update.
func (svc *Service) RequestConfigurationUpdate() {
	select {
	case svc.reloadCh <- struct{}{}:
	default:
	}
}

// updateConfiguration reads alerts configuration from file
// compares it with cached and replace if needed.
func (svc *Service) updateConfiguration(ctx context.Context) error {
	start := time.Now()
	defer func() {
		if dur := time.Since(start); dur > time.Second {
			svc.l.Warnf("updateConfiguration took %s.", dur)
		}
	}()

	if err := svc.reload(ctx); err != nil {
		return errors.WithStack(err)
	}
	svc.l.Infof("Configuration reloaded.")

	return nil
}

// reload asks VMAlert to reload configuration.
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
	defer resp.Body.Close() //nolint:gosec,errcheck,nolintlint

	b, err := io.ReadAll(resp.Body)
	svc.l.Debugf("VMAlert reload: %s", b)
	if err != nil {
		return errors.WithStack(err)
	}

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("expected 200, got %d", resp.StatusCode)
	}
	return nil
}

// IsReady verifies that VMAlert works.
func (svc *Service) IsReady(ctx context.Context) error {
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
	svc.l.Debugf("VMAlert health: %s", b)
	if err != nil {
		return errors.WithStack(err)
	}
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("expected 200, got %d", resp.StatusCode)
	}

	return nil
}

// Describe implements prometheus.Collector.
func (svc *Service) Describe(ch chan<- *prom.Desc) {
	svc.irtm.Describe(ch)
}

// Collect implements prometheus.Collector.
func (svc *Service) Collect(ch chan<- prom.Metric) {
	svc.irtm.Collect(ch)
}

// Check interfaces.
var (
	_ prom.Collector = (*Service)(nil)
)
