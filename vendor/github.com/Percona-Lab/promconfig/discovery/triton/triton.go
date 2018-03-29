// Copyright 2017 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package triton

import (
	"fmt"
	"time"

	"github.com/prometheus/common/model"

	config_util "github.com/Percona-Lab/promconfig/common/config"
	yaml_util "github.com/Percona-Lab/promconfig/util/yaml"
)

var (
	// DefaultSDConfig is the default Triton SD configuration.
	DefaultSDConfig = SDConfig{
		Port:            9163,
		RefreshInterval: model.Duration(60 * time.Second),
		Version:         1,
	}
)

// SDConfig is the configuration for Triton based service discovery.
type SDConfig struct {
	Account         string                `yaml:"account"`
	DNSSuffix       string                `yaml:"dns_suffix"`
	Endpoint        string                `yaml:"endpoint"`
	Port            int                   `yaml:"port"`
	RefreshInterval model.Duration        `yaml:"refresh_interval,omitempty"`
	TLSConfig       config_util.TLSConfig `yaml:"tls_config,omitempty"`
	Version         int                   `yaml:"version"`
	// Catches all undefined fields and must be empty after parsing.
	XXX map[string]interface{} `yaml:",inline"`
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (c *SDConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*c = DefaultSDConfig
	type plain SDConfig
	err := unmarshal((*plain)(c))
	if err != nil {
		return err
	}
	if c.Account == "" {
		return fmt.Errorf("Triton SD configuration requires an account")
	}
	if c.DNSSuffix == "" {
		return fmt.Errorf("Triton SD configuration requires a dns_suffix")
	}
	if c.Endpoint == "" {
		return fmt.Errorf("Triton SD configuration requires an endpoint")
	}
	if c.RefreshInterval <= 0 {
		return fmt.Errorf("Triton SD configuration requires RefreshInterval to be a positive integer")
	}
	return yaml_util.CheckOverflow(c.XXX, "triton_sd_config")
}
