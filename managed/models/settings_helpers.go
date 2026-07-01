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
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"strings"
	"time"
	"unicode"

	"github.com/AlekSi/pointer"
	"github.com/google/uuid"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/utils/validators"
)

// adreBehaviorControlAllowed keys accepted in ADRE behavior_controls maps (keep in sync with adre.KnownBehaviorControlKeys).
var adreBehaviorControlAllowed = map[string]struct{}{
	"intro":                   {},
	"ask_user":                {},
	"todowrite_instructions":  {},
	"todowrite_reminder":      {},
	"ai_safety":               {},
	"toolset_instructions":    {},
	"permission_errors":       {},
	"general_instructions":    {},
	"style_guide":             {},
	"cluster_name":            {},
	"system_prompt_additions": {},
	"files":                   {},
	"time_skills":             {},
	"time_runbooks":           {}, // legacy; PMM maps to time_skills when calling Holmes
}

func validateAdreBehaviorControlsMap(field string, m map[string]bool) error {
	for k := range m {
		if _, ok := adreBehaviorControlAllowed[k]; !ok {
			return fmt.Errorf("%s: unknown behavior_controls key %q", field, k)
		}
	}
	return nil
}

// ErrTxRequired is returned when a transaction is required.
var ErrTxRequired = errors.New("TxRequired")

// GetSettings returns current PMM Server settings.
func GetSettings(q reform.DBTX) (*Settings, error) {
	var b []byte
	err := q.QueryRow("SELECT settings FROM settings").Scan(&b)
	if err != nil {
		return nil, fmt.Errorf("failed to select settings: %w", err)
	}

	var s Settings

	err = json.Unmarshal(b, &s) //nolint:musttag
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
	}
	s.fillDefaults()

	return &s, nil
}

