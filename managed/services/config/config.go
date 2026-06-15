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
	"errors"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/percona/pmm/managed/services/telemetry"
)

const (
	envConfigPath     = "PMM_TEST_CONFIG_PATH"
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
		_, err := os.Stat(configPath)
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("config file [%s] doesn't exist: %w", configPath, err)
		}
	} else {
		s.l.Debugf("[%s] is not set, using default location [%s]", envConfigPath, defaultConfigPath)
	}

	var cfg Config

	_, err := os.Stat(configPath)
	if err == nil {
		s.l.Trace("config exists, reading file")
		buf, err := os.ReadFile(configPath) //nolint:gosec
		if err != nil {
			return fmt.Errorf("error while reading config [%s]: %w", configPath, err)
		}
		err = yaml.Unmarshal(buf, &cfg)
		if err != nil {
			return fmt.Errorf("cannot unmarshal config [%s]: %w", configPath, err)
		}
	} else {
		s.l.Trace("config does not exist, fallback to embedded config")
		err := yaml.Unmarshal([]byte(defaultConfig), &cfg)
		if err != nil {
			return fmt.Errorf("cannot unmarshal config [%s]: %w", configPath, err)
		}
	}

	err = cfg.Services.Telemetry.Init(s.l)
	if err != nil {
		return err
	}

	s.Config = cfg

	return nil
}
