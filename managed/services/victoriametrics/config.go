// pmm-managed
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

package victoriametrics

import (
	_ "embed" //nolint:golint
	"github.com/sirupsen/logrus"
	"time"
)

// ServiceConfig victoriametrics config.
type ServiceConfig struct {
	l                          *logrus.Entry
	ConfigurationUpdateTimeout time.Duration `yaml:"configuration_update_timeout"` //nolint:tagliatelle
	ScrapeConfigPath           string        `yaml:"scrape_config_path"`           //nolint:tagliatelle
	BasePrometheusConfigPath   string        `yaml:"base_prometheus_config_path"`  //nolint:tagliatelle
	VictoriaMetricsDir         string        `yaml:"victoriametrics_dir"`          //nolint:tagliatelle
	VictoriaMetricsDataDir     string        `yaml:"victoriametrics_data_dir"`     //nolint:tagliatelle
	VictoriaMetricsDirUser     string        `yaml:"victoriametrics_dir_user"`     //nolint:tagliatelle
	VictoriaMetricsDirGroup    string        `yaml:"victoriametrics_dir_group"`    //nolint:tagliatelle
}

// Init initializes telemetry config.
func (c *ServiceConfig) Init(l *logrus.Entry) error {
	c.l = l

	return nil
}