// ChangeSettingsParams contains values to change data in settings table.
type ChangeSettingsParams struct {
	EnableUpdates *bool

	EnableTelemetry *bool

	MetricsResolutions MetricsResolutions

	DataRetention time.Duration

	// List of AWS partitions to use. If empty - default partitions will be used. If nil - no changes will be made.
	AWSPartitions []string

	SSHKey *string

	// Enable Advisors
	EnableAdvisors *bool

	EnableNomad *bool

	// List of Advisor checks to disable
	DisableAdvisorChecks []string
	// List of Advisor checks to enable
	EnableAdvisorChecks []string
	// Advisors run intervals
	AdvisorsRunInterval AdvisorsRunIntervals

	// Enable Azure Discover features.
	EnableAzurediscover *bool
	// Enable Percona Alerting features.
	EnableAlerting *bool

	// Enable native Query Analytics UI (Technical Preview).
	EnableNativeQan *bool

	// Enable Access Control features.
	EnableAccessControl *bool

	// EnableVMCache enables caching for vmdb search queries
	EnableVMCache *bool

	// PMM Server public address.
	PMMPublicAddress *string

	// Enable Backup Management features.
	EnableBackupManagement *bool

	// EnableInternalPgQAN enables Query Analytics for PMM's internal PG database.
	EnableInternalPgQAN *bool

	// DefaultRoleID sets a default role to be assigned to new users.
	DefaultRoleID *int

	// List of items in format 'db.table.column' to be encrypted.
	EncryptedItems []string

	// Duration for which an update is snoozed
	UpdateSnoozeDuration time.Duration

	// EnableAdre enables the ADRE (HolmesGPT) integration.
	EnableAdre *bool
	// AdreURL is the HolmesGPT base URL (e.g. http://holmesgpt:8080).
	AdreURL *string
	// AdreTLSSkipVerify disables TLS certificate verification for PMM → HolmesGPT.
	AdreTLSSkipVerify *bool
	// AdreChatPrompt is the system prompt for chat (fast) mode. Max 4096 bytes.
	AdreChatPrompt *string
	// AdreInvestigationPrompt is the system prompt for investigation mode. Max 4096 bytes.
	AdreInvestigationPrompt *string
	// AdreChatModel is default Holmes model alias for fast mode chat. Empty uses Holmes default.
	AdreChatModel *string
	// AdreInvestigationModel is default Holmes model alias for investigation mode chat. Empty uses Holmes default.
	AdreInvestigationModel *string
	// AdreDefaultChatMode is the default mode when UI does not send one: "fast" or "investigation".
	AdreDefaultChatMode *string
	// AdreBehaviorControlsFast / Investigation / FormatReport: Holmes behavior_controls maps. Nil = no change.
	AdreBehaviorControlsFast          *map[string]bool
	AdreBehaviorControlsInvestigation *map[string]bool
	AdreBehaviorControlsFormatReport  *map[string]bool
	// AdreMaxConversationMessages: cap on conversation_history messages to Holmes (0 = default 40; nil = no change).
	AdreMaxConversationMessages *int
	// AdreQanInsightsPrompt: system prompt for QAN AI Insights. Max AdrePromptMaxBytes.
	AdreQanInsightsPrompt *string
	// AdreQanInsightsModel is default Holmes model alias for QAN AI Insights. Empty uses Holmes default.
	AdreQanInsightsModel *string
	// ServiceNow integration URL (non-secret). The API key and client token are secrets handled
	// separately (stored encrypted in adre_provisioning), not through settings.
	ServiceNowURL *string
	// PromptMaxBytes defines max prompt size for ADRE prompts.
	PromptMaxBytes *int
	// AdreChatRetentionDays: automatic purge of ADRE chats (days, 0 = never). Nil = no change.
	AdreChatRetentionDays *int
	// EnableSlackBot enables PMM-managed Slack (Socket Mode) integration.
	EnableSlackBot *bool
	// SlackAutoInvestigate enables event-driven auto-investigate (Grafana Alertmanager → investigation).
	SlackAutoInvestigate *bool
	// Slack human-chat allowlists and auto-investigate output/cost settings. Pointer-to-slice:
	// nil = no change; non-nil = replace the whole list.
	SlackAllowedChannels         *[]string
	SlackAllowedUsers            *[]string
	SlackAutoInvestigateChannels *[]string
	SlackAlertBotIDs             *[]string
	AutoInvestigateMinSeverity   *string
	AutoInvestigateLabelMatchers *[]string
	AutoInvestigateHourlyCap     *int

	// OTEL server collector and ClickHouse retention (nil sub-fields = no change).
	OtelCollectorEnabled     *bool
	OtelLogsRetentionDays    *int
	OtelTracesRetentionDays  *int
	OtelMetricsRetentionDays *int
}

// SetPMMServerID should be run on start up to generate unique PMM Server ID.
func SetPMMServerID(q reform.DBTX) error {
	settings, err := GetSettings(q)
	if err != nil {
		return err
	}
	if settings.PMMServerID != "" {
		return nil
	}
	settings.PMMServerID = uuid.NewString()
	return SaveSettings(q, settings)
}

