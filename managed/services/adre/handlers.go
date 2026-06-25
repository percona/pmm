// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package adre

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
	pkgenv "github.com/percona/pmm/managed/utils/env"
	"github.com/percona/pmm/managed/utils/validators"
)

// allowInsecureADREURLs reports whether the dev-only escape hatch permitting plaintext http to public
// ADRE / ServiceNow / callback URLs is enabled (PMM_DEV_ADRE_ALLOW_INSECURE_URL). Default off.
func allowInsecureADREURLs() bool {
	b, _ := strconv.ParseBool(os.Getenv(pkgenv.AdreAllowInsecureURL))
	return b
}

func sanitizeQanInsightsAnalysis(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return raw
	}

	summaryIdx := strings.Index(strings.ToLower(trimmed), "## summary")
	if summaryIdx <= 0 {
		return trimmed
	}

	prefix := strings.ToLower(trimmed[:summaryIdx])
	if strings.Contains(prefix, "runbook") ||
		strings.Contains(prefix, "fetch_runbook") ||
		strings.Contains(prefix, "fetch_skill") ||
		strings.Contains(prefix, "i found a runbook") ||
		strings.Contains(prefix, "i found a skill") ||
		strings.Contains(prefix, "used it to troubleshoot") {
		return strings.TrimSpace(trimmed[summaryIdx:])
	}

	return trimmed
}

const (
	adreDisabledMsg  = "ADRE is disabled. Enable it in Settings."
	adreURLNotSetMsg = "HolmesGPT URL is not configured. Set it in Settings."
)

// AlertWebhookSink processes a Grafana alert-webhook payload (raw JSON body). Implemented by
// autoinvestigate.Service; kept as a raw-bytes interface so adre needn't import that package.
type AlertWebhookSink interface {
	ProcessWebhook(ctx context.Context, raw []byte)
}

// Handlers provides HTTP handlers for the ADRE proxy API.
type Handlers struct {
	db            *reform.DB
	grafana       GrafanaAuth
	vm            v1.API
	clickhouse    ClickHousePools
	streams       *ActiveChatStreams
	searchLimiter *SearchRateLimiter
	reqTimeout    time.Duration
	streamTimeout time.Duration
	l             *logrus.Entry
	alertSink     AlertWebhookSink
}

// SetAlertSink injects the auto-investigate sink used by the alert webhook. Safe to leave unset (the
// webhook then returns 503).
func (h *Handlers) SetAlertSink(sink AlertWebhookSink) {
	h.alertSink = sink
}

// NewHandlers creates new ADRE HTTP handlers.
func NewHandlers(db *reform.DB, grafana GrafanaAuth, vm v1.API, clickhouse ClickHousePools) *Handlers {
	return &Handlers{
		db:            db,
		grafana:       grafana,
		vm:            vm,
		clickhouse:    clickhouse,
		streams:       NewActiveChatStreams(),
		searchLimiter: NewSearchRateLimiter(),
		reqTimeout:    5 * time.Minute, //nolint:mnd
		streamTimeout: 5 * time.Minute, //nolint:mnd
		l:             logrus.WithField("component", "adre-handlers"),
	}
}

// checkAdreEnabled returns (settings, true) if ADRE is enabled and URL is set; otherwise writes an error and returns (nil, false).
//
//nolint:funcorder // helper grouped near the constructor; reads better than visibility ordering
func (h *Handlers) checkAdreEnabled(w http.ResponseWriter) (*models.Settings, bool) {
	settings, err := models.GetSettings(h.db)
	if err != nil {
		h.l.Errorf("GetSettings: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to get settings")
		return nil, false
	}
	if !settings.IsAdreEnabled() {
		writeJSONError(w, http.StatusBadRequest, adreDisabledMsg)
		return nil, false
	}
	url := settings.GetAdreURL()
	if url == "" {
		writeJSONError(w, http.StatusBadRequest, adreURLNotSetMsg)
		return nil, false
	}
	return settings, true
}

type adreSettingsResponse struct {
	Enabled                       bool            `json:"enabled"`
	URL                           string          `json:"url"`
	ChatPrompt                    string          `json:"chat_prompt"`
	InvestigationPrompt           string          `json:"investigation_prompt"`
	ChatModel                     string          `json:"chat_model"`
	InvestigationModel            string          `json:"investigation_model"`
	ChatPromptDisplay             string          `json:"chat_prompt_display"`
	InvestigationPromptDisplay    string          `json:"investigation_prompt_display"`
	DefaultChatMode               string          `json:"default_chat_mode"`
	BehaviorControlsFast          map[string]bool `json:"behavior_controls_fast"`
	BehaviorControlsInvestigation map[string]bool `json:"behavior_controls_investigation"`
	BehaviorControlsFormatReport  map[string]bool `json:"behavior_controls_format_report"`
	AdreMaxConversationMessages   int             `json:"adre_max_conversation_messages"`
	QanInsightsPrompt             string          `json:"qan_insights_prompt"`
	QanInsightsPromptDisplay      string          `json:"qan_insights_prompt_display"`
	QanInsightsModel              string          `json:"qan_insights_model"`
	ServiceNowURL                 string          `json:"servicenow_url"`
	ServiceNowConfigured          bool            `json:"servicenow_configured"`
	PromptMaxBytes                int             `json:"prompt_max_bytes"`
	AdreChatRetentionDays         int             `json:"adre_chat_retention_days"`
	SlackEnabled                  bool            `json:"slack_enabled"`
	SlackAutoInvestigate          bool            `json:"slack_auto_investigate"`
	SlackConfigured               bool            `json:"slack_configured"`
	TLSSkipVerify                 bool            `json:"tls_skip_verify"`
	SlackAllowedChannels          []string        `json:"slack_allowed_channels"`
	SlackAllowedUsers             []string        `json:"slack_allowed_users"`
	SlackAutoInvestigateChannels  []string        `json:"slack_auto_investigate_channels"`
	AutoInvestigateMinSeverity    string          `json:"auto_investigate_min_severity"`
	AutoInvestigateLabelMatchers  []string        `json:"auto_investigate_label_matchers"`
	AutoInvestigateHourlyCap      int             `json:"auto_investigate_hourly_cap"`
}

