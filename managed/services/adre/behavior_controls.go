// Copyright (C) 2026 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package adre

import (
	"fmt"
	"maps"

	"github.com/percona/pmm/managed/models"
)

// KnownBehaviorControlKeys are Holmes PromptComponent keys accepted in PMM settings PATCH (see Holmes HTTP API).
var KnownBehaviorControlKeys = []string{
	"intro",
	"ask_user",
	"todowrite_instructions",
	"todowrite_reminder",
	"ai_safety",
	"toolset_instructions",
	"permission_errors",
	"general_instructions",
	"style_guide",
	"cluster_name",
	"system_prompt_additions",
	"files",
	"time_runbooks",
}

// AdreMaxConversationMessagesDefault caps conversation_history size sent to Holmes (fail-fast overflow mitigation).
const AdreMaxConversationMessagesDefault = 40

// DefaultBehaviorControlsFast disables runbooks and TodoWrite for Fast mode (Holmes fast-mode recipe).
func DefaultBehaviorControlsFast() map[string]bool {
	return map[string]bool{
		"time_runbooks":            false,
		"todowrite_instructions":   false,
		"todowrite_reminder":       false,
	}
}

// DefaultBehaviorControlsInvestigation is nil: do not send behavior_controls (Holmes defaults for investigation).
func DefaultBehaviorControlsInvestigation() map[string]bool {
	return nil
}

// DefaultBehaviorControlsFormatReport minimizes prompt noise for the JSON formatting pass.
func DefaultBehaviorControlsFormatReport() map[string]bool {
	return map[string]bool{
		"time_runbooks":          false,
		"todowrite_instructions": false,
		"todowrite_reminder":     false,
	}
}

// ResolveBehaviorControlsForPostChat returns behavior_controls for Holmes from settings and UI mode ("fast" or "investigation").
// Empty stored map means use shipped preset (Decision 7).
func ResolveBehaviorControlsForPostChat(settings *models.Settings, mode string) map[string]bool {
	if mode == "investigation" {
		src := settings.Adre.BehaviorControlsInvestigation
		if len(src) == 0 {
			return nil
		}
		return maps.Clone(src)
	}
	src := settings.Adre.BehaviorControlsFast
	if len(src) == 0 {
		return DefaultBehaviorControlsFast()
	}
	return maps.Clone(src)
}

// ResolveBehaviorControlsForInvestigation returns behavior_controls for investigation chat/run.
func ResolveBehaviorControlsForInvestigation(settings *models.Settings) map[string]bool {
	src := settings.Adre.BehaviorControlsInvestigation
	if len(src) == 0 {
		return DefaultBehaviorControlsInvestigation()
	}
	return maps.Clone(src)
}

// ResolveBehaviorControlsForFormatReport returns behavior_controls for FormatInvestigationReport.
func ResolveBehaviorControlsForFormatReport(settings *models.Settings) map[string]bool {
	src := settings.Adre.BehaviorControlsFormatReport
	if len(src) == 0 {
		return DefaultBehaviorControlsFormatReport()
	}
	return maps.Clone(src)
}

// MaxConversationMessages returns the effective cap from settings.
func MaxConversationMessages(settings *models.Settings) int {
	n := settings.Adre.AdreMaxConversationMessages
	if n <= 0 {
		return AdreMaxConversationMessagesDefault
	}
	if n < 4 {
		return 4
	}
	if n > 200 {
		return 200
	}
	return n
}

// TrimConversationHistory keeps the leading system message (if any) and the last N non-system messages,
// preserving order. If there is no system first, only tail is kept (callers should run ensureHolmesLeadingSystemMessage after).
func TrimConversationHistory(hist []interface{}, maxMsgs int) []interface{} {
	if maxMsgs <= 0 || len(hist) <= maxMsgs {
		return hist
	}
	first, ok := hist[0].(map[string]interface{})
	hasSystemFirst := ok && first != nil
	if role, _ := first["role"].(string); !hasSystemFirst || role != "system" {
		// No leading system: keep last maxMsgs entries as-is.
		return hist[len(hist)-maxMsgs:]
	}
	rest := hist[1:]
	if len(rest) <= maxMsgs-1 {
		return hist
	}
	tail := rest[len(rest)-(maxMsgs-1):]
	out := make([]interface{}, 0, len(tail)+1)
	out = append(out, first)
	out = append(out, tail...)
	return out
}

// ValidateBehaviorControlsMap returns an error if any key is unknown.
func ValidateBehaviorControlsMap(m map[string]bool) error {
	if len(m) == 0 {
		return nil
	}
	allowed := make(map[string]struct{}, len(KnownBehaviorControlKeys))
	for _, k := range KnownBehaviorControlKeys {
		allowed[k] = struct{}{}
	}
	for k := range m {
		if _, ok := allowed[k]; !ok {
			return fmt.Errorf("unknown behavior_controls key %q", k)
		}
	}
	return nil
}