// UpdateSettings updates only non-zero, non-empty values.
func UpdateSettings(q reform.DBTX, params *ChangeSettingsParams) (*Settings, error) { //nolint:gocognit,cyclop,unparam
	err := ValidateSettings(params)
	if err != nil {
		return nil, NewInvalidArgumentError("%s", err.Error())
	}

	if params.DefaultRoleID != nil {
		tx, ok := q.(*reform.TX)
		if !ok {
			return nil, fmt.Errorf("%w: changing Role ID requires a *reform.TX", ErrTxRequired)
		}

		var r Role
		err := findRole(tx, *params.DefaultRoleID, &r)
		if err != nil {
			return nil, err
		}
	}

	settings, err := GetSettings(q)
	if err != nil {
		return nil, err
	}

	if params.EnableUpdates != nil {
		settings.Updates.Enabled = params.EnableUpdates
	}

	if params.UpdateSnoozeDuration != 0 {
		settings.Updates.SnoozeDuration = params.UpdateSnoozeDuration
	}

	if params.EnableTelemetry != nil {
		settings.Telemetry.Enabled = params.EnableTelemetry

		if !*settings.Telemetry.Enabled {
			settings.Telemetry.UUID = ""
		}
	}
	if params.MetricsResolutions.LR != 0 {
		settings.MetricsResolutions.LR = params.MetricsResolutions.LR
	}
	if params.MetricsResolutions.MR != 0 {
		settings.MetricsResolutions.MR = params.MetricsResolutions.MR
	}
	if params.MetricsResolutions.HR != 0 {
		settings.MetricsResolutions.HR = params.MetricsResolutions.HR
	}
	if params.DataRetention != 0 {
		settings.DataRetention = params.DataRetention
	}

	if params.AWSPartitions != nil {
		settings.AWSPartitions = deduplicateStrings(params.AWSPartitions)
	}

	if params.SSHKey != nil {
		settings.SSHKey = pointer.GetString(params.SSHKey)
	}

	if params.EnableAdvisors != nil {
		settings.SaaS.Enabled = params.EnableAdvisors
	}

	if params.EnableNomad != nil {
		settings.Nomad.Enabled = params.EnableNomad
	}

	if params.AdvisorsRunInterval.RareInterval != 0 {
		settings.SaaS.AdvisorRunIntervals.RareInterval = params.AdvisorsRunInterval.RareInterval
	}
	if params.AdvisorsRunInterval.StandardInterval != 0 {
		settings.SaaS.AdvisorRunIntervals.StandardInterval = params.AdvisorsRunInterval.StandardInterval
	}
	if params.AdvisorsRunInterval.FrequentInterval != 0 {
		settings.SaaS.AdvisorRunIntervals.FrequentInterval = params.AdvisorsRunInterval.FrequentInterval
	}

	if len(params.DisableAdvisorChecks) != 0 {
		settings.SaaS.DisabledAdvisors = deduplicateStrings(append(settings.SaaS.DisabledAdvisors, params.DisableAdvisorChecks...))
	}

	if len(params.EnableAdvisorChecks) != 0 {
		m := make(map[string]struct{}, len(params.EnableAdvisorChecks))
		for _, p := range params.EnableAdvisorChecks {
			m[p] = struct{}{}
		}

		var res []string
		for _, c := range settings.SaaS.DisabledAdvisors {
			if _, ok := m[c]; !ok {
				res = append(res, c)
			}
		}
		settings.SaaS.DisabledAdvisors = res
	}

	if params.EnableVMCache != nil {
		settings.VictoriaMetrics.CacheEnabled = params.EnableVMCache
	}

	if params.PMMPublicAddress != nil {
		settings.PMMPublicAddress = pointer.GetString(params.PMMPublicAddress)
	}

	if params.EnableAzurediscover != nil {
		settings.Azurediscover.Enabled = params.EnableAzurediscover
	}

	if params.EnableAlerting != nil {
		settings.Alerting.Enabled = params.EnableAlerting
	}

	if params.EnableNativeQan != nil {
		settings.NativeQan.Enabled = params.EnableNativeQan
	}

	if params.EnableAccessControl != nil {
		settings.AccessControl.Enabled = params.EnableAccessControl
	}

	if params.EnableBackupManagement != nil {
		settings.BackupManagement.Enabled = params.EnableBackupManagement
	}

	if params.DefaultRoleID != nil {
		settings.DefaultRoleID = *params.DefaultRoleID
	}

	if params.EncryptedItems != nil {
		settings.EncryptedItems = params.EncryptedItems
	}

	if params.EnableAdre != nil {
		settings.Adre.Enabled = params.EnableAdre
	}
	if params.AdreURL != nil {
		settings.Adre.URL = pointer.GetString(params.AdreURL)
	}
	if params.AdreTLSSkipVerify != nil {
		settings.Adre.TLSSkipVerify = *params.AdreTLSSkipVerify
	}
	if params.AdreChatPrompt != nil {
		settings.Adre.ChatPrompt = pointer.GetString(params.AdreChatPrompt)
	}
	if params.AdreInvestigationPrompt != nil {
		settings.Adre.InvestigationPrompt = pointer.GetString(params.AdreInvestigationPrompt)
	}
	if params.AdreChatModel != nil {
		settings.Adre.ChatModel = strings.TrimSpace(pointer.GetString(params.AdreChatModel))
	}
	if params.AdreInvestigationModel != nil {
		settings.Adre.InvestigationModel = strings.TrimSpace(pointer.GetString(params.AdreInvestigationModel))
	}
	if params.AdreDefaultChatMode != nil {
		mode := strings.TrimSpace(pointer.GetString(params.AdreDefaultChatMode))
		if mode == "chat" {
			mode = "fast"
		}
		settings.Adre.DefaultChatMode = mode
	}
	if params.AdreBehaviorControlsFast != nil {
		settings.Adre.BehaviorControlsFast = maps.Clone(*params.AdreBehaviorControlsFast)
	}
	if params.AdreBehaviorControlsInvestigation != nil {
		settings.Adre.BehaviorControlsInvestigation = maps.Clone(*params.AdreBehaviorControlsInvestigation)
	}
	if params.AdreBehaviorControlsFormatReport != nil {
		settings.Adre.BehaviorControlsFormatReport = maps.Clone(*params.AdreBehaviorControlsFormatReport)
	}
	if params.AdreMaxConversationMessages != nil {
		settings.Adre.AdreMaxConversationMessages = *params.AdreMaxConversationMessages
	}
	if params.AdreQanInsightsPrompt != nil {
		settings.Adre.QanInsightsPrompt = pointer.GetString(params.AdreQanInsightsPrompt)
	}
	if params.AdreQanInsightsModel != nil {
		settings.Adre.QanInsightsModel = strings.TrimSpace(pointer.GetString(params.AdreQanInsightsModel))
	}
	if params.ServiceNowURL != nil {
		settings.Adre.ServiceNowURL = pointer.GetString(params.ServiceNowURL)
	}
	if params.PromptMaxBytes != nil {
		settings.Adre.PromptMaxBytes = *params.PromptMaxBytes
	}
	if params.AdreChatRetentionDays != nil {
		settings.Adre.AdreChatRetentionDays = new(*params.AdreChatRetentionDays)
	}

	if params.EnableSlackBot != nil {
		settings.Adre.SlackEnabled = *params.EnableSlackBot
		if !settings.Adre.SlackEnabled {
			settings.Adre.SlackAutoInvestigate = false
		}
	}
	if params.SlackAutoInvestigate != nil {
		settings.Adre.SlackAutoInvestigate = *params.SlackAutoInvestigate && settings.Adre.SlackEnabled
	}
	if params.SlackAllowedChannels != nil {
		settings.Adre.SlackAllowedChannels = normalizeSlackIDs(*params.SlackAllowedChannels)
	}
	if params.SlackAllowedUsers != nil {
		settings.Adre.SlackAllowedUsers = normalizeSlackIDs(*params.SlackAllowedUsers)
	}
	if params.SlackAutoInvestigateChannels != nil {
		settings.Adre.SlackAutoInvestigateChannels = normalizeSlackIDs(*params.SlackAutoInvestigateChannels)
	}
	if params.SlackAlertBotIDs != nil {
		settings.Adre.SlackAlertBotIDs = normalizeSlackIDs(*params.SlackAlertBotIDs)
	}
	if params.AutoInvestigateMinSeverity != nil {
		settings.Adre.AutoInvestigateMinSeverity = strings.TrimSpace(*params.AutoInvestigateMinSeverity)
	}
	if params.AutoInvestigateLabelMatchers != nil {
		settings.Adre.AutoInvestigateLabelMatchers = normalizeTrimmedList(*params.AutoInvestigateLabelMatchers)
	}
	if params.AutoInvestigateHourlyCap != nil {
		settings.Adre.AutoInvestigateHourlyCap = *params.AutoInvestigateHourlyCap
	}

	if params.OtelCollectorEnabled != nil {
		settings.Otel.CollectorEnabled = params.OtelCollectorEnabled
	}
	if params.OtelLogsRetentionDays != nil {
		settings.Otel.LogsRetentionDays = params.OtelLogsRetentionDays
	}
	if params.OtelTracesRetentionDays != nil {
		settings.Otel.TracesRetentionDays = params.OtelTracesRetentionDays
	}
	if params.OtelMetricsRetentionDays != nil {
		settings.Otel.MetricsRetentionDays = params.OtelMetricsRetentionDays
	}

	if params.EnableAdre != nil && !*params.EnableAdre {
		settings.Adre.SlackEnabled = false
		settings.Adre.SlackAutoInvestigate = false
	}

	if err := validateAdreSlackSettings(settings); err != nil { //nolint:noinlineerr
		return nil, NewInvalidArgumentError("%s", err.Error())
	}

	err = SaveSettings(q, settings)
	if err != nil {
		return nil, err
	}
	return settings, nil
}

