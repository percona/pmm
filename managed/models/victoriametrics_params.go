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

package models

import (
	"os"
	"strings"

	config "github.com/percona/promconfig"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

const (
	// BasePrometheusConfigPath - basic path with prometheus config,
	// that user can mount to container.
	BasePrometheusConfigPath = "/srv/prometheus/prometheus.base.yml"
	VMBaseURL                = "http://127.0.0.1:9090/prometheus/"
)

// VictoriaMetricsParams - defines flags and settings for victoriametrics.
type VictoriaMetricsParams struct {
	// VMAlertFlags additional flags for VMAlert.
	VMAlertFlags []string
	// BaseConfigPath defines path for basic prometheus config.
	BaseConfigPath string
	// URL defines URL of Victoria Metrics
	URL string
}

// NewVictoriaMetricsParams - returns configuration params for VictoriaMetrics.
func NewVictoriaMetricsParams(basePath string, vmURL string) (*VictoriaMetricsParams, error) {
	vmp := &VictoriaMetricsParams{
		BaseConfigPath: basePath,
		URL:            vmURL,
	}
	if err := vmp.UpdateParams(); err != nil {
		return vmp, err
	}

	return vmp, nil
}

// UpdateParams - reads configuration file and updates corresponding flags.
func (vmp *VictoriaMetricsParams) UpdateParams() error {
	if err := vmp.loadVMAlertParams(); err != nil {
		return errors.Wrap(err, "cannot update VMAlertFlags config param")
	}

	return nil
}

// loadVMAlertParams - load params and converts it to vmalert flags.
func (vmp *VictoriaMetricsParams) loadVMAlertParams() error {
	buf, err := os.ReadFile(vmp.BaseConfigPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return errors.Wrap(err, "cannot read baseConfigPath for VMAlertParams")
		}
		// fast return if users configuration doesn't exist with path
		// /srv/prometheus/prometheus.base.yml,
		// its maybe mounted into container by user.
		return nil
	}
	var cfg config.Config
	if err = yaml.Unmarshal(buf, &cfg); err != nil {
		return errors.Wrap(err, "cannot unmarshal baseConfigPath for VMAlertFlags")
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

func (vmp *VictoriaMetricsParams) ExternalVM() bool {
	return !strings.HasPrefix(vmp.URL, "http://127.0.0.1")
}
