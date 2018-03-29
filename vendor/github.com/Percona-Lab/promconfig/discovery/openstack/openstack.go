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

package openstack

import (
	"fmt"
	"time"

	yaml_util "github.com/Percona-Lab/promconfig/util/yaml"
	"github.com/prometheus/common/model"
)

var (
	// DefaultSDConfig is the default OpenStack SD configuration.
	DefaultSDConfig = SDConfig{
		Port:            80,
		RefreshInterval: model.Duration(60 * time.Second),
	}
)

// SDConfig is the configuration for OpenStack based service discovery.
type SDConfig struct {
	IdentityEndpoint string         `yaml:"identity_endpoint"`
	Username         string         `yaml:"username"`
	UserID           string         `yaml:"userid"`
	Password         string         `yaml:"password"`
	ProjectName      string         `yaml:"project_name"`
	ProjectID        string         `yaml:"project_id"`
	DomainName       string         `yaml:"domain_name"`
	DomainID         string         `yaml:"domain_id"`
	Role             Role           `yaml:"role"`
	Region           string         `yaml:"region"`
	RefreshInterval  model.Duration `yaml:"refresh_interval,omitempty"`
	Port             int            `yaml:"port"`

	// Catches all undefined fields and must be empty after parsing.
	XXX map[string]interface{} `yaml:",inline"`
}

// OpenStackRole is role of the target in OpenStack.
type Role string

// The valid options for OpenStackRole.
const (
	// OpenStack document reference
	// https://docs.openstack.org/nova/pike/admin/arch.html#hypervisors
	OpenStackRoleHypervisor Role = "hypervisor"
	// OpenStack document reference
	// https://docs.openstack.org/horizon/pike/user/launch-instances.html
	OpenStackRoleInstance Role = "instance"
)

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (c *Role) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := unmarshal((*string)(c)); err != nil {
		return err
	}
	switch *c {
	case OpenStackRoleHypervisor, OpenStackRoleInstance:
		return nil
	default:
		return fmt.Errorf("Unknown OpenStack SD role %q", *c)
	}
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (c *SDConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*c = DefaultSDConfig
	type plain SDConfig
	err := unmarshal((*plain)(c))
	if err != nil {
		return err
	}
	if c.Role == "" {
		return fmt.Errorf("role missing (one of: instance, hypervisor)")
	}
	if c.Region == "" {
		return fmt.Errorf("Openstack SD configuration requires a region")
	}
	return yaml_util.CheckOverflow(c.XXX, "openstack_sd_config")
}
