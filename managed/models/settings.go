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
	"net/url"
	"strings"
	"time"

	"github.com/AlekSi/pointer"
)

// Default values for settings. These values are used when settings are not set.
const (
	AdvisorsEnabledDefault                    = true
	AlertingEnabledDefault                    = true
	NativeQanEnabledDefault                   = false
	TelemetryEnabledDefault                   = true
	UpdatesEnabledDefault                     = true
	BackupManagementEnabledDefault            = true
	VictoriaMetricsCacheEnabledDefault        = false
	AzureDiscoverEnabledDefault               = false
	AccessControlEnabledDefault               = false
	InternalPgQANEnabledDefault               = false
	OtelCollectorEnabledDefault               = true
	OtelLogsRetentionDaysDefault              = 7
	OtelTracesRetentionDaysDefault            = 7
	OtelClickHouseMetricsRetentionDaysDefault = 90
	AdreEnabledDefault                        = false
	AdrePromptMaxBytes                        = 16 * 1024
	AdrePromptMaxBytesHardMax                 = 64 * 1024
	// AdreChatRetentionDaysDefault is the automatic ADRE chat retention when unset in settings (days).
	AdreChatRetentionDaysDefault = 365
	// AdreSchemaVersionCurrent is bumped when a one-way ADRE settings migration runs in fillDefaults.
	AdreSchemaVersionCurrent = 2
	awsPartitionID           = "aws"
)

// MetricsResolutions contains standard VictoriaMetrics metrics resolutions.
type MetricsResolutions struct {
	HR time.Duration `json:"hr"`
	MR time.Duration `json:"mr"`
	LR time.Duration `json:"lr"`
}

// Value implements database/sql/driver.Valuer interface. Should be defined on the value.
func (r MetricsResolutions) Value() (driver.Value, error) { return jsonValue(r) }

// Scan implements database/sql.Scanner interface. Should be defined on the pointer.
func (r *MetricsResolutions) Scan(src any) error { return jsonScan(r, src) }

// Advisors contains settings related to the Portal Advisors.
type Advisors struct {
	// Advisor checks disabled, false by default.
	Enabled *bool `json:"enabled"`
	// List of disabled advisors
	DisabledAdvisors []string `json:"disabled_advisors"`
	// Advisor run intervals
	AdvisorRunIntervals AdvisorsRunIntervals `json:"advisor_run_intervals"`
}

