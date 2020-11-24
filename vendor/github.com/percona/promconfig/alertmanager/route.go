// promconfig
// Copyright 2020 Percona LLC
//
// Based on Prometheus systems and service monitoring server.
// Copyright 2015 The Prometheus Authors.
//
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

package alertmanager

import "github.com/percona/promconfig"

// A Route is a node that contains definitions of how to handle alerts.
type Route struct {
	Receiver string `yaml:"receiver,omitempty"`

	GroupBy []string `yaml:"group_by,omitempty"`

	Match    map[string]string `yaml:"match,omitempty"`
	MatchRE  map[string]string `yaml:"match_re,omitempty"`
	Continue bool              `yaml:"continue"`
	Routes   []*Route          `yaml:"routes,omitempty"`

	GroupWait      promconfig.Duration `yaml:"group_wait,omitempty"`
	GroupInterval  promconfig.Duration `yaml:"group_interval,omitempty"`
	RepeatInterval promconfig.Duration `yaml:"repeat_interval,omitempty"`
}