// ValidateSettings validates settings changes.
func ValidateSettings(params *ChangeSettingsParams) error {
	// TODO: consider refactoring this and the validation for Advisors run intervals
	checkCases := []struct {
		dur       time.Duration
		fieldName string
	}{
		{params.MetricsResolutions.HR, "hr"},
		{params.MetricsResolutions.MR, "mr"},
		{params.MetricsResolutions.LR, "lr"},
	}
	for _, v := range checkCases {
		if v.dur == 0 {
			continue
		}

		_, err := validators.ValidateMetricResolution(v.dur)
		if err != nil {
			switch err.(type) { //nolint:errorlint
			case validators.DurationNotAllowedError:
				return fmt.Errorf("%s: should be a natural number of seconds", v.fieldName)
			case validators.MinDurationError:
				return fmt.Errorf("%s: minimal resolution is 1s", v.fieldName)
			default:
				return fmt.Errorf("%s: unknown error: %w", v.fieldName, err)
			}
		}
	}

	checkCases = []struct {
		dur       time.Duration
		fieldName string
	}{
		{params.AdvisorsRunInterval.RareInterval, "rare_interval"},
		{params.AdvisorsRunInterval.StandardInterval, "standard_interval"},
		{params.AdvisorsRunInterval.FrequentInterval, "frequent_interval"},
	}
	for _, v := range checkCases {
		if v.dur == 0 {
			continue
		}

		_, err := validators.ValidateAdvisorRunInterval(v.dur)
		if err != nil {
			switch err.(type) { //nolint:errorlint
			case validators.DurationNotAllowedError:
				return fmt.Errorf("%s: should be a natural number of seconds", v.fieldName)
			case validators.MinDurationError:
				return fmt.Errorf("%s: minimal resolution is 1s", v.fieldName)
			default:
				return fmt.Errorf("%s: unknown error: %w", v.fieldName, err)
			}
		}
	}

	if params.DataRetention != 0 {
		_, err := validators.ValidateDataRetention(params.DataRetention)
		if err != nil {
			switch err.(type) { //nolint:errorlint
			case validators.DurationNotAllowedError:
				return errors.New("data_retention: should be a natural number of days")
			case validators.MinDurationError:
				return errors.New("data_retention: minimal resolution is 24h")
			default:
				return fmt.Errorf("data_retention: unknown error: %w", err)
			}
		}
	}

	err := validators.ValidateAWSPartitions(params.AWSPartitions)
	if err != nil {
		return err
	}

	if params.AdreChatPrompt != nil && len(*params.AdreChatPrompt) > AdrePromptMaxBytes {
		return fmt.Errorf("chat_prompt: max %d bytes", AdrePromptMaxBytes)
	}
	if params.AdreInvestigationPrompt != nil && len(*params.AdreInvestigationPrompt) > AdrePromptMaxBytes {
		return fmt.Errorf("investigation_prompt: max %d bytes", AdrePromptMaxBytes)
	}
	if params.AdreDefaultChatMode != nil {
		mode := strings.TrimSpace(*params.AdreDefaultChatMode)
		if mode != "chat" && mode != "fast" && mode != "investigation" {
			return errors.New(`default_chat_mode: must be "fast" or "investigation"`)
		}
	}
	if params.AdreChatModel != nil {
		err := validateAdreModelAlias("chat_model", *params.AdreChatModel)
		if err != nil {
			return err
		}
	}
	if params.AdreInvestigationModel != nil {
		err := validateAdreModelAlias("investigation_model", *params.AdreInvestigationModel)
		if err != nil {
			return err
		}
	}
	if params.AdreQanInsightsModel != nil {
		err := validateAdreModelAlias("qan_insights_model", *params.AdreQanInsightsModel)
		if err != nil {
			return err
		}
	}
	if params.AdreMaxConversationMessages != nil {
		n := *params.AdreMaxConversationMessages
		if n != 0 && (n < 4 || n > 200) {
			return errors.New("adre_max_conversation_messages: must be between 4 and 200, or 0 for default")
		}
	}
	if params.AdreBehaviorControlsFast != nil {
		err := validateAdreBehaviorControlsMap("behavior_controls_fast", *params.AdreBehaviorControlsFast)
		if err != nil {
			return err
		}
	}
	if params.AdreBehaviorControlsInvestigation != nil {
		err := validateAdreBehaviorControlsMap("behavior_controls_investigation", *params.AdreBehaviorControlsInvestigation)
		if err != nil {
			return err
		}
	}
	if params.AdreBehaviorControlsFormatReport != nil {
		err := validateAdreBehaviorControlsMap("behavior_controls_format_report", *params.AdreBehaviorControlsFormatReport)
		if err != nil {
			return err
		}
	}
	if params.AdreQanInsightsPrompt != nil && len(*params.AdreQanInsightsPrompt) > AdrePromptMaxBytes {
		return fmt.Errorf("qan_insights_prompt: max %d bytes", AdrePromptMaxBytes)
	}
	if params.AdreChatRetentionDays != nil {
		n := *params.AdreChatRetentionDays
		if n < 0 || n > 36500 {
			return errors.New("adre_chat_retention_days: must be between 0 and 36500")
		}
	}

	if err := validateOtelSettingsParams(params); err != nil { //nolint:noinlineerr
		return err
	}
	if err := validateAdreSlackListParams(params); err != nil { //nolint:noinlineerr
		return err
	}

	return nil
}

