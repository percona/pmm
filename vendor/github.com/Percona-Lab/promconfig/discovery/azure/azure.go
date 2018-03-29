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

package azure

import (
	"fmt"
	"time"

	"github.com/prometheus/common/model"

	yaml_util "github.com/Percona-Lab/promconfig/util/yaml"
)

var (
	// DefaultSDConfig is the default Azure SD configuration.
	DefaultSDConfig = SDConfig{
		Port:            80,
		RefreshInterval: model.Duration(5 * time.Minute),
	}
)

// SDConfig is the configuration for Azure based service discovery.
type SDConfig struct {
	Port            int            `yaml:"port"`
	SubscriptionID  string         `yaml:"subscription_id"`
	TenantID        string         `yaml:"tenant_id,omitempty"`
	ClientID        string         `yaml:"client_id,omitempty"`
	ClientSecret    string         `yaml:"client_secret,omitempty"`
	RefreshInterval model.Duration `yaml:"refresh_interval,omitempty"`

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
	if c.SubscriptionID == "" {
		return fmt.Errorf("Azure SD configuration requires a subscription_id")
	}

	return yaml_util.CheckOverflow(c.XXX, "azure_sd_config")
}
