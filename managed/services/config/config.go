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

// Package config provides configuration facility.
//
// Deprecated: please don't extend this package, it will be removed soon https://jira.percona.com/browse/PMM-11155
package config

import (
	_ "embed"
	"os"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/percona/pmm/managed/services/telemetry"
)

const (
	envConfigPath     = "PERCONA_PMM_CONFIG_PATH"
	defaultConfigPath = "/etc/percona/pmm/pmm-managed.yml"
)

//go:embed pmm-managed.yaml
var defaultConfig string

// Service config service.
type Service struct {
	l      *logrus.Entry
	Config Config
}

// Config application config.
type Config struct {
	Services struct {
		Telemetry telemetry.ServiceConfig `yaml:"telemetry"`
	} `yaml:"services"`
}

// NewService makes new config service.
func NewService() *Service {
	l := logrus.WithField("component", "config")

	return &Service{
		l: l,
	}
}

// Load initializes config.
func (s *Service) Load() error {
	configPath, present := os.LookupEnv(envConfigPath)
	if present {
		if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
			return errors.Wrapf(err, "config file [%s] doen't not exit", configPath)
		}
	} else {
		s.l.Debugf("[%s] is not set, using default location [%s]", envConfigPath, defaultConfigPath)
	}

	var cfg Config

	if _, err := os.Stat(configPath); err == nil {
		s.l.Trace("config exists, reading file")
		buf, err := os.ReadFile(configPath) //nolint:gosec
		if err != nil {
			return errors.Wrapf(err, "error while reading config [%s]", configPath)
		}
		if err := yaml.Unmarshal(buf, &cfg); err != nil {
			return errors.Wrapf(err, "cannot unmarshal config [%s]", configPath)
		}
	} else {
		s.l.Trace("config does not exist, fallback to embedded config")
		if err := yaml.Unmarshal([]byte(defaultConfig), &cfg); err != nil {
			return errors.Wrapf(err, "cannot unmarshal config [%s]", configPath)
		}
	}

	if err := cfg.Services.Telemetry.Init(s.l); err != nil {
		return err
	}

	s.Config = cfg

	return nil
}