// normalizeSlackIDs trims, drops empties, and de-duplicates a Slack ID list while preserving the
// admin's ordering (write-time normalization).
func normalizeSlackIDs(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		t := strings.TrimSpace(v)
		if t == "" {
			continue
		}
		if _, dup := seen[t]; dup {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}

// normalizeTrimmedList trims and drops empty entries, preserving order.
func normalizeTrimmedList(in []string) []string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		if t := strings.TrimSpace(v); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// isASCIIAlnum reports whether s is non-empty and entirely ASCII letters/digits.
func isASCIIAlnum(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if (r < '0' || r > '9') && (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') {
			return false
		}
	}
	return true
}

// validateSlackIDList caps list size and checks each entry is a plausible Slack object ID (ASCII
// alphanumeric, length 6–24). Lenient on prefix so enterprise-grid user IDs (W…) are accepted.
func validateSlackIDList(field string, ids []string) error {
	if len(ids) > 200 {
		return fmt.Errorf("%s: max 200 entries", field)
	}
	for _, id := range ids {
		t := strings.TrimSpace(id)
		if t == "" {
			continue
		}
		if len(t) < 6 || len(t) > 24 || !isASCIIAlnum(t) {
			return fmt.Errorf("%s: %q is not a valid Slack ID", field, t)
		}
	}
	return nil
}