// Settings contains PMM Server settings.
type Settings struct {
	PMMPublicAddress string `json:"pmm_public_address"`

	Updates struct {
		Enabled        *bool         `json:"enabled"`
		SnoozeDuration time.Duration `json:"snooze_duration"`
	} `json:"updates"`

	Telemetry struct {
		Enabled *bool  `json:"enabled"`
		UUID    string `json:"uuid"`
	} `json:"telemetry"`

	MetricsResolutions MetricsResolutions `json:"metrics_resolutions"`

	DataRetention time.Duration `json:"data_retention"`

	AWSPartitions []string `json:"aws_partitions"`

	AWSInstanceChecked bool `json:"aws_instance_checked"`

	SSHKey string `json:"ssh_key"`

	VictoriaMetrics struct {
		CacheEnabled *bool `json:"cache_enabled"`
	} `json:"victoria_metrics"`

	SaaS Advisors `json:"sass"` // sic :(

	Nomad struct {
		Enabled *bool `json:"enabled"`
	}

	// Otel collector on server (receives OTLP from agents); retention for OTEL ClickHouse tables.
	Otel struct {
		CollectorEnabled  *bool `json:"collector_enabled"`
		LogsRetentionDays *int  `json:"logs_retention_days"`
		// TracesRetentionDays is TTL for otel.otel_traces (and trace TTL in server otel-collector exporters).
		TracesRetentionDays *int `json:"traces_retention_days"`
		// MetricsRetentionDays is TTL for otel.otel_metrics_sum (and sum-metric TTL in server otel-collector exporters).
		MetricsRetentionDays *int `json:"metrics_retention_days"`
	} `json:"otel"`

	// Adre (Autonomous Database Reliability Engineer) / HolmesGPT integration.
	Adre struct {
		Enabled             *bool  `json:"enabled"`
		URL                 string `json:"url"`
		ChatPrompt          string `json:"chat_prompt"`
		InvestigationPrompt string `json:"investigation_prompt"`
		// ChatModel is default Holmes model for fast/chat mode. Empty uses Holmes default model.
		ChatModel string `json:"chat_model"`
		// InvestigationModel is default Holmes model for investigation mode. Empty uses Holmes default model.
		InvestigationModel string `json:"investigation_model"`
		// DefaultChatMode: "fast" or "investigation" (empty defaults to "investigation"; legacy "chat" mapped to "fast" in fillDefaults).
		DefaultChatMode string `json:"default_chat_mode"`
		// BehaviorControlsFast / Investigation / FormatReport are Holmes behavior_controls maps
		// (see Holmes HTTP API). Empty map uses PMM shipped presets when sending to Holmes.
		BehaviorControlsFast          map[string]bool `json:"behavior_controls_fast,omitempty"`
		BehaviorControlsInvestigation map[string]bool `json:"behavior_controls_investigation,omitempty"`
		BehaviorControlsFormatReport  map[string]bool `json:"behavior_controls_format_report,omitempty"`
		// AdreMaxConversationMessages caps messages sent in conversation_history to Holmes (0 = default 40).
		AdreMaxConversationMessages int `json:"adre_max_conversation_messages"`
		// AdreSchemaVersion bumps when a one-way settings migration runs (e.g. prompt reset).
		AdreSchemaVersion int `json:"adre_schema_version"`
		// QanInsightsPrompt is the system prompt for QAN AI Insights (query analytics and optimization). Empty = use built-in default.
		QanInsightsPrompt string `json:"qan_insights_prompt"`
		// QanInsightsModel is default Holmes model for QAN AI Insights. Empty uses Holmes default model.
		QanInsightsModel string `json:"qan_insights_model"`
		// ServiceNow integration. The URL is non-secret and stays here; the API key and client token
		// are secrets stored encrypted in the adre_provisioning table (not in this JSONB blob).
		ServiceNowURL string `json:"servicenow_url"`
		// PromptMaxBytes defines max prompt size for ADRE prompts (bytes).
		PromptMaxBytes int `json:"prompt_max_bytes"`
		// AdreChatRetentionDays deletes ADRE chat threads with last_message_at older than this many days (0 = never auto-purge). Nil in JSON defaults in fillDefaults.
		AdreChatRetentionDays *int `json:"adre_chat_retention_days"`
		// Slack integration (Socket Mode). Enable/auto-investigate flags are non-secret and stay here;
		// the bot and app tokens are secrets stored encrypted in the adre_provisioning table.
		SlackEnabled         bool `json:"slack_enabled"`
		SlackAutoInvestigate bool `json:"slack_auto_investigate"`
		// TLSSkipVerify disables TLS certificate verification for PMM → HolmesGPT.
		TLSSkipVerify bool `json:"tls_skip_verify"`
		// Slack human-chat authorization allowlists (fail-closed: empty ⇒ nobody allowed). Values are
		// Slack object IDs (channels Cxxxx/Gxxxx/Dxxxx, users Uxxxx/Wxxxx), not names.
		SlackAllowedChannels []string `json:"slack_allowed_channels,omitempty"`
		SlackAllowedUsers    []string `json:"slack_allowed_users,omitempty"`
		// SlackAutoInvestigateChannels are the alert channels: the bot scrapes Grafana alert messages
		// posted here to trigger an auto-investigation, and posts the investigation thread back into them.
		// They also receive summaries from the webhook/poll fallback path. Fail-closed for the scrape:
		// only messages in these channels are considered.
		SlackAutoInvestigateChannels []string `json:"slack_auto_investigate_channels,omitempty"`
		// SlackAlertBotIDs optionally restricts which Slack bot/app IDs (Bxxxx) the scrape accepts alert
		// messages from (e.g. the Grafana Slack app). Empty ⇒ accept any bot message in the alert
		// channels (looser; set this to harden against spoofed alerts).
		SlackAlertBotIDs []string `json:"slack_alert_bot_ids,omitempty"`
		// Auto-investigate selection + cost guards. AutoInvestigateMinSeverity is a floor (e.g.
		// "critical"; empty ⇒ no floor). AutoInvestigateLabelMatchers are "key=value" pairs an alert
		// must match (all of them). AutoInvestigateHourlyCap bounds auto-investigations per hour
		// (0 ⇒ unbounded, not recommended).
		AutoInvestigateMinSeverity   string   `json:"auto_investigate_min_severity,omitempty"`
		AutoInvestigateLabelMatchers []string `json:"auto_investigate_label_matchers,omitempty"`
		AutoInvestigateHourlyCap     int      `json:"auto_investigate_hourly_cap,omitempty"`
	} `json:"adre"`

	Alerting struct {
		Enabled *bool `json:"enabled"`
	} `json:"alerting"`

	NativeQan struct {
		Enabled *bool `json:"enabled"`
	} `json:"native_qan"`

	Azurediscover struct {
		Enabled *bool `json:"enabled"`
	} `json:"azure"`

	BackupManagement struct {
		Enabled *bool `json:"enabled"`
	} `json:"backup_management"`

	// PMMServerID is generated on the first start of PMM server.
	PMMServerID string `json:"pmmServerID"`

	// DefaultRoleID defines a default role to be assigned to new users.
	DefaultRoleID int `json:"default_role_id"`

	// AccessControl holds information about access control.
	AccessControl struct {
		// Enabled is true if access control is enabled.
		Enabled *bool `json:"enabled"`
	} `json:"access_control"`

	// Contains all encrypted tables in format 'db.table.column'.
	EncryptedItems []string `json:"encrypted_items"`
}

