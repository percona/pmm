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

package consul

import (
	"fmt"
	"strings"

	config_util "github.com/Percona-Lab/promconfig/common/config"
	yaml_util "github.com/Percona-Lab/promconfig/util/yaml"
)

var (
	// DefaultSDConfig is the default Consul SD configuration.
	DefaultSDConfig = SDConfig{
		TagSeparator: ",",
		Scheme:       "http",
		Server:       "localhost:8500",
	}
)

// SDConfig is the configuration for Consul service discovery.
type SDConfig struct {
	Server       string `yaml:"server,omitempty"`
	Token        string `yaml:"token,omitempty"`
	Datacenter   string `yaml:"datacenter,omitempty"`
	TagSeparator string `yaml:"tag_separator,omitempty"`
	Scheme       string `yaml:"scheme,omitempty"`
	Username     string `yaml:"username,omitempty"`
	Password     string `yaml:"password,omitempty"`
	// The list of services for which targets are discovered.
	// Defaults to all services if empty.
	Services []string `yaml:"services"`

	TLSConfig config_util.TLSConfig `yaml:"tls_config,omitempty"`
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
	if err := yaml_util.CheckOverflow(c.XXX, "consul_sd_config"); err != nil {
		return err
	}
	if strings.TrimSpace(c.Server) == "" {
		return fmt.Errorf("Consul SD configuration requires a server address")
	}
	return nil
}
