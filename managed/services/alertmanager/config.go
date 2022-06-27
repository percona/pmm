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

package alertmanager

import (
	_ "embed" //nolint:golint
	"github.com/sirupsen/logrus"
)

// ServiceConfig victoriametrics config.
type ServiceConfig struct {
	l              *logrus.Entry
	Host           string `yaml:"host"`             //nolint:tagliatelle
	DirUser        string `yaml:"dir_user"`         //nolint:tagliatelle
	DirGroup       string `yaml:"dir_group"`        //nolint:tagliatelle
	Dir            string `yaml:"dir"`              //nolint:tagliatelle
	CertDir        string `yaml:"cert_dir"`         //nolint:tagliatelle
	DataDir        string `yaml:"data_dir"`         //nolint:tagliatelle
	ConfigPath     string `yaml:"config_path"`      //nolint:tagliatelle
	BaseConfigPath string `yaml:"base_config_path"` //nolint:tagliatelle
}

// Init initializes telemetry config.
func (c *ServiceConfig) Init(l *logrus.Entry) error {
	c.l = l

	return nil
}