// IsAlertingEnabled returns true if alerting is enabled.
func (s *Settings) IsAlertingEnabled() bool {
	if s.Alerting.Enabled != nil {
		return *s.Alerting.Enabled
	}
	return AlertingEnabledDefault
}

// IsNativeQanEnabled returns true if native Query Analytics UI is enabled.
func (s *Settings) IsNativeQanEnabled() bool {
	if s.NativeQan.Enabled != nil {
		return *s.NativeQan.Enabled
	}
	return NativeQanEnabledDefault
}

// IsTelemetryEnabled returns true if telemetry is enabled.
func (s *Settings) IsTelemetryEnabled() bool {
	if s.Telemetry.Enabled != nil {
		return *s.Telemetry.Enabled
	}
	return TelemetryEnabledDefault
}

// IsUpdatesEnabled returns true if updates are enabled.
func (s *Settings) IsUpdatesEnabled() bool {
	if s.Updates.Enabled != nil {
		return *s.Updates.Enabled
	}
	return UpdatesEnabledDefault
}

// IsBackupManagementEnabled returns true if backup management is enabled.
func (s *Settings) IsBackupManagementEnabled() bool {
	if s.BackupManagement.Enabled != nil {
		return *s.BackupManagement.Enabled
	}
	return BackupManagementEnabledDefault
}

// IsAdvisorsEnabled returns true if advisors are enabled.
func (s *Settings) IsAdvisorsEnabled() bool {
	if s.SaaS.Enabled != nil {
		return *s.SaaS.Enabled
	}

	return AdvisorsEnabledDefault
}

// IsNomadEnabled returns true if Nomad is enabled.
func (s *Settings) IsNomadEnabled() bool {
	return pointer.GetBool(s.Nomad.Enabled) && s.PMMPublicAddress != ""
}

// IsAzureDiscoverEnabled returns true if Azure discovery is enabled.
func (s *Settings) IsAzureDiscoverEnabled() bool {
	if s.Azurediscover.Enabled != nil {
		return *s.Azurediscover.Enabled
	}
	return AzureDiscoverEnabledDefault
}

// IsAccessControlEnabled returns true if access control is enabled.
func (s *Settings) IsAccessControlEnabled() bool {
	if s.AccessControl.Enabled != nil {
		return *s.AccessControl.Enabled
	}
	return AccessControlEnabledDefault
}

// IsVictoriaMetricsCacheEnabled returns true if VictoriaMetrics cache is enabled.
func (s *Settings) IsVictoriaMetricsCacheEnabled() bool {
	if s.VictoriaMetrics.CacheEnabled != nil {
		return *s.VictoriaMetrics.CacheEnabled
	}
	return VictoriaMetricsCacheEnabledDefault
}

// IsOtelCollectorEnabled returns true if the OTEL collector on the server is enabled.
func (s *Settings) IsOtelCollectorEnabled() bool {
	if s.Otel.CollectorEnabled != nil {
		return *s.Otel.CollectorEnabled
	}
	return OtelCollectorEnabledDefault
}

// GetOtelLogsRetentionDays returns the TTL in days for otel.logs in ClickHouse.
func (s *Settings) GetOtelLogsRetentionDays() int {
	if s.Otel.LogsRetentionDays != nil && *s.Otel.LogsRetentionDays > 0 {
		return *s.Otel.LogsRetentionDays
	}
	return OtelLogsRetentionDaysDefault
}

