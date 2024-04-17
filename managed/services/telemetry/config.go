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

// Package telemetry provides telemetry functionality.
package telemetry

import (
	_ "embed" //nolint:golint
	"os"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/percona/pmm/managed/utils/envvars"
)

const (
	envDisableSend           = "PERCONA_TEST_TELEMETRY_DISABLE_SEND"
	envConfigFile            = "PERCONA_TEST_TELEMETRY_FILE"
	envDisableStartDelay     = "PERCONA_TEST_TELEMETRY_DISABLE_START_DELAY"
	envReportingInterval     = "PERCONA_TEST_TELEMETRY_INTERVAL"
	envReportingRetryBackoff = "PERCONA_TEST_TELEMETRY_RETRY_BACKOFF"
)

const (
	dsVM              = DataSourceName("VM")
	dsQANDBSelect     = DataSourceName("QANDB_SELECT")
	dsPMMDBSelect     = DataSourceName("PMMDB_SELECT")
	dsGRAFANADBSelect = DataSourceName("GRAFANADB_SELECT")
	dsEnvVars         = DataSourceName("ENV_VARS")
)

// DataSources holds all possible data source types.
type DataSources struct {
	VM              *DSConfigVM        `yaml:"VM"`
	QanDBSelect     *DSConfigQAN       `yaml:"QANDB_SELECT"`
	PmmDBSelect     *DSConfigPMMDB     `yaml:"PMMDB_SELECT"`
	GrafanaDBSelect *DSConfigGrafanaDB `yaml:"GRAFANADB_SELECT"`
	EnvVars         *DSConfigEnvVars   `yaml:"ENV_VARS"`
}

// ServiceConfig telemetry config.
type ServiceConfig struct {
	l            *logrus.Entry
	Enabled      bool            `yaml:"enabled"`
	telemetry    []Config        `yaml:"-"`
	SaasHostname string          `yaml:"saas_hostname"`
	DataSources  DataSources     `yaml:"datasources"`
	Reporting    ReportingConfig `yaml:"reporting"`
}

// FileConfig top level telemetry config element.
type FileConfig struct {
	Telemetry []Config `yaml:"telemetry"`
}

// DSConfigQAN telemetry config.
type DSConfigQAN struct {
	Enabled bool          `yaml:"enabled"`
	Timeout time.Duration `yaml:"timeout"`
	DSN     string        `yaml:"-"`
}

// DSConfigVM telemetry config.
type DSConfigVM struct {
	Enabled bool          `yaml:"enabled"`
	Timeout time.Duration `yaml:"timeout"`
	Address string        `yaml:"address"`
}

