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
	VSendResolved bool `yaml:"send_resolved" json:"send_resolved"`
}

// EmailConfig configures notifications via mail.
type EmailConfig struct {
	NotifierConfig `yaml:",inline" json:",inline"`

	// Email address to notify.
	To           string               `yaml:"to,omitempty" json:"to,omitempty"`
	From         string               `yaml:"from,omitempty" json:"from,omitempty"`
	Hello        string               `yaml:"hello,omitempty" json:"hello,omitempty"`
	Smarthost    HostPort             `yaml:"smarthost,omitempty" json:"smarthost,omitempty"`
	AuthUsername string               `yaml:"auth_username,omitempty" json:"auth_username,omitempty"`
	AuthPassword string               `yaml:"auth_password,omitempty" json:"auth_password,omitempty"`
	AuthSecret   string               `yaml:"auth_secret,omitempty" json:"auth_secret,omitempty"`
	AuthIdentity string               `yaml:"auth_identity,omitempty" json:"auth_identity,omitempty"`
	Headers      map[string]string    `yaml:"headers,omitempty" json:"headers,omitempty"`
	HTML         string               `yaml:"html,omitempty" json:"html,omitempty"`
	Text         string               `yaml:"text,omitempty" json:"text,omitempty"`
	RequireTLS   *bool                `yaml:"require_tls,omitempty" json:"require_tls,omitempty"`
	TLSConfig    promconfig.TLSConfig `yaml:"tls_config,omitempty" json:"tls_config,omitempty"`
}

// PagerdutyConfig configures notifications via PagerDuty.
type PagerdutyConfig struct {
	NotifierConfig `yaml:",inline" json:",inline"`

	HTTPConfig promconfig.HTTPClientConfig `yaml:"http_config,omitempty" json:"http_config,omitempty"`

	ServiceKey  string            `yaml:"service_key,omitempty" json:"service_key,omitempty"`
	RoutingKey  string            `yaml:"routing_key,omitempty" json:"routing_key,omitempty"`
	URL         string            `yaml:"url,omitempty" json:"url,omitempty"`
	Client      string            `yaml:"client,omitempty" json:"client,omitempty"`
	ClientURL   string            `yaml:"client_url,omitempty" json:"client_url,omitempty"`
	Description string            `yaml:"description,omitempty" json:"description,omitempty"`
	Details     map[string]string `yaml:"details,omitempty" json:"details,omitempty"`
	Images      []PagerdutyImage  `yaml:"images,omitempty" json:"images,omitempty"`
	Links       []PagerdutyLink   `yaml:"links,omitempty" json:"links,omitempty"`
	Severity    string            `yaml:"severity,omitempty" json:"severity,omitempty"`
	Class       string            `yaml:"class,omitempty" json:"class,omitempty"`
	Component   string            `yaml:"component,omitempty" json:"component,omitempty"`
	Group       string            `yaml:"group,omitempty" json:"group,omitempty"`
}

// PagerdutyLink is a link
type PagerdutyLink struct {
	Href string `yaml:"href,omitempty" json:"href,omitempty"`
	Text string `yaml:"text,omitempty" json:"text,omitempty"`
}

// PagerdutyImage is an image
type PagerdutyImage struct {
	Src  string `yaml:"src,omitempty" json:"src,omitempty"`
	Alt  string `yaml:"alt,omitempty" json:"alt,omitempty"`
	Href string `yaml:"href,omitempty" json:"href,omitempty"`
}

// SlackAction configures a single Slack action that is sent with each notification.
// See https://api.slack.com/docs/message-attachments#action_fields and https://api.slack.com/docs/message-buttons
// for more information.
type SlackAction struct {
	Type         string                  `yaml:"type,omitempty"  json:"type,omitempty"`
	Text         string                  `yaml:"text,omitempty"  json:"text,omitempty"`
	URL          string                  `yaml:"url,omitempty"   json:"url,omitempty"`
	Style        string                  `yaml:"style,omitempty" json:"style,omitempty"`
	Name         string                  `yaml:"name,omitempty"  json:"name,omitempty"`
	Value        string                  `yaml:"value,omitempty"  json:"value,omitempty"`
	ConfirmField *SlackConfirmationField `yaml:"confirm,omitempty"  json:"confirm,omitempty"`
}

