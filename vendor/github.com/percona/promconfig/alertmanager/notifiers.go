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
	"time"

	"github.com/percona/promconfig"
)

// NotifierConfig contains base options common across all notifier configurations.
type NotifierConfig struct {
	SendResolved bool `yaml:"send_resolved"`
}

// EmailConfig configures notifications via mail.
type EmailConfig struct {
	NotifierConfig `yaml:",inline"`

	// Email address to notify.
	To           string               `yaml:"to,omitempty"`
	From         string               `yaml:"from,omitempty"`
	Hello        string               `yaml:"hello,omitempty"`
	Smarthost    HostPort             `yaml:"smarthost,omitempty"`
	AuthUsername string               `yaml:"auth_username,omitempty"`
	AuthPassword string               `yaml:"auth_password,omitempty"`
	AuthSecret   string               `yaml:"auth_secret,omitempty"`
	AuthIdentity string               `yaml:"auth_identity,omitempty"`
	Headers      map[string]string    `yaml:"headers,omitempty"`
	HTML         string               `yaml:"html,omitempty"`
	Text         string               `yaml:"text,omitempty"`
	RequireTLS   *bool                `yaml:"require_tls,omitempty"`
	TLSConfig    promconfig.TLSConfig `yaml:"tls_config,omitempty"`
}

// PagerdutyConfig configures notifications via PagerDuty.
type PagerdutyConfig struct {
	NotifierConfig `yaml:",inline"`

	HTTPConfig promconfig.HTTPClientConfig `yaml:"http_config,omitempty"`

	ServiceKey  string            `yaml:"service_key,omitempty"`
	RoutingKey  string            `yaml:"routing_key,omitempty"`
	URL         string            `yaml:"url,omitempty"`
	Client      string            `yaml:"client,omitempty"`
	ClientURL   string            `yaml:"client_url,omitempty"`
	Description string            `yaml:"description,omitempty"`
	Details     map[string]string `yaml:"details,omitempty"`
	Images      []PagerdutyImage  `yaml:"images,omitempty"`
	Links       []PagerdutyLink   `yaml:"links,omitempty"`
	Severity    string            `yaml:"severity,omitempty"`
	Class       string            `yaml:"class,omitempty"`
	Component   string            `yaml:"component,omitempty"`
	Group       string            `yaml:"group,omitempty"`
}

// PagerdutyLink is a link
type PagerdutyLink struct {
	Href string `yaml:"href,omitempty"`
	Text string `yaml:"text,omitempty"`
}

// PagerdutyImage is an image
type PagerdutyImage struct {
	Src  string `yaml:"src,omitempty"`
	Alt  string `yaml:"alt,omitempty"`
	Href string `yaml:"href,omitempty"`
}

// SlackAction configures a single Slack action that is sent with each notification.
// See https://api.slack.com/docs/message-attachments#action_fields and https://api.slack.com/docs/message-buttons
// for more information.
type SlackAction struct {
	Type         string                  `yaml:"type,omitempty"`
	Text         string                  `yaml:"text,omitempty"`
	URL          string                  `yaml:"url,omitempty"  `
	Style        string                  `yaml:"style,omitempty"`
	Name         string                  `yaml:"name,omitempty"`
	Value        string                  `yaml:"value,omitempty"`
	ConfirmField *SlackConfirmationField `yaml:"confirm,omitempty"`
}

// SlackConfirmationField protect users from destructive actions or particularly distinguished decisions
// by asking them to confirm their button click one more time.
// See https://api.slack.com/docs/interactive-message-field-guide#confirmation_fields for more information.
type SlackConfirmationField struct {
	Text        string `yaml:"text,omitempty"`
	Title       string `yaml:"title,omitempty"`
	OkText      string `yaml:"ok_text,omitempty"`
	DismissText string `yaml:"dismiss_text,omitempty"`
}

// SlackField configures a single Slack field that is sent with each notification.
// Each field must contain a title, value, and optionally, a boolean value to indicate if the field
// is short enough to be displayed next to other fields designated as short.
// See https://api.slack.com/docs/message-attachments#fields for more information.
type SlackField struct {
	Title string `yaml:"title,omitempty"`
	Value string `yaml:"value,omitempty"`
	Short *bool  `yaml:"short,omitempty"`
}