// DSConfigPMMDB telemetry config.
type DSConfigPMMDB struct { //nolint:musttag
	Enabled                bool          `yaml:"enabled"`
	Timeout                time.Duration `yaml:"timeout"`
	UseSeparateCredentials bool          `yaml:"use_separate_credentials"`
	DSN                    struct {
		Scheme string
		Host   string
		DB     string
		Params string
	} `yaml:"-"`
	// Credentials used by PMM
	Credentials struct {
		Username string
		Password string
	} `yaml:"-"`
	SeparateCredentials struct {
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"separate_credentials"`
}

// DSConfigGrafanaDB is a Grafana telemetry config.
type DSConfigGrafanaDB DSConfigPMMDB

// DSConfigEnvVars is an env variable telemetry config.
type DSConfigEnvVars struct {
	Enabled bool `yaml:"enabled"`
}

// Config telemetry config.
type Config struct {
	ID        string           `yaml:"id"`
	Source    string           `yaml:"source"`
	Query     string           `yaml:"query"`
	Summary   string           `yaml:"summary"`
	Transform *ConfigTransform `yaml:"transform"`
	Extension ExtensionType    `yaml:"extension"`
	Data      []ConfigData
}

// ConfigTransform is a telemetry config transformation.
type ConfigTransform struct {
	Type   ConfigTransformType `yaml:"type"`
	Metric string              `yaml:"metric"`
}

// ConfigTransformType is a config transform type.
type ConfigTransformType string

const (
	// JSONTransform converts multiple metrics in one formatted as JSON.
	JSONTransform = ConfigTransformType("JSON")
	// StripValuesTransform strips values from metrics, replacing them with 1 to indicate presence.
	StripValuesTransform = ConfigTransformType("StripValues")
)

// ConfigData is a telemetry data config.
type ConfigData struct {
	MetricName string `yaml:"metric_name"`
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
	Send         bool          `yaml:"send"`
	SendOnStart  bool          `yaml:"send_on_start"`
	Interval     time.Duration `yaml:"interval"`
	RetryBackoff time.Duration `yaml:"retry_backoff"`
	SendTimeout  time.Duration `yaml:"send_timeout"`
	RetryCount   int           `yaml:"retry_count"`
}

//go:embed config.default.yml
var defaultConfig string

// ExtensionType represents the type of telemetry extension.
type ExtensionType string

const (
	// UIEventsExtension is a constant for the UI events telemetry extension.
	UIEventsExtension = ExtensionType("UIEventsExtension")
)

// Init initializes telemetry config.
func (c *ServiceConfig) Init(l *logrus.Entry) error { //nolint:gocognit
	c.l = l

	configFile := os.Getenv(envConfigFile)

	telemetry, err := c.loadMetricsConfig(configFile)
	if err != nil {
		return errors.Wrap(err, "failed to load telemetry config")
	}
	c.telemetry = telemetry

	if d, err := time.ParseDuration(os.Getenv(envReportingInterval)); err == nil && d > 0 {
		l.Warnf("Interval changed to %s.", d)
		c.Reporting.Interval = d
	}
	if d, err := time.ParseDuration(os.Getenv(envReportingRetryBackoff)); err == nil && d > 0 {
		l.Warnf("Retry backoff changed to %s.", d)
		c.Reporting.RetryBackoff = d
	}

	if c.SaasHostname == "" {
		host, err := envvars.GetPlatformAddress()
		c.SaasHostname = host
		if err != nil {
			return errors.Wrap(err, "failed to get SaaSHost")
		}
	}

	disabledSendStr, ok := os.LookupEnv(envDisableSend)
	if ok {
		disabledSend, err := strconv.ParseBool(disabledSendStr)
		if err != nil {
			c.l.Warnf("Cannot parse environment variable [%s] as bool.", envDisableSend)
		} else {
			c.l.Debugf("Overriding Telemetry.Reporting.Send with environment variable [%s] to %t.", envDisableSend, disabledSend)
			c.Reporting.Send = !disabledSend
		}
	} else {
		c.l.Debugf("[%s] is not set", envDisableSend)
	}

	disableOnStartSendStr, ok := os.LookupEnv(envDisableStartDelay)
	if ok {
		disableOnStartSend, err := strconv.ParseBool(disableOnStartSendStr)
		if err != nil {
			c.l.Warnf("Cannot parse environment variable [%s] as bool.", envDisableStartDelay)
		} else {
			c.l.Debugf("Overriding Telemetry.Reporting.SendOnStart with environment variable [%s] to %t.", envDisableStartDelay, disableOnStartSend)
			c.Reporting.SendOnStart = !disableOnStartSend
		}
	}

	return nil
}

func (c *ServiceConfig) loadMetricsConfig(configFile string) ([]Config, error) {
	var fileConfigs []FileConfig
	var fileCfg FileConfig

	var config []byte
	if configFile != "" {
		file, err := os.ReadFile(configFile) //nolint:gosec
		if err != nil {
			return nil, err
		}
		config = file
	} else {
		c.l.Info("Using default metrics config")
		config = []byte(defaultConfig)
	}
	if err := yaml.Unmarshal(config, &fileCfg); err != nil {
		return nil, errors.Wrap(err, "cannot unmarshal default config")
	}
	fileConfigs = append(fileConfigs, fileCfg)

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
