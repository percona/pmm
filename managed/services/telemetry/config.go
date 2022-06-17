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

// Package telemetry provides telemetry functionality.
package telemetry

import (
	_ "embed" //nolint:golint
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/percona/pmm/managed/utils/envvars"
)

// ServiceConfig telemetry config.
type ServiceConfig struct {
	l            *logrus.Entry
	Enabled      bool            `yaml:"enabled"`
	LoadDefaults bool            `yaml:"load_defaults"` //nolint:tagliatelle
	telemetry    []Config        `yaml:"-"`
	Endpoints    EndpointsConfig `yaml:"endpoints"`
	SaasHostname string          `yaml:"saas_hostname"` //nolint:tagliatelle
	DataSources  struct {
		VM          *DataSourceVictoriaMetrics `yaml:"VM"`
		QanDBSelect *DSConfigQAN               `yaml:"QANDB_SELECT"` //nolint:tagliatelle
		PmmDBSelect *DSConfigPMMDB             `yaml:"PMMDB_SELECT"` //nolint:tagliatelle
	} `yaml:"datasources"`
	Reporting ReportingConfig `yaml:"reporting"`
}

// FileConfig top level telemetry config element.
type FileConfig struct {
	Telemetry []Config `yaml:"telemetry"`
}

// EndpointsConfig telemetry endpoint config.
type EndpointsConfig struct {
	Report string `yaml:"report"`
}

// ReportEndpointURL returns reporting endpoint URL.
func (c *ServiceConfig) ReportEndpointURL() string {
	return fmt.Sprintf(c.Endpoints.Report, c.SaasHostname)
}

// DSConfigQAN telemetry config.
type DSConfigQAN struct {
	Enabled bool          `yaml:"enabled"`
	Timeout time.Duration `yaml:"timeout"`
	DSN     string        `yaml:"dsn"`
}

// DataSourceVictoriaMetrics telemetry config.
type DataSourceVictoriaMetrics struct {
	Enabled bool          `yaml:"enabled"`
	Timeout time.Duration `yaml:"timeout"`
	Address string        `yaml:"address"`
}

// DSConfigPMMDB telemetry config.
type DSConfigPMMDB struct {
	Enabled                bool          `yaml:"enabled"`
	Timeout                time.Duration `yaml:"timeout"`
	UseSeparateCredentials bool          `yaml:"use_separate_credentials"` //nolint:tagliatelle
	// Credentials used by PMM
	DSN struct {
		Scheme string
		Host   string
		DB     string
		Params string
	} `yaml:"-"`
	Credentials struct {
		Username string
		Password string
	} `yaml:"-"`
	SeparateCredentials struct {
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"separate_credentials"` //nolint:tagliatelle
}

// Config telemetry config.
type Config struct {
	ID      string `yaml:"id"`
	Source  string `yaml:"source"`
	Query   string `yaml:"query"`
	Summary string `yaml:"summary"`
	Data    []ConfigData
}

// ConfigData telemetry config.
type ConfigData struct {
	MetricName string `yaml:"metric_name"` //nolint:tagliatelle
	Label      string `yaml:"label"`
	Value      string `yaml:"value"`
	Column     string `yaml:"column"`
}

func (c *Config) mapByColumn() map[string][]ConfigData {
	result := make(map[string][]ConfigData, len(c.Data))
	for _, each := range c.Data {
		result[each.Column] = append(result[each.Column], each)
	}
	return result
}

// ReportingConfig reporting config.
type ReportingConfig struct {
	SkipTLSVerification bool          `yaml:"skip_tls_verification"` //nolint:tagliatelle
	SendOnStart         bool          `yaml:"send_on_start"`         //nolint:tagliatelle
	IntervalEnv         string        `yaml:"interval_env"`          //nolint:tagliatelle
	Interval            time.Duration `yaml:"interval"`
	RetryBackoffEnv     string        `yaml:"retry_backoff_env"` //nolint:tagliatelle
	RetryBackoff        time.Duration `yaml:"retry_backoff"`     //nolint:tagliatelle
	SendTimeout         time.Duration `yaml:"send_timeout"`      //nolint:tagliatelle
	RetryCount          int           `yaml:"retry_count"`       //nolint:tagliatelle
}

//go:embed config.default.yml
var defaultConfig string

// Init initializes telemetry config.
func (c *ServiceConfig) Init(l *logrus.Entry) error { //nolint:gocognit
	c.l = l

	telemetry, err := c.loadConfig()
	if err != nil {
		return errors.Wrap(err, "failed to load telemetry config")
	}
	c.telemetry = telemetry

	if d, err := time.ParseDuration(os.Getenv(c.Reporting.IntervalEnv)); err == nil && d > 0 {
		l.Warnf("Interval changed to %s.", d)
		c.Reporting.Interval = d
	}
	if d, err := time.ParseDuration(os.Getenv(c.Reporting.RetryBackoffEnv)); err == nil && d > 0 {
		l.Warnf("Retry backoff changed to %s.", d)
		c.Reporting.RetryBackoff = d
	}

	if c.SaasHostname == "" {
		host, err := envvars.GetSAASHost()
		c.SaasHostname = host
		if err != nil {
			return errors.Wrap(err, "failed to get SaaSHost")
		}
	}

	return nil
}

func (c *ServiceConfig) loadConfig() ([]Config, error) { //nolint:cyclop
	var fileConfigs []FileConfig //nolint:prealloc
	var fileCfg FileConfig

	if c.LoadDefaults {
		defaultConfigBytes := []byte(defaultConfig)
		if err := yaml.Unmarshal(defaultConfigBytes, &fileCfg); err != nil {
			return nil, errors.Wrap(err, "cannot unmashal default config")
		}
		fileConfigs = append(fileConfigs, fileCfg)
	}

	if err := c.validateConfig(fileConfigs); err != nil {
		c.l.Errorf(err.Error())
	}

	return c.merge(fileConfigs), nil
}

func (c *ServiceConfig) merge(cfgs []FileConfig) []Config {
	var result []Config
	ids := make(map[string]bool)
	for _, cfg := range cfgs {
		for _, each := range cfg.Telemetry {
			_, exist := ids[each.ID]
			if !exist {
				ids[each.ID] = true
				result = append(result, each)
			}
		}
	}
	return result
}

func (c *ServiceConfig) validateConfig(cfgs []FileConfig) error {
	ids := make(map[string]bool)
	for _, cfg := range cfgs {
		for _, each := range cfg.Telemetry {
			_, exist := ids[each.ID]
			if exist {
				return errors.Errorf("telemetry config ID duplication: %s", each.ID)
			}
			ids[each.ID] = true
		}
	}
	return nil
}
