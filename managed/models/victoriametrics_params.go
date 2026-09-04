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

package models

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	config "github.com/percona/promconfig"
	"gopkg.in/yaml.v3"
)

const (
	// BasePrometheusConfigPath - basic path with prometheus config,
	// that user can mount to container.
	BasePrometheusConfigPath = "/srv/prometheus/prometheus.base.yml"
	// VMBaseURL is the base URL for VictoriaMetrics.
	VMBaseURL = "http://127.0.0.1:9090/prometheus/"
)

// VictoriaMetricsParams - defines flags and settings for victoriametrics.
type VictoriaMetricsParams struct {
	// VMAlertFlags additional flags for VMAlert.
	VMAlertFlags []string
	// BaseConfigPath defines path for basic prometheus config.
	BaseConfigPath string
	// url defines url of Victoria Metrics
	url *url.URL
}

// ParseVictoriaMetricsURL parses and validates a VictoriaMetrics base URL (PMM_VM_URL): an http or
// https URL with a host. A trailing slash is appended when missing so that paths resolve under it.
// Error messages never echo credentials the URL may carry.
func ParseVictoriaMetricsURL(vmURL string) (*url.URL, error) {
	if !strings.HasSuffix(vmURL, "/") {
		vmURL += "/"
	}

	URL, err := url.Parse(vmURL)
	if err != nil {
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			err = urlErr.Err
		}
		return nil, fmt.Errorf("invalid VictoriaMetrics URL: %w", err)
	}
	if (URL.Scheme != "http" && URL.Scheme != "https") || URL.Host == "" || URL.Opaque != "" {
		return nil, fmt.Errorf("invalid VictoriaMetrics URL %q: expected http(s)://host[:port][/path]", URL.Redacted())
	}

	return URL, nil
}

// NewVictoriaMetricsParams - returns configuration params for VictoriaMetrics.
func NewVictoriaMetricsParams(basePath, vmURL string) (*VictoriaMetricsParams, error) {
	URL, err := ParseVictoriaMetricsURL(vmURL)
	if err != nil {
		return nil, err
	}
	vmp := &VictoriaMetricsParams{
		BaseConfigPath: basePath,
		url:            URL,
	}
	err = vmp.UpdateParams()
	if err != nil {
		return vmp, err
	}

	return vmp, nil
}

// UpdateParams - reads configuration file and updates corresponding flags.
func (vmp *VictoriaMetricsParams) UpdateParams() error {
	err := vmp.loadVMAlertParams()
	if err != nil {
		return fmt.Errorf("cannot update VMAlertFlags config param: %w", err)
	}

	return nil
}

// loadVMAlertParams - load params and converts it to vmalert flags.
func (vmp *VictoriaMetricsParams) loadVMAlertParams() error {
	buf, err := os.ReadFile(vmp.BaseConfigPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("cannot read baseConfigPath for VMAlertParams: %w", err)
		}
		// fast return if users configuration doesn't exist with path
		// /srv/prometheus/prometheus.base.yml,
		// its maybe mounted into container by user.
		return nil
	}
	var cfg config.Config
	err = yaml.Unmarshal(buf, &cfg)
	if err != nil {
		return fmt.Errorf("cannot unmarshal baseConfigPath for VMAlertFlags: %w", err)
	}
	vmalertFlags := make([]string, 0, len(vmp.VMAlertFlags))
	for _, r := range cfg.RuleFiles {
		vmalertFlags = append(vmalertFlags, "--rule="+r)
	}
	if cfg.GlobalConfig.EvaluationInterval != 0 {
		vmalertFlags = append(vmalertFlags, "--evaluationInterval="+cfg.GlobalConfig.EvaluationInterval.String())
	}
	vmp.VMAlertFlags = vmalertFlags

	return nil
}

// ExternalVM returns true if VictoriaMetrics is configured to run externally.
func (vmp *VictoriaMetricsParams) ExternalVM() bool {
	return !internalAddr(vmp.url.Hostname())
}

// URL returns the base URL for VictoriaMetrics.
func (vmp *VictoriaMetricsParams) URL() string {
	return vmp.url.String()
}

// URLFor returns the URL for a specific path in VictoriaMetrics.
func (vmp *VictoriaMetricsParams) URLFor(path string) (*url.URL, error) {
	if path == "" {
		return vmp.url, nil
	}
	return vmp.url.Parse(path)
}