// IsAdreEnabled returns true if ADRE (HolmesGPT) integration is enabled.
func (s *Settings) IsAdreEnabled() bool {
	if s.Adre.Enabled != nil {
		return *s.Adre.Enabled
	}
	return AdreEnabledDefault
}

// GetAdreURL returns the HolmesGPT base URL, or empty if disabled or not set.
func (s *Settings) GetAdreURL() string {
	if !s.IsAdreEnabled() || s.Adre.URL == "" {
		return ""
	}
	return s.Adre.URL
}

// NormalizePMMPublicAddressOrigin parses PMM "Public address" (host:port or full URL) into an http(s) origin without trailing slash, or "" if unset/invalid.
func NormalizePMMPublicAddressOrigin(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	addr := raw
	if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
		addr = "https://" + addr
	}
	u, err := url.Parse(addr)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	u.Path = ""
	u.RawQuery = ""
	u.Fragment = ""
	return strings.TrimSuffix(u.String(), "/")
}

// GetEffectiveSlackLinkBaseURL returns the base URL for rewriting relative PMM paths in Slack (no trailing slash).
// Uses PMM **Public address** from Advanced settings (see NormalizePMMPublicAddressOrigin).
func (s *Settings) GetEffectiveSlackLinkBaseURL() string {
	if s == nil {
		return ""
	}
	return NormalizePMMPublicAddressOrigin(s.PMMPublicAddress)
}

// slackListContains reports whether id (trimmed, exact, case-sensitive — Slack IDs are uppercase
// tokens) is present in list. Fail-closed: an empty list contains nothing.
func slackListContains(list []string, id string) bool {
	id = strings.TrimSpace(id)
	if id == "" {
		return false
	}
	for _, v := range list {
		if strings.TrimSpace(v) == id {
			return true
		}
	}
	return false
}

// IsSlackChannelAllowed reports whether a channel is allow-listed for human Slack chat (fail-closed).
func (s *Settings) IsSlackChannelAllowed(channelID string) bool {
	return slackListContains(s.Adre.SlackAllowedChannels, channelID)
}

// IsSlackUserAllowed reports whether a user is allow-listed for human Slack chat (fail-closed).
func (s *Settings) IsSlackUserAllowed(userID string) bool {
	return slackListContains(s.Adre.SlackAllowedUsers, userID)
}

// IsSlackAlertChannel reports whether a channel is a configured alert channel — i.e. one the bot
// scrapes for Grafana alert messages and posts investigation threads into (fail-closed).
func (s *Settings) IsSlackAlertChannel(channelID string) bool {
	return slackListContains(s.Adre.SlackAutoInvestigateChannels, channelID)
}

// IsSlackAlertBot reports whether a bot ID is an accepted alert sender. The SlackAlertBotIDs list is an
// optional filter: empty ⇒ accept any bot in the alert channels; non-empty ⇒ the bot must match.
func (s *Settings) IsSlackAlertBot(botID string) bool {
	if len(s.Adre.SlackAlertBotIDs) == 0 {
		return true
	}
	return slackListContains(s.Adre.SlackAlertBotIDs, botID)
}

// GetAdreChatRetentionDays returns automatic ADRE chat retention in days (0 = no automatic purge).
func (s *Settings) GetAdreChatRetentionDays() int {
	if s.Adre.AdreChatRetentionDays != nil {
		return *s.Adre.AdreChatRetentionDays
	}
	return AdreChatRetentionDaysDefault
}

// GetOtelTracesRetentionDays returns the TTL in days for otel.otel_traces in ClickHouse.
func (s *Settings) GetOtelTracesRetentionDays() int {
	if s.Otel.TracesRetentionDays != nil && *s.Otel.TracesRetentionDays > 0 {
		return *s.Otel.TracesRetentionDays
	}
	return OtelTracesRetentionDaysDefault
}

// GetOtelMetricsRetentionDays returns the TTL in days for otel.otel_metrics_sum in ClickHouse.
func (s *Settings) GetOtelMetricsRetentionDays() int {
	if s.Otel.MetricsRetentionDays != nil && *s.Otel.MetricsRetentionDays > 0 {
		return *s.Otel.MetricsRetentionDays
	}
	return OtelClickHouseMetricsRetentionDaysDefault
}