func applyAdreSettingsDefaults(r *adreSettingsResponse) {
	if r.DefaultChatMode == "" {
		r.DefaultChatMode = "investigation" //nolint:goconst
	}
	if r.PromptMaxBytes <= 0 {
		r.PromptMaxBytes = models.AdrePromptMaxBytes
	}
	if r.AdreMaxConversationMessages <= 0 {
		r.AdreMaxConversationMessages = AdreMaxConversationMessagesDefault
	}
	if r.AdreChatRetentionDays < 0 {
		r.AdreChatRetentionDays = models.AdreChatRetentionDaysDefault
	}
}

// GetSettings handles GET /v1/adre/settings.
func (h *Handlers) GetSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	settings, err := models.GetSettings(h.db)
	if err != nil {
		h.l.Errorf("GetSettings: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to get settings")
		return
	}
	chatPromptDisplay := settings.Adre.ChatPrompt
	if chatPromptDisplay == "" {
		chatPromptDisplay = DefaultChatPrompt
	}
	investigationPromptDisplay := settings.Adre.InvestigationPrompt
	if investigationPromptDisplay == "" {
		investigationPromptDisplay = DefaultInvestigationPrompt
	}
	qanInsightsPromptDisplay := settings.Adre.QanInsightsPrompt
	if qanInsightsPromptDisplay == "" {
		qanInsightsPromptDisplay = DefaultQanInsightsPrompt
	}
	serviceNowTokensSet, slackConfigured, err := models.AdreSecretsConfigured(h.db)
	if err != nil {
		h.l.Warnf("AdreSecretsConfigured: %v", err)
	}
	serviceNowConfigured := settings.Adre.ServiceNowURL != "" && serviceNowTokensSet
	resp := adreSettingsResponse{
		Enabled:                       settings.IsAdreEnabled(),
		URL:                           settings.GetAdreURL(),
		ChatPrompt:                    settings.Adre.ChatPrompt,
		InvestigationPrompt:           settings.Adre.InvestigationPrompt,
		ChatModel:                     settings.Adre.ChatModel,
		InvestigationModel:            settings.Adre.InvestigationModel,
		ChatPromptDisplay:             chatPromptDisplay,
		InvestigationPromptDisplay:    investigationPromptDisplay,
		DefaultChatMode:               settings.Adre.DefaultChatMode,
		BehaviorControlsFast:          settings.Adre.BehaviorControlsFast,
		BehaviorControlsInvestigation: settings.Adre.BehaviorControlsInvestigation,
		BehaviorControlsFormatReport:  settings.Adre.BehaviorControlsFormatReport,
		AdreMaxConversationMessages:   settings.Adre.AdreMaxConversationMessages,
		QanInsightsPrompt:             settings.Adre.QanInsightsPrompt,
		QanInsightsPromptDisplay:      qanInsightsPromptDisplay,
		QanInsightsModel:              settings.Adre.QanInsightsModel,
		ServiceNowURL:                 settings.Adre.ServiceNowURL,
		ServiceNowConfigured:          serviceNowConfigured,
		PromptMaxBytes:                settings.Adre.PromptMaxBytes,
		AdreChatRetentionDays:         settings.GetAdreChatRetentionDays(),
		SlackEnabled:                  settings.Adre.SlackEnabled,
		SlackAutoInvestigate:          settings.Adre.SlackAutoInvestigate,
		SlackConfigured:               slackConfigured,
		TLSSkipVerify:                 settings.Adre.TLSSkipVerify,
		SlackAllowedChannels:          settings.Adre.SlackAllowedChannels,
		SlackAllowedUsers:             settings.Adre.SlackAllowedUsers,
		SlackAutoInvestigateChannels:  settings.Adre.SlackAutoInvestigateChannels,
		AutoInvestigateMinSeverity:    settings.Adre.AutoInvestigateMinSeverity,
		AutoInvestigateLabelMatchers:  settings.Adre.AutoInvestigateLabelMatchers,
		AutoInvestigateHourlyCap:      settings.Adre.AutoInvestigateHourlyCap,
	}
	applyAdreSettingsDefaults(&resp)
	body, err := json.Marshal(resp)
	if err != nil {
		h.l.Errorf("Marshal settings: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to encode response")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(body); err != nil { //nolint:noinlineerr
		h.l.Warnf("Write settings response: %v", err)
	}
}

