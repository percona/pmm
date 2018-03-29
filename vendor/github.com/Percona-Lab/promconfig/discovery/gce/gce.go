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

package gce

import (
	"fmt"
	"time"

	"github.com/prometheus/common/model"

	yaml_util "github.com/Percona-Lab/promconfig/util/yaml"
)

var (
	// DefaultSDConfig is the default GCE SD configuration.
	DefaultSDConfig = SDConfig{
		Port:            80,
		TagSeparator:    ",",
		RefreshInterval: model.Duration(60 * time.Second),
	}
)

// SDConfig is the configuration for GCE based service discovery.
type SDConfig struct {
	// Project: The Google Cloud Project ID
	Project string `yaml:"project"`

	// Zone: The zone of the scrape targets.
	// If you need to configure multiple zones use multiple gce_sd_configs
	Zone string `yaml:"zone"`

	// Filter: Can be used optionally to filter the instance list by other criteria.
	// Syntax of this filter string is described here in the filter query parameter section:
	// https://cloud.google.com/compute/docs/reference/latest/instances/list
	Filter string `yaml:"filter,omitempty"`

	RefreshInterval model.Duration `yaml:"refresh_interval,omitempty"`
	Port            int            `yaml:"port"`
	TagSeparator    string         `yaml:"tag_separator,omitempty"`

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
	if err := yaml_util.CheckOverflow(c.XXX, "gce_sd_config"); err != nil {
		return err
	}
	if c.Project == "" {
		return fmt.Errorf("GCE SD configuration requires a project")
	}
	if c.Zone == "" {
		return fmt.Errorf("GCE SD configuration requires a zone")
	}
	return nil
}