// validateAdreSlackListParams validates the Slack allowlists / output channels and the
// auto-investigate selection + cost-guard parameters from a settings change.
func validateAdreSlackListParams(params *ChangeSettingsParams) error {
	lists := []struct {
		field string
		ptr   *[]string
	}{
		{"slack_allowed_channels", params.SlackAllowedChannels},
		{"slack_allowed_users", params.SlackAllowedUsers},
		{"slack_auto_investigate_channels", params.SlackAutoInvestigateChannels},
		{"slack_alert_bot_ids", params.SlackAlertBotIDs},
	}
	for _, l := range lists {
		if l.ptr == nil {
			continue
		}
		if err := validateSlackIDList(l.field, *l.ptr); err != nil {
			return err
		}
	}
	if params.AutoInvestigateMinSeverity != nil {
		switch strings.ToLower(strings.TrimSpace(*params.AutoInvestigateMinSeverity)) {
		case "", "info", "warning", "critical":
		default:
			return errors.New("auto_investigate_min_severity: must be one of info, warning, critical (or empty)")
		}
	}
	if params.AutoInvestigateLabelMatchers != nil {
		if len(*params.AutoInvestigateLabelMatchers) > 50 {
			return errors.New("auto_investigate_label_matchers: max 50 entries")
		}
		for _, m := range *params.AutoInvestigateLabelMatchers {
			t := strings.TrimSpace(m)
			if t != "" && !strings.Contains(t, "=") {
				return fmt.Errorf("auto_investigate_label_matchers: %q must be key=value", t)
			}
		}
	}
	if params.AutoInvestigateHourlyCap != nil && (*params.AutoInvestigateHourlyCap < 0 || *params.AutoInvestigateHourlyCap > 10000) {
		return errors.New("auto_investigate_hourly_cap: must be between 0 and 10000")
	}
	return nil
}