// PostSettings handles POST /v1/adre/settings.
func (h *Handlers) PostSettings(w http.ResponseWriter, r *http.Request) { //nolint:cyclop,gocognit,maintidx
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	// Writing ADRE settings (integration URLs/secrets, prompts, models) is admin-only.
	// GET stays viewer; the auth proxy also enforces this via methodRules, this is defence in depth.
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	var body struct {
		Enabled                       *bool            `json:"enabled"`
		URL                           *string          `json:"url"`
		ChatPrompt                    *string          `json:"chat_prompt"`
		InvestigationPrompt           *string          `json:"investigation_prompt"`
		ChatModel                     *string          `json:"chat_model"`
		InvestigationModel            *string          `json:"investigation_model"`
		DefaultChatMode               *string          `json:"default_chat_mode"`
		BehaviorControlsFast          *map[string]bool `json:"behavior_controls_fast"`
		BehaviorControlsInvestigation *map[string]bool `json:"behavior_controls_investigation"`
		BehaviorControlsFormatReport  *map[string]bool `json:"behavior_controls_format_report"`
		AdreMaxConversationMessages   *int             `json:"adre_max_conversation_messages"`
		QanInsightsPrompt             *string          `json:"qan_insights_prompt"`
		QanInsightsModel              *string          `json:"qan_insights_model"`
		ServiceNowURL                 *string          `json:"servicenow_url"`
		ServiceNowAPIKey              *string          `json:"servicenow_api_key"`
		ServiceNowClientToken         *string          `json:"servicenow_client_token"`
		PromptMaxBytes                *int             `json:"prompt_max_bytes"`
		AdreChatRetentionDays         *int             `json:"adre_chat_retention_days"`
		SlackEnabled                  *bool            `json:"slack_enabled"`
		SlackAutoInvestigate          *bool            `json:"slack_auto_investigate"`
		SlackBotToken                 *string          `json:"slack_bot_token"`
		SlackAppToken                 *string          `json:"slack_app_token"`
		TLSSkipVerify                 *bool            `json:"tls_skip_verify"`
		SlackAllowedChannels          *[]string        `json:"slack_allowed_channels"`
		SlackAllowedUsers             *[]string        `json:"slack_allowed_users"`
		SlackAutoInvestigateChannels  *[]string        `json:"slack_auto_investigate_channels"`
		AutoInvestigateMinSeverity    *string          `json:"auto_investigate_min_severity"`
		AutoInvestigateLabelMatchers  *[]string        `json:"auto_investigate_label_matchers"`
		AutoInvestigateHourlyCap      *int             `json:"auto_investigate_hourly_cap"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { //nolint:noinlineerr
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}
	hasChange := body.Enabled != nil || body.URL != nil || body.ChatPrompt != nil ||
		body.InvestigationPrompt != nil || body.ChatModel != nil || body.InvestigationModel != nil || body.DefaultChatMode != nil ||
		body.BehaviorControlsFast != nil || body.BehaviorControlsInvestigation != nil || body.BehaviorControlsFormatReport != nil ||
		body.AdreMaxConversationMessages != nil || body.QanInsightsPrompt != nil || body.QanInsightsModel != nil ||
		body.ServiceNowURL != nil || body.ServiceNowAPIKey != nil || body.ServiceNowClientToken != nil ||
		body.PromptMaxBytes != nil || body.AdreChatRetentionDays != nil ||
		body.SlackEnabled != nil || body.SlackAutoInvestigate != nil || body.SlackBotToken != nil || body.SlackAppToken != nil ||
		body.TLSSkipVerify != nil ||
		body.SlackAllowedChannels != nil || body.SlackAllowedUsers != nil || body.SlackAutoInvestigateChannels != nil ||
		body.AutoInvestigateMinSeverity != nil || body.AutoInvestigateLabelMatchers != nil || body.AutoInvestigateHourlyCap != nil
	if !hasChange {
		writeJSONError(w, http.StatusBadRequest, "No changes provided")
		return
	}
	if body.URL != nil {
		trimmed := strings.TrimSpace(*body.URL)
		body.URL = &trimmed
		if trimmed != "" {
			if _, err := validators.RequireSecureServiceURL(trimmed, allowInsecureADREURLs()); err != nil { //nolint:noinlineerr
				writeJSONError(w, http.StatusBadRequest, err.Error())
				return
			}
		}
	}
	if body.ServiceNowURL != nil {
		trimmed := strings.TrimSpace(*body.ServiceNowURL)
		body.ServiceNowURL = &trimmed
		if trimmed != "" {
			if _, err := validators.RequireSecureExternalURL(trimmed); err != nil { //nolint:noinlineerr
				writeJSONError(w, http.StatusBadRequest, "servicenow_url: "+err.Error())
				return
			}
		}
	}
	currentSettings, err := models.GetSettings(h.db)
	if err != nil {
		h.l.Errorf("GetSettings before validate: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to get settings")
		return
	}
	effectivePromptMaxBytes := currentSettings.Adre.PromptMaxBytes
	if effectivePromptMaxBytes <= 0 {
		effectivePromptMaxBytes = models.AdrePromptMaxBytes
	}
	if body.PromptMaxBytes != nil {
		n := *body.PromptMaxBytes
		if n < 1024 || n > models.AdrePromptMaxBytesHardMax {
			writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("prompt_max_bytes: must be between 1024 and %d", models.AdrePromptMaxBytesHardMax))
			return
		}
		effectivePromptMaxBytes = n
	}
	if body.ChatPrompt != nil && len(*body.ChatPrompt) > effectivePromptMaxBytes {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("chat_prompt: max %d bytes", effectivePromptMaxBytes))
		return
	}
	if body.InvestigationPrompt != nil && len(*body.InvestigationPrompt) > effectivePromptMaxBytes {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("investigation_prompt: max %d bytes", effectivePromptMaxBytes))
		return
	}
	if body.DefaultChatMode != nil {
		mode := strings.TrimSpace(*body.DefaultChatMode)
		if mode != "chat" && mode != "fast" && mode != "investigation" { //nolint:goconst
			writeJSONError(w, http.StatusBadRequest, `default_chat_mode: must be "fast" or "investigation"`)
			return
		}
		body.DefaultChatMode = &mode
	}
	if body.ChatModel != nil {
		trimmed := strings.TrimSpace(*body.ChatModel)
		body.ChatModel = &trimmed
	}
	if body.InvestigationModel != nil {
		trimmed := strings.TrimSpace(*body.InvestigationModel)
		body.InvestigationModel = &trimmed
	}
	if body.QanInsightsModel != nil {
		trimmed := strings.TrimSpace(*body.QanInsightsModel)
		body.QanInsightsModel = &trimmed
	}
	if body.AdreMaxConversationMessages != nil {
		n := *body.AdreMaxConversationMessages
		if n != 0 && (n < 4 || n > 200) {
			writeJSONError(w, http.StatusBadRequest, "adre_max_conversation_messages: must be between 4 and 200, or 0 for default")
			return
		}
	}
	if body.BehaviorControlsFast != nil {
		err := ValidateBehaviorControlsMap(*body.BehaviorControlsFast)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "behavior_controls_fast: "+err.Error())
			return
		}
	}
	if body.BehaviorControlsInvestigation != nil {
		err := ValidateBehaviorControlsMap(*body.BehaviorControlsInvestigation)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "behavior_controls_investigation: "+err.Error())
			return
		}
	}
	if body.BehaviorControlsFormatReport != nil {
		err := ValidateBehaviorControlsMap(*body.BehaviorControlsFormatReport)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "behavior_controls_format_report: "+err.Error())
			return
		}
	}
	if body.QanInsightsPrompt != nil && len(*body.QanInsightsPrompt) > effectivePromptMaxBytes {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("qan_insights_prompt: max %d bytes", effectivePromptMaxBytes))
		return
	}
	if body.AdreChatRetentionDays != nil {
		n := *body.AdreChatRetentionDays
		if n < 0 || n > 36500 {
			writeJSONError(w, http.StatusBadRequest, "adre_chat_retention_days: must be between 0 and 36500")
			return
		}
	}
	params := &models.ChangeSettingsParams{
		EnableAdre:                        body.Enabled,
		AdreURL:                           body.URL,
		AdreChatPrompt:                    body.ChatPrompt,
		AdreInvestigationPrompt:           body.InvestigationPrompt,
		AdreChatModel:                     body.ChatModel,
		AdreInvestigationModel:            body.InvestigationModel,
		AdreDefaultChatMode:               body.DefaultChatMode,
		AdreBehaviorControlsFast:          body.BehaviorControlsFast,
		AdreBehaviorControlsInvestigation: body.BehaviorControlsInvestigation,
		AdreBehaviorControlsFormatReport:  body.BehaviorControlsFormatReport,
		AdreMaxConversationMessages:       body.AdreMaxConversationMessages,
		AdreQanInsightsPrompt:             body.QanInsightsPrompt,
		AdreQanInsightsModel:              body.QanInsightsModel,
		ServiceNowURL:                     body.ServiceNowURL,
		PromptMaxBytes:                    body.PromptMaxBytes,
		AdreChatRetentionDays:             body.AdreChatRetentionDays,
		EnableSlackBot:                    body.SlackEnabled,
		SlackAutoInvestigate:              body.SlackAutoInvestigate,
		AdreTLSSkipVerify:                 body.TLSSkipVerify,
		SlackAllowedChannels:              body.SlackAllowedChannels,
		SlackAllowedUsers:                 body.SlackAllowedUsers,
		SlackAutoInvestigateChannels:      body.SlackAutoInvestigateChannels,
		AutoInvestigateMinSeverity:        body.AutoInvestigateMinSeverity,
		AutoInvestigateLabelMatchers:      body.AutoInvestigateLabelMatchers,
		AutoInvestigateHourlyCap:          body.AutoInvestigateHourlyCap,
	}
	if _, err := models.UpdateSettings(h.db, params); err != nil { //nolint:noinlineerr
		h.l.Errorf("UpdateSettings: %v", err)
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Integration secrets (ServiceNow API key/client token, Slack bot/app tokens) are stored encrypted in
	// adre_provisioning, not in the settings JSONB. Load once and reuse for the masked response below.
	prov, err := models.GetAdreProvisioning(h.db)
	if err != nil {
		h.l.Errorf("GetAdreProvisioning: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to load provisioning")
		return
	}
	// Apply only the provided (non-nil) tokens: an empty string clears, nil keeps the existing value.
	if body.ServiceNowAPIKey != nil || body.ServiceNowClientToken != nil ||
		body.SlackBotToken != nil || body.SlackAppToken != nil {
		if body.ServiceNowAPIKey != nil {
			prov.ServiceNowAPIKey = *body.ServiceNowAPIKey
		}
		if body.ServiceNowClientToken != nil {
			prov.ServiceNowClientToken = *body.ServiceNowClientToken
		}
		if body.SlackBotToken != nil {
			prov.SlackBotToken = *body.SlackBotToken
		}
		if body.SlackAppToken != nil {
			prov.SlackAppToken = *body.SlackAppToken
		}
		if err := models.SaveAdreProvisioning(h.db, prov); err != nil { //nolint:noinlineerr
			h.l.Errorf("SaveAdreProvisioning: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "Failed to save integration secrets")
			return
		}
	}
	settings, _ := models.GetSettings(h.db)
	// When auto-investigate is enabled, self-heal the Grafana alert-webhook contact point here so admins
	// don't have to separately re-run deployment provisioning after toggling it on. Idempotent and
	// best-effort: the auto-investigate reconciliation poll works even if this fails.
	if settings != nil && settings.Adre.SlackAutoInvestigate {
		if err := h.provisioner().EnsureAlertWebhook(incomingAuthContext(r)); err != nil {
			h.l.Warnf("ensure auto-investigate webhook contact point after settings save: %v", err)
		}
	}
	chatPromptDisplay := settings.Adre.ChatPrompt
	if chatPromptDisplay == "" {
		chatPromptDisplay = DefaultChatPrompt
	}
	investigationPromptDisplay := settings.Adre.InvestigationPrompt
	if investigationPromptDisplay == "" {
		investigationPromptDisplay = DefaultInvestigationPrompt
	}
	qanInsightsPromptDisplayPost := settings.Adre.QanInsightsPrompt
	if qanInsightsPromptDisplayPost == "" {
		qanInsightsPromptDisplayPost = DefaultQanInsightsPrompt
	}
	serviceNowConfigured := settings.Adre.ServiceNowURL != "" && prov.ServiceNowAPIKey != "" && prov.ServiceNowClientToken != ""
	slackConfigured := prov.SlackBotToken != "" && prov.SlackAppToken != ""
	resp := adreSettingsResponse{
		Enabled:                       settings.IsAdreEnabled(),
		URL:                           settings.GetAdreURL(),
		ChatPrompt:                    settings.Adre.ChatPrompt,
		InvestigationPrompt:           settings.Adre.InvestigationPrompt,
		ChatModel:                     settings.Adre.ChatModel,
		InvestigationModel:            settings.Adre.InvestigationModel,
		ChatPromptDisplay:             chatPromptDisplay,
		InvestigationPromptDisplay:    investigationPromptDisplay,
		DefaultChatMode:               settings.Adre.DefaultChatMode,
		BehaviorControlsFast:          settings.Adre.BehaviorControlsFast,
		BehaviorControlsInvestigation: settings.Adre.BehaviorControlsInvestigation,
		BehaviorControlsFormatReport:  settings.Adre.BehaviorControlsFormatReport,
		AdreMaxConversationMessages:   settings.Adre.AdreMaxConversationMessages,
		QanInsightsPrompt:             settings.Adre.QanInsightsPrompt,
		QanInsightsPromptDisplay:      qanInsightsPromptDisplayPost,
		QanInsightsModel:              settings.Adre.QanInsightsModel,
		ServiceNowURL:                 settings.Adre.ServiceNowURL,
		ServiceNowConfigured:          serviceNowConfigured,
		PromptMaxBytes:                settings.Adre.PromptMaxBytes,
		AdreChatRetentionDays:         settings.GetAdreChatRetentionDays(),
		SlackEnabled:                  settings.Adre.SlackEnabled,
		SlackAutoInvestigate:          settings.Adre.SlackAutoInvestigate,
		SlackConfigured:               slackConfigured,
		TLSSkipVerify:                 settings.Adre.TLSSkipVerify,
		SlackAllowedChannels:          settings.Adre.SlackAllowedChannels,
		SlackAllowedUsers:             settings.Adre.SlackAllowedUsers,
		SlackAutoInvestigateChannels:  settings.Adre.SlackAutoInvestigateChannels,
		AutoInvestigateMinSeverity:    settings.Adre.AutoInvestigateMinSeverity,
		AutoInvestigateLabelMatchers:  settings.Adre.AutoInvestigateLabelMatchers,
		AutoInvestigateHourlyCap:      settings.Adre.AutoInvestigateHourlyCap,
	}
	applyAdreSettingsDefaults(&resp)
	respBody, err := json.Marshal(resp)
	if err != nil {
		h.l.Errorf("Marshal settings: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to encode response")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(respBody); err != nil { //nolint:noinlineerr
		h.l.Warnf("Write settings response: %v", err)
	}
}

// GetModels handles GET /v1/adre/models.
func (h *Handlers) GetModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	settings, ok := h.checkAdreEnabled(w)
	if !ok {
		return
	}
	client := NewClientFromSettings(settings)
	ctx, cancel := context.WithTimeout(r.Context(), h.reqTimeout)
	defer cancel()
	modelsList, err := client.Models(ctx)
	if err != nil {
		h.l.Warnf("HolmesGPT Models: %v", err)
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	resp := struct {
		ModelName []string `json:"model_name"`
	}{ModelName: modelsList}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil { //nolint:noinlineerr
		h.l.Errorf("Encode models: %v", err)
	}
}

// maxDashboardContextBytes caps PMM UI Grafana URL context appended to additional_system_prompt.
const maxDashboardContextBytes = 32 * 1024

// ResolveChatSystemPrompt returns the additional_system_prompt for chat (fast or investigation) mode
// with the always-on ScopeGuardrail appended. Empty settings value uses the built-in default.
// Idempotent: if the resolved prompt already contains scopeGuardrailMarker the guardrail is not appended again.
func ResolveChatSystemPrompt(settings *models.Settings, mode string) string {
	base := DefaultChatPrompt
	if mode == "investigation" {
		if settings.Adre.InvestigationPrompt != "" {
			base = settings.Adre.InvestigationPrompt
		} else {
			base = DefaultInvestigationPrompt
		}
	} else if settings.Adre.ChatPrompt != "" {
		base = settings.Adre.ChatPrompt
	}
	return appendScopeGuardrail(base)
}

// ResolveQanInsightsSystemPrompt returns the system prompt for QAN AI Insights with ScopeGuardrail appended.
// Empty settings value uses the built-in default. Idempotent (see ResolveChatSystemPrompt).
func ResolveQanInsightsSystemPrompt(settings *models.Settings) string {
	base := DefaultQanInsightsPrompt
	if settings.Adre.QanInsightsPrompt != "" {
		base = settings.Adre.QanInsightsPrompt
	}
	return appendScopeGuardrail(base)
}

// appendScopeGuardrail appends ScopeGuardrail to p (separated by a blank line). If p already contains
// scopeGuardrailMarker, p is returned unchanged so customers who manually pasted the guardrail into
// their custom prompt do not get a duplicated tail.
func appendScopeGuardrail(p string) string {
	if strings.Contains(p, scopeGuardrailMarker) {
		return p
	}
	return strings.TrimRight(p, "\n") + "\n\n" + ScopeGuardrail
}

func resolveChatModel(settings *models.Settings, mode string, reqModel string) string {
	if model := strings.TrimSpace(reqModel); model != "" {
		return model
	}
	if mode == "investigation" {
		return strings.TrimSpace(settings.Adre.InvestigationModel)
	}
	return strings.TrimSpace(settings.Adre.ChatModel)
}

// PostChat handles POST /v1/adre/chat. If body has "stream": true, streams the response.
func (h *Handlers) PostChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	settings, err := models.GetSettings(h.db)
	if err != nil {
		h.l.Errorf("GetSettings: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to get settings")
		return
	}
	if !settings.IsAdreEnabled() {
		writeJSONError(w, http.StatusBadRequest, adreDisabledMsg)
		return
	}
	if settings.GetAdreURL() == "" {
		writeJSONError(w, http.StatusBadRequest, adreURLNotSetMsg)
		return
	}
	dec := json.NewDecoder(r.Body)
	dec.UseNumber()
	var body chatRequestBody
	if err := dec.Decode(&body); err != nil { //nolint:noinlineerr
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}
	if strings.TrimSpace(body.Ask) == "" {
		writeJSONError(w, http.StatusBadRequest, "ask is required")
		return
	}
	mode := "fast"
	if body.Mode != nil {
		m := strings.TrimSpace(*body.Mode)
		switch m {
		case "investigation":
			mode = "investigation"
		case "fast", "chat":
			mode = "fast"
		}
	} else if settings.Adre.DefaultChatMode == "investigation" {
		mode = "investigation"
	}
	req := &body.ChatRequest
	req.BehaviorControls = ResolveBehaviorControlsForPostChat(settings, mode)
	h.l.WithFields(logrus.Fields{
		"mode":              mode,
		"behavior_controls": req.BehaviorControls,
	}).Debug("PostChat behavior controls resolved")
	h.postChatWithPersistence(w, r, settings, &body)
}

// PostQanInsights handles POST /v1/adre/qan-insights. Runs query analytics and optimization via Holmes (non-streaming).
func (h *Handlers) PostQanInsights(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	settings, ok := h.checkAdreEnabled(w)
	if !ok {
		return
	}
	if settings.GetAdreURL() == "" {
		writeJSONError(w, http.StatusBadRequest, adreURLNotSetMsg)
		return
	}
	var body struct {
		ServiceID   string `json:"service_id"`
		QueryText   string `json:"query_text"`
		QueryID     string `json:"query_id"`
		Fingerprint string `json:"fingerprint"`
		TimeFrom    string `json:"time_from"`
		TimeTo      string `json:"time_to"`
		Force       bool   `json:"force"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { //nolint:noinlineerr
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}
	body.ServiceID = strings.TrimSpace(body.ServiceID)
	body.QueryText = strings.TrimSpace(body.QueryText)
	if body.ServiceID == "" || body.QueryText == "" {
		writeJSONError(w, http.StatusBadRequest, "service_id and query_text are required")
		return
	}
	if !body.Force && body.QueryID != "" {
		rows, err := h.db.Query(
			"SELECT analysis, created_at FROM qan_insights_cache WHERE query_id = $1 AND service_id = $2 ORDER BY created_at DESC LIMIT 1",
			body.QueryID, body.ServiceID,
		)
		if err == nil {
			var cachedAnalysis string
			var cachedAt time.Time
			found := rows.Next() && rows.Scan(&cachedAnalysis, &cachedAt) == nil
			_ = rows.Close() //nolint:sqlclosecheck
			if found {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{ //nolint:errchkjson // response already committed
					"analysis":   cachedAnalysis,
					"created_at": cachedAt.Format(time.RFC3339), //nolint:goconst
					"cached":     true,
				})
				return
			}
		}
	}
	userMessage := fmt.Sprintf(
		"Analyze this query and provide optimization suggestions based on QAN metrics and schema.\n"+
			"service_id: %s\nquery_id: %s\nfingerprint: %s\nquery_text: %s\ntime_from: %s\ntime_to: %s",
		body.ServiceID, body.QueryID, body.Fingerprint, body.QueryText, body.TimeFrom, body.TimeTo,
	)
	pageContext := map[string]string{
		"service_id":  body.ServiceID,
		"query_text":  body.QueryText,
		"query_id":    body.QueryID,
		"fingerprint": body.Fingerprint,
		"time_from":   body.TimeFrom,
		"time_to":     body.TimeTo,
	}
	client := NewClientFromSettings(settings)
	ctx, cancel := context.WithTimeout(r.Context(), h.reqTimeout)
	defer cancel()
	qanModel := strings.TrimSpace(settings.Adre.QanInsightsModel)
	qanStart := time.Now()
	chatResp, err := client.Chat(ctx, &ChatRequest{
		Ask:                    userMessage,
		Model:                  qanModel,
		AdditionalSystemPrompt: ResolveQanInsightsSystemPrompt(settings),
		PageContext:            pageContext,
		Stream:                 false,
	})
	if err != nil {
		h.l.Warnf("QanInsights Chat: %v", err)
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	analysis := sanitizeQanInsightsAnalysis(chatResp.Analysis)
	triggeredBy := h.userLoginFromRequest(r)
	featureRef := body.QueryID + "|" + body.ServiceID
	if body.QueryID != "" {
		_, err := h.db.Exec(
			`INSERT INTO qan_insights_cache (id, query_id, service_id, fingerprint, time_from, time_to, analysis, created_at)
			 VALUES (gen_random_uuid()::text, $1, $2, $3, $4, $5, $6, NOW())
			 ON CONFLICT (query_id, service_id) DO UPDATE SET analysis = $6, fingerprint = $3, time_from = $4, time_to = $5, created_at = NOW()`,
			body.QueryID, body.ServiceID, body.Fingerprint, body.TimeFrom, body.TimeTo, analysis,
		)
		if err != nil {
			h.l.Warnf("QanInsights cache upsert: %v", err)
		}
	}
	_, _ = RecordHolmesUsage(ctx, UsageRecordInput{
		DB:           h.db,
		Feature:      HolmesFeatureQanInsights,
		FeatureRef:   featureRef,
		Model:        qanModel,
		Metadata:     chatResp.Metadata,
		TriggeredBy:  triggeredBy,
		LatencyMs:    int(time.Since(qanStart).Milliseconds()),
		QanQueryID:   body.QueryID,
		QanServiceID: body.ServiceID,
	})
	usage := ParseHolmesMetadata(chatResp.Metadata)
	resp := map[string]any{
		"analysis":   analysis,
		"created_at": time.Now().Format(time.RFC3339),
		"cached":     false,
	}
	if usage != nil {
		resp["usage"] = HolmesUsageMap(usage, qanModel)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp) //nolint:errchkjson // response already committed
}

// GetQanInsights handles GET /v1/adre/qan-insights. Returns cached analysis for a query+service pair.
// Cache miss is HTTP 200 with cached:false and empty analysis (not 404) so browsers and axios do not treat a normal miss as a failed request.
func (h *Handlers) GetQanInsights(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	_, ok := h.checkAdreEnabled(w)
	if !ok {
		return
	}
	queryID := strings.TrimSpace(r.URL.Query().Get("query_id"))
	serviceID := strings.TrimSpace(r.URL.Query().Get("service_id"))
	if queryID == "" || serviceID == "" {
		writeJSONError(w, http.StatusBadRequest, "query_id and service_id are required")
		return
	}
	rows, err := h.db.Query(
		`SELECT analysis, created_at, model, prompt_tokens, completion_tokens, total_tokens, cached_tokens, total_cost
		 FROM qan_insights_cache WHERE query_id = $1 AND service_id = $2 ORDER BY created_at DESC LIMIT 1`,
		queryID, serviceID,
	)
	if err != nil {
		h.l.Errorf("GetQanInsights cache lookup: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to check cache")
		return
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{ //nolint:errchkjson // response already committed
			"analysis": "",
			"cached":   false,
		})
		return
	}
	var analysis string
	var createdAt time.Time
	var model string
	var promptTokens, completionTokens, totalTokens, cachedTokens *int32
	var totalCost *float64
	if err := rows.Scan(&analysis, &createdAt, &model, &promptTokens, &completionTokens, &totalTokens, &cachedTokens, &totalCost); err != nil { //nolint:noinlineerr
		h.l.Errorf("GetQanInsights scan: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to read cache")
		return
	}
	resp := map[string]any{
		"analysis":   analysis,
		"created_at": createdAt.Format(time.RFC3339),
		"cached":     true,
	}
	if model != "" || promptTokens != nil || totalTokens != nil || totalCost != nil {
		u := &HolmesUsage{
			Model:            model,
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      totalTokens,
			CachedTokens:     cachedTokens,
			TotalCost:        totalCost,
		}
		resp["usage"] = HolmesUsageMap(u, model)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp) //nolint:errchkjson // response already committed
}