// SlackConfirmationField protect users from destructive actions or particularly distinguished decisions
// by asking them to confirm their button click one more time.
// See https://api.slack.com/docs/interactive-message-field-guide#confirmation_fields for more information.
type SlackConfirmationField struct {
	Text        string `yaml:"text,omitempty"  json:"text,omitempty"`
	Title       string `yaml:"title,omitempty"  json:"title,omitempty"`
	OkText      string `yaml:"ok_text,omitempty"  json:"ok_text,omitempty"`
	DismissText string `yaml:"dismiss_text,omitempty"  json:"dismiss_text,omitempty"`
}

// SlackField configures a single Slack field that is sent with each notification.
// Each field must contain a title, value, and optionally, a boolean value to indicate if the field
// is short enough to be displayed next to other fields designated as short.
// See https://api.slack.com/docs/message-attachments#fields for more information.
type SlackField struct {
	Title string `yaml:"title,omitempty" json:"title,omitempty"`
	Value string `yaml:"value,omitempty" json:"value,omitempty"`
	Short *bool  `yaml:"short,omitempty" json:"short,omitempty"`
}

// SlackConfig configures notifications via Slack.
type SlackConfig struct {
	NotifierConfig `yaml:",inline" json:",inline"`

	HTTPConfig promconfig.HTTPClientConfig `yaml:"http_config,omitempty" json:"http_config,omitempty"`

	APIURL *string `yaml:"api_url,omitempty" json:"api_url,omitempty"`

	// Slack channel override, (like #other-channel or @username).
	Channel  string `yaml:"channel,omitempty" json:"channel,omitempty"`
	Username string `yaml:"username,omitempty" json:"username,omitempty"`
	Color    string `yaml:"color,omitempty" json:"color,omitempty"`

	Title       string         `yaml:"title,omitempty" json:"title,omitempty"`
	TitleLink   string         `yaml:"title_link,omitempty" json:"title_link,omitempty"`
	Pretext     string         `yaml:"pretext,omitempty" json:"pretext,omitempty"`
	Text        string         `yaml:"text,omitempty" json:"text,omitempty"`
	Fields      []*SlackField  `yaml:"fields,omitempty" json:"fields,omitempty"`
	ShortFields bool           `yaml:"short_fields" json:"short_fields,omitempty"`
	Footer      string         `yaml:"footer,omitempty" json:"footer,omitempty"`
	Fallback    string         `yaml:"fallback,omitempty" json:"fallback,omitempty"`
	CallbackID  string         `yaml:"callback_id,omitempty" json:"callback_id,omitempty"`
	IconEmoji   string         `yaml:"icon_emoji,omitempty" json:"icon_emoji,omitempty"`
	IconURL     string         `yaml:"icon_url,omitempty" json:"icon_url,omitempty"`
	ImageURL    string         `yaml:"image_url,omitempty" json:"image_url,omitempty"`
	ThumbURL    string         `yaml:"thumb_url,omitempty" json:"thumb_url,omitempty"`
	LinkNames   bool           `yaml:"link_names" json:"link_names,omitempty"`
	MrkdwnIn    []string       `yaml:"mrkdwn_in,omitempty" json:"mrkdwn_in,omitempty"`
	Actions     []*SlackAction `yaml:"actions,omitempty" json:"actions,omitempty"`
}

// WebhookConfig configures notifications via a generic webhook.
type WebhookConfig struct {
	NotifierConfig `yaml:",inline" json:",inline"`

	HTTPConfig promconfig.HTTPClientConfig `yaml:"http_config,omitempty" json:"http_config,omitempty"`

	// URL to send POST request to.
	URL string `yaml:"url" json:"url"`
	// MaxAlerts is the maximum number of alerts to be sent per webhook message.
	// Alerts exceeding this threshold will be truncated. Setting this to 0
	// allows an unlimited number of alerts.
	MaxAlerts uint64 `yaml:"max_alerts" json:"max_alerts"`
}

