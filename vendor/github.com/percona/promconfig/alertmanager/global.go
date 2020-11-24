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

import (
	"github.com/percona/promconfig"
)

// GlobalConfig defines configuration parameters that are valid globally
// unless overwritten.
type GlobalConfig struct {
	// ResolveTimeout is the time after which an alert is declared resolved
	// if it has not been updated.
	ResolveTimeout promconfig.Duration `yaml:"resolve_timeout"`

	HTTPConfig promconfig.HTTPClientConfig `yaml:"http_config,omitempty"`

	SMTPFrom         string   `yaml:"smtp_from,omitempty"`
	SMTPHello        string   `yaml:"smtp_hello,omitempty"`
	SMTPSmarthost    HostPort `yaml:"smtp_smarthost,omitempty"`
	SMTPAuthUsername string   `yaml:"smtp_auth_username,omitempty"`
	SMTPAuthPassword string   `yaml:"smtp_auth_password,omitempty"`
	SMTPAuthSecret   string   `yaml:"smtp_auth_secret,omitempty"`
	SMTPAuthIdentity string   `yaml:"smtp_auth_identity,omitempty"`
	SMTPRequireTLS   bool     `yaml:"smtp_require_tls"`
	SlackAPIURL      string   `yaml:"slack_api_url,omitempty"`
	PagerdutyURL     string   `yaml:"pagerduty_url,omitempty"`
	OpsGenieAPIURL   string   `yaml:"opsgenie_api_url,omitempty"`
	OpsGenieAPIKey   string   `yaml:"opsgenie_api_key,omitempty"`
	WeChatAPIURL     string   `yaml:"wechat_api_url,omitempty"`
	WeChatAPISecret  string   `yaml:"wechat_api_secret,omitempty"`
	WeChatAPICorpID  string   `yaml:"wechat_api_corp_id,omitempty"`
	VictorOpsAPIURL  string   `yaml:"victorops_api_url,omitempty"`
	VictorOpsAPIKey  string   `yaml:"victorops_api_key,omitempty"`
}

// HostPort represents a "host:port" network address.
type HostPort struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
}
