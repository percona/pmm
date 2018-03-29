// Copyright 2015 The Prometheus Authors
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

package zookeeper

import (
	"fmt"
	"strings"
	"time"

	"github.com/prometheus/common/model"

	yaml_util "github.com/Percona-Lab/promconfig/util/yaml"
)

var (
	// DefaultServersetSDConfig is the default Serverset SD configuration.
	DefaultServersetSDConfig = ServersetSDConfig{
		Timeout: model.Duration(10 * time.Second),
	}
	// DefaultNerveSDConfig is the default Nerve SD configuration.
	DefaultNerveSDConfig = NerveSDConfig{
		Timeout: model.Duration(10 * time.Second),
	}
)

// ServersetSDConfig is the configuration for Twitter serversets in Zookeeper based discovery.
type ServersetSDConfig struct {
	Servers []string       `yaml:"servers"`
	Paths   []string       `yaml:"paths"`
	Timeout model.Duration `yaml:"timeout,omitempty"`

	// Catches all undefined fields and must be empty after parsing.
	XXX map[string]interface{} `yaml:",inline"`
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (c *ServersetSDConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*c = DefaultServersetSDConfig
	type plain ServersetSDConfig
	err := unmarshal((*plain)(c))
	if err != nil {
		return err
	}
	if err := yaml_util.CheckOverflow(c.XXX, "serverset_sd_config"); err != nil {
		return err
	}
	if len(c.Servers) == 0 {
		return fmt.Errorf("serverset SD config must contain at least one Zookeeper server")
	}
	if len(c.Paths) == 0 {
		return fmt.Errorf("serverset SD config must contain at least one path")
	}
	for _, path := range c.Paths {
		if !strings.HasPrefix(path, "/") {
			return fmt.Errorf("serverset SD config paths must begin with '/': %s", path)
		}
	}
	return nil
}

// NerveSDConfig is the configuration for AirBnB's Nerve in Zookeeper based discovery.
type NerveSDConfig struct {
	Servers []string       `yaml:"servers"`
	Paths   []string       `yaml:"paths"`
	Timeout model.Duration `yaml:"timeout,omitempty"`

	// Catches all undefined fields and must be empty after parsing.
	XXX map[string]interface{} `yaml:",inline"`
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (c *NerveSDConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*c = DefaultNerveSDConfig
	type plain NerveSDConfig
	err := unmarshal((*plain)(c))
	if err != nil {
		return err
	}
	if err := yaml_util.CheckOverflow(c.XXX, "nerve_sd_config"); err != nil {
		return err
	}
	if len(c.Servers) == 0 {
		return fmt.Errorf("nerve SD config must contain at least one Zookeeper server")
	}
	if len(c.Paths) == 0 {
		return fmt.Errorf("nerve SD config must contain at least one path")
	}
	for _, path := range c.Paths {
		if !strings.HasPrefix(path, "/") {
			return fmt.Errorf("nerve SD config paths must begin with '/': %s", path)
		}
	}
	return nil
}