// SlackConfig configures notifications via Slack.
type SlackConfig struct {
	NotifierConfig `yaml:",inline"`

	HTTPConfig promconfig.HTTPClientConfig `yaml:"http_config,omitempty"`

	APIURL *string `yaml:"api_url,omitempty"`

	// Slack channel override, (like #other-channel or @username).
	Channel  string `yaml:"channel,omitempty"`
	Username string `yaml:"username,omitempty"`
	Color    string `yaml:"color,omitempty"`

	Title       string         `yaml:"title,omitempty"`
	TitleLink   string         `yaml:"title_link,omitempty"`
	Pretext     string         `yaml:"pretext,omitempty"`
	Text        string         `yaml:"text,omitempty"`
	Fields      []*SlackField  `yaml:"fields,omitempty"`
	ShortFields bool           `yaml:"short_fields"`
	Footer      string         `yaml:"footer,omitempty"`
	Fallback    string         `yaml:"fallback,omitempty"`
	CallbackID  string         `yaml:"callback_id,omitempty"`
	IconEmoji   string         `yaml:"icon_emoji,omitempty"`
	IconURL     string         `yaml:"icon_url,omitempty"`
	ImageURL    string         `yaml:"image_url,omitempty"`
	ThumbURL    string         `yaml:"thumb_url,omitempty"`
	LinkNames   bool           `yaml:"link_names"`
	MrkdwnIn    []string       `yaml:"mrkdwn_in,omitempty"`
	Actions     []*SlackAction `yaml:"actions,omitempty"`
}

// WebhookConfig configures notifications via a generic webhook.
type WebhookConfig struct {
	NotifierConfig `yaml:",inline"`

	HTTPConfig promconfig.HTTPClientConfig `yaml:"http_config,omitempty"`

	// URL to send POST request to.
	URL string `yaml:"url"`
	// MaxAlerts is the maximum number of alerts to be sent per webhook message.
	// Alerts exceeding this threshold will be truncated. Setting this to 0
	// allows an unlimited number of alerts.
	MaxAlerts uint64 `yaml:"max_alerts"`
}

// WechatConfig configures notifications via Wechat.
type WechatConfig struct {
	NotifierConfig `yaml:",inline"`

	HTTPConfig promconfig.HTTPClientConfig `yaml:"http_config,omitempty"`

	APISecret   string `yaml:"api_secret,omitempty"`
	CorpID      string `yaml:"corp_id,omitempty"`
	Message     string `yaml:"message,omitempty"`
	APIURL      string `yaml:"api_url,omitempty"`
	ToUser      string `yaml:"to_user,omitempty"`
	ToParty     string `yaml:"to_party,omitempty"`
	ToTag       string `yaml:"to_tag,omitempty"`
	AgentID     string `yaml:"agent_id,omitempty"`
	MessageType string `yaml:"message_type,omitempty"`
}

// OpsGenieConfig configures notifications via OpsGenie.
type OpsGenieConfig struct {
	NotifierConfig `yaml:",inline"`

	HTTPConfig promconfig.HTTPClientConfig `yaml:"http_config,omitempty"`

	APIKey      string                    `yaml:"api_key,omitempty"`
	APIURL      string                    `yaml:"api_url,omitempty"`
	Message     string                    `yaml:"message,omitempty"`
	Description string                    `yaml:"description,omitempty"`
	Source      string                    `yaml:"source,omitempty"`
	Details     map[string]string         `yaml:"details,omitempty"`
	Responders  []OpsGenieConfigResponder `yaml:"responders,omitempty"`
	Tags        string                    `yaml:"tags,omitempty"`
	Note        string                    `yaml:"note,omitempty"`
	Priority    string                    `yaml:"priority,omitempty"`
}

type OpsGenieConfigResponder struct {
	// One of those 3 should be filled.
	ID       string `yaml:"id,omitempty"`
	Name     string `yaml:"name,omitempty"`
	Username string `yaml:"username,omitempty"`

	// team, user, escalation, schedule etc.
	Type string `yaml:"type,omitempty"`
}

// VictorOpsConfig configures notifications via VictorOps.
type VictorOpsConfig struct {
	NotifierConfig `yaml:",inline"`

	HTTPConfig promconfig.HTTPClientConfig `yaml:"http_config,omitempty"`

	APIKey            string            `yaml:"api_key"`
	APIURL            string            `yaml:"api_url"`
	RoutingKey        string            `yaml:"routing_key"`
	MessageType       string            `yaml:"message_type"`
	StateMessage      string            `yaml:"state_message"`
	EntityDisplayName string            `yaml:"entity_display_name"`
	MonitoringTool    string            `yaml:"monitoring_tool"`
	CustomFields      map[string]string `yaml:"custom_fields,omitempty"`
}

type PushoverConfig struct {
	NotifierConfig `yaml:",inline"`

	HTTPConfig promconfig.HTTPClientConfig `yaml:"http_config,omitempty"`

	UserKey  string        `yaml:"user_key,omitempty"`
	Token    string        `yaml:"token,omitempty"`
	Title    string        `yaml:"title,omitempty"`
	Message  string        `yaml:"message,omitempty"`
	URL      string        `yaml:"url,omitempty"`
	URLTitle string        `yaml:"url_title,omitempty"`
	Sound    string        `yaml:"sound,omitempty"`
	Priority string        `yaml:"priority,omitempty"`
	Retry    time.Duration `yaml:"retry,omitempty"`
	Expire   time.Duration `yaml:"expire,omitempty"`
	HTML     bool          `yaml:"html"`
}