// AdvisorsRunIntervals represents intervals between Advisors checks.
type AdvisorsRunIntervals struct {
	StandardInterval time.Duration `json:"standard_interval"`
	RareInterval     time.Duration `json:"rare_interval"`
	FrequentInterval time.Duration `json:"frequent_interval"`
}

// fillDefaults sets zero values to their default values.
// Used for migrating settings to the newer version.
func (s *Settings) fillDefaults() {
	// no default for Telemetry UUID - it set by telemetry service

	if s.MetricsResolutions.HR == 0 {
		s.MetricsResolutions.HR = 5 * time.Second //nolint:mnd
	}
	if s.MetricsResolutions.MR == 0 {
		s.MetricsResolutions.MR = 10 * time.Second //nolint:mnd
	}
	if s.MetricsResolutions.LR == 0 {
		s.MetricsResolutions.LR = 60 * time.Second //nolint:mnd
	}

	if s.DataRetention == 0 {
		s.DataRetention = 30 * 24 * time.Hour //nolint:mnd
	}

	if len(s.AWSPartitions) == 0 {
		s.AWSPartitions = []string{awsPartitionID}
	}

	if s.SaaS.AdvisorRunIntervals.RareInterval == 0 {
		s.SaaS.AdvisorRunIntervals.RareInterval = 78 * time.Hour //nolint:mnd
	}

	if s.SaaS.AdvisorRunIntervals.StandardInterval == 0 {
		s.SaaS.AdvisorRunIntervals.StandardInterval = 24 * time.Hour //nolint:mnd
	}

	if s.SaaS.AdvisorRunIntervals.FrequentInterval == 0 {
		s.SaaS.AdvisorRunIntervals.FrequentInterval = 4 * time.Hour //nolint:mnd
	}

	if s.Updates.SnoozeDuration == 0 {
		s.Updates.SnoozeDuration = DefaultSnoozeDuration
	}

	if s.Otel.LogsRetentionDays == nil || (s.Otel.LogsRetentionDays != nil && *s.Otel.LogsRetentionDays <= 0) {
		s.Otel.LogsRetentionDays = new(OtelLogsRetentionDaysDefault)
	}
	if s.Otel.TracesRetentionDays == nil || (s.Otel.TracesRetentionDays != nil && *s.Otel.TracesRetentionDays <= 0) {
		s.Otel.TracesRetentionDays = new(OtelTracesRetentionDaysDefault)
	}
	if s.Otel.MetricsRetentionDays == nil || (s.Otel.MetricsRetentionDays != nil && *s.Otel.MetricsRetentionDays <= 0) {
		s.Otel.MetricsRetentionDays = new(OtelClickHouseMetricsRetentionDaysDefault)
	}

	if s.Adre.Enabled == nil {
		s.Adre.Enabled = new(AdreEnabledDefault)
	}
	if s.Adre.AdreSchemaVersion < AdreSchemaVersionCurrent {
		// One-way migration: new behavior_controls model, Fast/Investigation prompts reset to built-in defaults (empty = use code defaults).
		s.Adre.ChatPrompt = ""
		s.Adre.InvestigationPrompt = ""
		if s.Adre.DefaultChatMode == "" || s.Adre.DefaultChatMode == "chat" {
			s.Adre.DefaultChatMode = "investigation"
		}
		s.Adre.BehaviorControlsFast = map[string]bool{
			"time_skills":            false,
			"todowrite_instructions": false,
			"todowrite_reminder":     false,
		}
		s.Adre.BehaviorControlsFormatReport = map[string]bool{
			"time_skills":            false,
			"todowrite_instructions": false,
			"todowrite_reminder":     false,
		}
		s.Adre.BehaviorControlsInvestigation = nil
		if s.Adre.AdreMaxConversationMessages <= 0 {
			s.Adre.AdreMaxConversationMessages = 40
		}
		s.Adre.AdreSchemaVersion = AdreSchemaVersionCurrent
	}
	if s.Adre.DefaultChatMode == "" {
		s.Adre.DefaultChatMode = "investigation"
	}
	if s.Adre.DefaultChatMode == "chat" {
		s.Adre.DefaultChatMode = "fast"
	}
	if s.Adre.AdreMaxConversationMessages <= 0 {
		s.Adre.AdreMaxConversationMessages = 40
	}
	if s.Adre.AdreChatRetentionDays == nil {
		s.Adre.AdreChatRetentionDays = new(AdreChatRetentionDaysDefault)
	}
}
