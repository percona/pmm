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
	"fmt"
	"maps"

	"github.com/percona/pmm/managed/models"
)

// KnownBehaviorControlKeys are Holmes PromptComponent keys accepted in PMM settings PATCH (see Holmes HTTP API).
var KnownBehaviorControlKeys = []string{
	"intro",
	"ask_user",
	"todowrite_instructions", //nolint:goconst
	"todowrite_reminder",
	"ai_safety", // removed in Holmes 0.30 (HolmesGPT #1970); accepted for saved settings, dropped before calling Holmes
	"toolset_instructions",
	"permission_errors",
	"general_instructions",
	"style_guide",
	"cluster_name",
	"system_prompt_additions",
	"files",
	"time_skills",
	"time_runbooks", // legacy; normalized to time_skills before calling Holmes (HolmesGPT #1953)
}

// AdreMaxConversationMessagesDefault caps conversation_history size sent to Holmes (fail-fast overflow mitigation).
const AdreMaxConversationMessagesDefault = 40

// DefaultBehaviorControlsFast disables timed skills catalog injection and TodoWrite for Fast mode (Holmes fast-mode recipe).
// Every preset drops Holmes' general_instructions (generic and Kubernetes investigation guidance, ~2.6k tokens per
// LLM call); the database-relevant rules live in the PMM prompts instead.
func DefaultBehaviorControlsFast() map[string]bool {
	return map[string]bool{
		"time_skills":            false,
		"todowrite_instructions": false,
		"todowrite_reminder":     false,
		"general_instructions":   false,
	}
}

// DefaultBehaviorControlsInvestigation opts into TodoWrite planning, which Holmes disables by default since 0.43 (ROB-574).
func DefaultBehaviorControlsInvestigation() map[string]bool {
	return map[string]bool{
		"todowrite_instructions": true,
		"todowrite_reminder":     true,
		"general_instructions":   false,
	}
}

// DefaultBehaviorControlsFormatReport minimizes prompt noise for the JSON formatting pass.
func DefaultBehaviorControlsFormatReport() map[string]bool {
	return map[string]bool{
		"time_skills":            false,
		"todowrite_instructions": false,
		"todowrite_reminder":     false,
		"general_instructions":   false,
	}
}

// NormalizeBehaviorControlsForHolmes maps legacy keys to Holmes PromptComponent names (HolmesGPT #1953: time_runbooks → time_skills)
// and drops keys Holmes no longer has.
func NormalizeBehaviorControlsForHolmes(m map[string]bool) map[string]bool {
	if len(m) == 0 {
		return m
	}
	out := maps.Clone(m)
	delete(out, "ai_safety")
	if v, ok := out["time_runbooks"]; ok {
		if _, has := out["time_skills"]; !has {
			out["time_skills"] = v
		}
		delete(out, "time_runbooks")
	}
	return out
}

// ResolveBehaviorControlsForPostChat returns behavior_controls for Holmes from settings and UI mode ("fast" or "investigation").
// Empty stored map means use shipped preset (Decision 7).
func ResolveBehaviorControlsForPostChat(settings *models.Settings, mode string) map[string]bool {
	if mode == "investigation" { //nolint:goconst
		return ResolveBehaviorControlsForInvestigation(settings)
	}
	src := settings.Adre.BehaviorControlsFast
	if len(src) == 0 {
		return NormalizeBehaviorControlsForHolmes(DefaultBehaviorControlsFast())
	}
	return NormalizeBehaviorControlsForHolmes(maps.Clone(src))
}

// ResolveBehaviorControlsForInvestigation returns behavior_controls for investigation chat/run: stored keys override the shipped preset.
func ResolveBehaviorControlsForInvestigation(settings *models.Settings) map[string]bool {
	out := DefaultBehaviorControlsInvestigation()
	maps.Copy(out, settings.Adre.BehaviorControlsInvestigation)
	return NormalizeBehaviorControlsForHolmes(out)
}

// ResolveBehaviorControlsForFormatReport returns behavior_controls for FormatInvestigationReport.
func ResolveBehaviorControlsForFormatReport(settings *models.Settings) map[string]bool {
	src := settings.Adre.BehaviorControlsFormatReport
	if len(src) == 0 {
		return NormalizeBehaviorControlsForHolmes(DefaultBehaviorControlsFormatReport())
	}
	return NormalizeBehaviorControlsForHolmes(maps.Clone(src))
}

// MaxConversationMessages returns the effective cap from settings.
func MaxConversationMessages(settings *models.Settings) int {
	n := settings.Adre.AdreMaxConversationMessages
	if n <= 0 {
		return AdreMaxConversationMessagesDefault
	}
	if n < 4 { //nolint:mnd
		return 4 //nolint:mnd
	}
	if n > 200 { //nolint:mnd
		return 200 //nolint:mnd
	}
	return n
}

// TrimConversationHistory keeps the leading system message (if any) and the last N non-system messages,
// preserving order. If there is no system first, only tail is kept (callers should run ensureHolmesLeadingSystemMessage after).
func TrimConversationHistory(hist []any, maxMsgs int) []any {
	if maxMsgs <= 0 || len(hist) <= maxMsgs {
		return hist
	}
	first, ok := hist[0].(map[string]any)
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
	out := make([]any, 0, len(tail)+1)
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
