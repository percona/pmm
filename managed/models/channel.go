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

package models

import (
	"database/sql/driver"
	"time"

	"gopkg.in/reform.v1"
)

//go:generate ../../bin/reform

// ChannelType represents notificaion channel type.
type ChannelType string

// Available notification channel types.
const (
	Email     = ChannelType("email")
	PagerDuty = ChannelType("pagerduty")
	Slack     = ChannelType("slack")
	WebHook   = ChannelType("webhook")
)

// Channel represents Integrated Alerting Notification Channel configuration.
//
//reform:ia_channels
type Channel struct {
	ID      string      `reform:"id,pk"`
	Summary string      `reform:"summary"`
	Type    ChannelType `reform:"type"`

	EmailConfig     *EmailConfig     `reform:"email_config"`
	PagerDutyConfig *PagerDutyConfig `reform:"pagerduty_config"`
	SlackConfig     *SlackConfig     `reform:"slack_config"`
	WebHookConfig   *WebHookConfig   `reform:"webhook_config"`

	Disabled bool `reform:"disabled"`

	CreatedAt time.Time `reform:"created_at"`
	UpdatedAt time.Time `reform:"updated_at"`
}

// BeforeInsert implements reform.BeforeInserter interface.
func (c *Channel) BeforeInsert() error {
	now := Now()
	c.CreatedAt = now
	c.UpdatedAt = now

	return nil
}

// BeforeUpdate implements reform.BeforeUpdater interface.
func (c *Channel) BeforeUpdate() error {
	c.UpdatedAt = Now()

	return nil
}

// AfterFind implements reform.AfterFinder interface.
func (c *Channel) AfterFind() error {
	c.CreatedAt = c.CreatedAt.UTC()
	c.UpdatedAt = c.UpdatedAt.UTC()

	return nil
}

// EmailConfig is email notification channel configuration.
type EmailConfig struct {
	SendResolved bool     `json:"send_resolved"`
	To           []string `json:"to"`
}

// Value implements database/sql/driver.Valuer interface. Should be defined on the value.
func (c EmailConfig) Value() (driver.Value, error) { return jsonValue(c) }

// Scan implements database/sql.Scanner interface. Should be defined on the pointer.
func (c *EmailConfig) Scan(src interface{}) error { return jsonScan(c, src) }

// PagerDutyConfig represents PagerDuty channel configuration.
type PagerDutyConfig struct {
	SendResolved bool   `json:"send_resolved"`
	RoutingKey   string `json:"routing_key,omitempty"`
	ServiceKey   string `json:"service_key,omitempty"`
}

// Value implements database/sql/driver.Valuer interface. Should be defined on the value.
func (c PagerDutyConfig) Value() (driver.Value, error) { return jsonValue(c) }

// Scan implements database/sql.Scanner interface. Should be defined on the pointer.
func (c *PagerDutyConfig) Scan(src interface{}) error { return jsonScan(c, src) }

// SlackConfig is slack notification channel configuration.
type SlackConfig struct {
	SendResolved bool   `json:"send_resolved"`
	Channel      string `json:"channel"`
}

// Value implements database/sql/driver.Valuer interface. Should be defined on the value.
func (c SlackConfig) Value() (driver.Value, error) { return jsonValue(c) }

// Scan implements database/sql.Scanner interface. Should be defined on the pointer.
func (c *SlackConfig) Scan(src interface{}) error { return jsonScan(c, src) }

// WebHookConfig is webhook notification channel configuration.
type WebHookConfig struct {
	SendResolved bool        `json:"send_resolved"`
	URL          string      `json:"url"`
	HTTPConfig   *HTTPConfig `json:"http_config,omitempty"`
	MaxAlerts    int32       `json:"max_alerts"`
}

// Value implements database/sql/driver.Valuer interface. Should be defined on the value.
func (c WebHookConfig) Value() (driver.Value, error) { return jsonValue(c) }

// Scan implements database/sql.Scanner interface. Should be defined on the pointer.
func (c *WebHookConfig) Scan(src interface{}) error { return jsonScan(c, src) }

// HTTPConfig is HTTP connection configuration.
type HTTPConfig struct {
	BasicAuth       *HTTPBasicAuth `json:"basic_auth,omitempty"`
	BearerToken     string         `json:"bearer_token,omitempty"`
	BearerTokenFile string         `json:"bearer_token_file,omitempty"`
	TLSConfig       *TLSConfig     `json:"tls_config,omitempty"`
	ProxyURL        string         `json:"proxy_url,omitempty"`
}

// HTTPBasicAuth is HTTP basic authentication configuration.
type HTTPBasicAuth struct {
	Username     string `json:"username,omitempty"`
	Password     string `json:"password,omitempty"`
	PasswordFile string `json:"password_file,omitempty"`
}

// TLSConfig is TLS configuration.
type TLSConfig struct {
	CAFile             string `json:"ca_file,omitempty"`
	CertFile           string `json:"cert_file,omitempty"`
	KeyFile            string `json:"key_file,omitempty"`
	ServerName         string `json:"server_name,omitempty"`
	InsecureSkipVerify bool   `json:"insecure_skip_verify,omitempty"`
	CAFileContent      string `json:"ca_file_content,omitempty"`
	CertFileContent    string `json:"cert_file_content,omitempty"`
	KeyFileContent     string `json:"key_file_content,omitempty"`
}

// check interfaces.
var (
	_ reform.BeforeInserter = (*Channel)(nil)
	_ reform.BeforeUpdater  = (*Channel)(nil)
	_ reform.AfterFinder    = (*Channel)(nil)
)