// WechatConfig configures notifications via Wechat.
type WechatConfig struct {
	NotifierConfig `yaml:",inline" json:",inline"`

	HTTPConfig promconfig.HTTPClientConfig `yaml:"http_config,omitempty" json:"http_config,omitempty"`

	APISecret   string `yaml:"api_secret,omitempty" json:"api_secret,omitempty"`
	CorpID      string `yaml:"corp_id,omitempty" json:"corp_id,omitempty"`
	Message     string `yaml:"message,omitempty" json:"message,omitempty"`
	APIURL      string `yaml:"api_url,omitempty" json:"api_url,omitempty"`
	ToUser      string `yaml:"to_user,omitempty" json:"to_user,omitempty"`
	ToParty     string `yaml:"to_party,omitempty" json:"to_party,omitempty"`
	ToTag       string `yaml:"to_tag,omitempty" json:"to_tag,omitempty"`
	AgentID     string `yaml:"agent_id,omitempty" json:"agent_id,omitempty"`
	MessageType string `yaml:"message_type,omitempty" json:"message_type,omitempty"`
}

// OpsGenieConfig configures notifications via OpsGenie.
type OpsGenieConfig struct {
	NotifierConfig `yaml:",inline" json:",inline"`

	HTTPConfig promconfig.HTTPClientConfig `yaml:"http_config,omitempty" json:"http_config,omitempty"`

	APIKey      string                    `yaml:"api_key,omitempty" json:"api_key,omitempty"`
	APIURL      string                    `yaml:"api_url,omitempty" json:"api_url,omitempty"`
	Message     string                    `yaml:"message,omitempty" json:"message,omitempty"`
	Description string                    `yaml:"description,omitempty" json:"description,omitempty"`
	Source      string                    `yaml:"source,omitempty" json:"source,omitempty"`
	Details     map[string]string         `yaml:"details,omitempty" json:"details,omitempty"`
	Responders  []OpsGenieConfigResponder `yaml:"responders,omitempty" json:"responders,omitempty"`
	Tags        string                    `yaml:"tags,omitempty" json:"tags,omitempty"`
	Note        string                    `yaml:"note,omitempty" json:"note,omitempty"`
	Priority    string                    `yaml:"priority,omitempty" json:"priority,omitempty"`
}

type OpsGenieConfigResponder struct {
	// One of those 3 should be filled.
	ID       string `yaml:"id,omitempty" json:"id,omitempty"`
	Name     string `yaml:"name,omitempty" json:"name,omitempty"`
	Username string `yaml:"username,omitempty" json:"username,omitempty"`

	// team, user, escalation, schedule etc.
	Type string `yaml:"type,omitempty" json:"type,omitempty"`
}

// VictorOpsConfig configures notifications via VictorOps.
type VictorOpsConfig struct {
	NotifierConfig `yaml:",inline" json:",inline"`

	HTTPConfig promconfig.HTTPClientConfig `yaml:"http_config,omitempty" json:"http_config,omitempty"`

	APIKey            string            `yaml:"api_key" json:"api_key"`
	APIURL            string            `yaml:"api_url" json:"api_url"`
	RoutingKey        string            `yaml:"routing_key" json:"routing_key"`
	MessageType       string            `yaml:"message_type" json:"message_type"`
	StateMessage      string            `yaml:"state_message" json:"state_message"`
	EntityDisplayName string            `yaml:"entity_display_name" json:"entity_display_name"`
	MonitoringTool    string            `yaml:"monitoring_tool" json:"monitoring_tool"`
	CustomFields      map[string]string `yaml:"custom_fields,omitempty" json:"custom_fields,omitempty"`
}

type PushoverConfig struct {
	NotifierConfig `yaml:",inline" json:",inline"`

	HTTPConfig promconfig.HTTPClientConfig `yaml:"http_config,omitempty" json:"http_config,omitempty"`

	UserKey  string        `yaml:"user_key,omitempty" json:"user_key,omitempty"`
	Token    string        `yaml:"token,omitempty" json:"token,omitempty"`
	Title    string        `yaml:"title,omitempty" json:"title,omitempty"`
	Message  string        `yaml:"message,omitempty" json:"message,omitempty"`
	URL      string        `yaml:"url,omitempty" json:"url,omitempty"`
	URLTitle string        `yaml:"url_title,omitempty" json:"url_title,omitempty"`
	Sound    string        `yaml:"sound,omitempty" json:"sound,omitempty"`
	Priority string        `yaml:"priority,omitempty" json:"priority,omitempty"`
	Retry    time.Duration `yaml:"retry,omitempty" json:"retry,omitempty"`
	Expire   time.Duration `yaml:"expire,omitempty" json:"expire,omitempty"`
	HTML     bool          `yaml:"html" json:"html,omitempty"`
}
