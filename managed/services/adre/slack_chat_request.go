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
	"strings"

	"github.com/percona/pmm/managed/models"
)

// SlackChatMode maps UI default chat mode to Holmes chat modes for the Slack bot (same rules as POST /v1/adre/chat when mode is omitted).
func SlackChatMode(settings *models.Settings) string {
	if strings.TrimSpace(settings.Adre.DefaultChatMode) == "investigation" { //nolint:goconst
		return "investigation"
	}
	return "fast" //nolint:goconst
}

// BuildSlackChatRequest builds a non-streaming Holmes /api/chat body for the PMM Slack integration (mirrors POST /v1/adre/chat without dashboard_context).
func BuildSlackChatRequest(settings *models.Settings, ask string, history []any, extraSystemPrompt string) *ChatRequest {
	mode := SlackChatMode(settings)
	maxN := MaxConversationMessages(settings)
	hist := TrimConversationHistory(history, maxN)
	hist = EnsureHolmesLeadingSystemMessage(hist)
	req := &ChatRequest{
		Ask:                    ask,
		ConversationHistory:    hist,
		Model:                  resolveChatModel(settings, mode, ""),
		BehaviorControls:       ResolveBehaviorControlsForPostChat(settings, mode),
		AdditionalSystemPrompt: ResolveChatSystemPrompt(settings, mode),
	}
	if extra := strings.TrimSpace(extraSystemPrompt); extra != "" {
		req.AdditionalSystemPrompt = strings.TrimRight(req.AdditionalSystemPrompt, "\n") + "\n\n" + extra
	}
	return req
}
