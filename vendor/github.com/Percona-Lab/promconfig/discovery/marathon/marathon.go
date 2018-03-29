// Copyright 2016 The Prometheus Authors
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

package marathon

import (
	"fmt"
	"time"

	config_util "github.com/Percona-Lab/promconfig/common/config"
	yaml_util "github.com/Percona-Lab/promconfig/util/yaml"
	"github.com/prometheus/common/model"
)

var (
	// DefaultSDConfig is the default Marathon SD configuration.
	DefaultSDConfig = SDConfig{
		Timeout:         model.Duration(30 * time.Second),
		RefreshInterval: model.Duration(30 * time.Second),
	}
)

// SDConfig is the configuration for services running on Marathon.
type SDConfig struct {
	Servers         []string              `yaml:"servers,omitempty"`
	Timeout         model.Duration        `yaml:"timeout,omitempty"`
	RefreshInterval model.Duration        `yaml:"refresh_interval,omitempty"`
	TLSConfig       config_util.TLSConfig `yaml:"tls_config,omitempty"`
	BearerToken     string                `yaml:"bearer_token,omitempty"`
	BearerTokenFile string                `yaml:"bearer_token_file,omitempty"`

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
	if err := yaml_util.CheckOverflow(c.XXX, "marathon_sd_config"); err != nil {
		return err
	}
	if len(c.Servers) == 0 {
		return fmt.Errorf("Marathon SD config must contain at least one Marathon server")
	}
	if len(c.BearerToken) > 0 && len(c.BearerTokenFile) > 0 {
		return fmt.Errorf("at most one of bearer_token & bearer_token_file must be configured")
	}

	return nil
}