// PostQanInsightsServiceNow handles POST /v1/adre/qan-insights/servicenow.
func (h *Handlers) PostQanInsightsServiceNow(w http.ResponseWriter, r *http.Request) {
	settings, ok := h.checkAdreEnabled(w)
	if !ok {
		return
	}
	prov, err := models.GetAdreProvisioning(h.db)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to load provisioning")
		return
	}
	if settings.Adre.ServiceNowURL == "" || prov.ServiceNowAPIKey == "" || prov.ServiceNowClientToken == "" {
		writeJSONError(w, http.StatusBadRequest, "ServiceNow is not configured. Set URL, API key, and client token in AI Assistant settings.")
		return
	}
	// Defence in depth: the stored URL was validated on write, but re-assert https before sending secrets.
	if _, err := validators.RequireSecureExternalURL(settings.Adre.ServiceNowURL); err != nil { //nolint:noinlineerr
		writeJSONError(w, http.StatusBadRequest, "servicenow_url: "+err.Error())
		return
	}

	var body struct {
		ServiceID   string `json:"service_id"`
		QueryText   string `json:"query_text"`
		Analysis    string `json:"analysis"`
		QueryID     string `json:"query_id"`
		Fingerprint string `json:"fingerprint"`
		TimeFrom    string `json:"time_from"`
		TimeTo      string `json:"time_to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { //nolint:noinlineerr
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}
	body.ServiceID = strings.TrimSpace(body.ServiceID)
	body.QueryText = strings.TrimSpace(body.QueryText)
	body.Analysis = strings.TrimSpace(body.Analysis)
	if body.ServiceID == "" || body.QueryText == "" || body.Analysis == "" {
		writeJSONError(w, http.StatusBadRequest, "service_id, query_text and analysis are required")
		return
	}

	description := fmt.Sprintf(
		"## QAN AI Insight\n\nService: %s\nQuery ID: %s\nFingerprint: %s\nTime: %s -> %s\n\n### Query\n%s\n\n### Analysis\n%s",
		body.ServiceID, body.QueryID, body.Fingerprint, body.TimeFrom, body.TimeTo, body.QueryText, body.Analysis,
	)
	payload := map[string]string{
		"client_token":      prov.ServiceNowClientToken,
		"short_description": "QAN AI Insight: " + body.ServiceID,
		"description":       description,
		"ticket_type":       "incident",
	}
	reqBody, err := json.Marshal(payload)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to build request")
		return
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, settings.Adre.ServiceNowURL, bytes.NewReader(reqBody))
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to build request")
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Sn-Apikey", prov.ServiceNowAPIKey)
	client := &http.Client{Timeout: 30 * time.Second} //nolint:mnd
	resp, err := client.Do(req)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "ServiceNow request failed: "+err.Error())
		return
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		writeJSONError(w, http.StatusBadGateway, fmt.Sprintf("ServiceNow returned HTTP %d", resp.StatusCode))
		return
	}
	var parsed struct {
		Result struct {
			Success      bool   `json:"success"`
			TicketID     string `json:"ticket_id"`
			Message      string `json:"message"`
			ErrorMessage string `json:"error_message"`
		} `json:"result"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil { //nolint:noinlineerr
		writeJSONError(w, http.StatusBadGateway, "Invalid ServiceNow response")
		return
	}
	if !parsed.Result.Success {
		msg := parsed.Result.ErrorMessage
		if msg == "" {
			msg = parsed.Result.Message
		}
		writeJSONError(w, http.StatusBadGateway, "ServiceNow error: "+msg)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{ //nolint:errchkjson // response already committed
		"success":       true,
		"ticket_id":     parsed.Result.TicketID,
		"ticket_number": parsed.Result.TicketID,
		"message":       parsed.Result.Message,
	})
}