func validateOtelSettingsParams(params *ChangeSettingsParams) error {
	checkRetention := func(name string, days *int) error {
		if days == nil {
			return nil
		}
		if *days <= 0 || *days > 365 {
			return fmt.Errorf("%s: must be between 1 and 365 days", name)
		}
		return nil
	}
	err := checkRetention("otel_logs_retention_days", params.OtelLogsRetentionDays)
	if err != nil {
		return err
	}
	err = checkRetention("otel_traces_retention_days", params.OtelTracesRetentionDays)
	if err != nil {
		return err
	}
	err = checkRetention("otel_metrics_retention_days", params.OtelMetricsRetentionDays)
	if err != nil {
		return err
	}
	return nil
}

// validateAdreSlackSettings checks Slack integration: when enabled it requires ADRE with a Holmes URL.
func validateAdreSlackSettings(settings *Settings) error {
	if settings.Adre.SlackAutoInvestigate && !settings.Adre.SlackEnabled {
		return errors.New("slack_auto_investigate requires slack_enabled")
	}
	if !settings.Adre.SlackEnabled {
		return nil
	}
	if settings.GetAdreURL() == "" {
		return errors.New("slack_enabled requires AI Assistant enabled with a Holmes URL")
	}
	return nil
}

// validateAdreModelAlias caps Holmes model id length and rejects Unicode control characters in ADRE model settings fields.
func validateAdreModelAlias(field, value string) error {
	v := strings.TrimSpace(value)
	if len(v) > 256 { //nolint:mnd
		return fmt.Errorf("%s: max 256 bytes", field)
	}
	for _, r := range v {
		if unicode.IsControl(r) {
			return fmt.Errorf("%s: contains invalid control characters", field)
		}
	}
	return nil
}

// SaveSettings saves PMM Server settings.
// It may modify passed settings to fill defaults.
func SaveSettings(q reform.DBTX, s *Settings) error {
	s.fillDefaults()

	b, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	_, err = q.Exec("UPDATE settings SET settings = $1", b)
	if err != nil {
		return fmt.Errorf("failed to update settings: %w", err)
	}

	return nil
}