// GetAlerts handles GET /v1/adre/alerts — fetches firing alerts from Grafana's Alertmanager API.
func (h *Handlers) GetAlerts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	_, ok := h.checkAdreEnabled(w)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), h.reqTimeout)
	defer cancel()
	authHeaders := make(http.Header)
	if v := r.Header.Get("Authorization"); v != "" {
		authHeaders.Set("Authorization", v)
	}
	if v := r.Header.Get("Cookie"); v != "" {
		authHeaders.Set("Cookie", v)
	}
	raw, err := h.grafana.GetAlertmanagerAlerts(ctx, authHeaders)
	if err != nil {
		h.l.Warnf("Grafana Alertmanager alerts: %v", err)
		writeJSONError(w, http.StatusBadGateway, fmt.Sprintf("Failed to fetch alerts: %v", err))
		return
	}
	// Grafana returns an array; frontend expects data.alerts or data.data.alerts. Wrap as {"alerts": raw}.
	var alerts json.RawMessage
	if err := json.Unmarshal(raw, &alerts); err != nil { //nolint:noinlineerr
		h.l.Warnf("Parse alerts: %v", err)
		writeJSONError(w, http.StatusBadGateway, "Invalid alerts response")
		return
	}
	out := map[string]json.RawMessage{"alerts": alerts}
	body, err := json.Marshal(out)
	if err != nil {
		h.l.Errorf("Marshal alerts: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to encode response")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(body); err != nil { //nolint:noinlineerr
		h.l.Errorf("Write alerts: %v", err)
	}
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message}) //nolint:errchkjson // response already committed
}
