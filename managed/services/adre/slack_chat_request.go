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

// SlackAnswerGuidance is appended to the Slack chat prompt: a thread reply is read in a narrow pane,
// often on a phone, so it leads with the answer and stays short.
const SlackAnswerGuidance = `Slack (this reply goes to a Slack thread; this overrides the substantive reply format above):
- Lead with the answer in 1–3 sentences, without a heading before it. Add a short ### heading or a **bold** label only when the reply has distinct parts.
- Keep the whole reply to about 25 lines. Use at most one table, with at most 4 columns; summarize anything more in a sentence and offer the detail.
- Leave out IDs (service_id, node_id, agent_id) unless the user asks for them or the next step needs one.
- Use backticks only for commands, queries and literal values, not for service or node names.
- Put SQL and shell commands in fenced code blocks.
- End with at most one follow-up question.`

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
		AdditionalSystemPrompt: strings.TrimRight(ResolveChatSystemPrompt(settings, mode), "\n") + "\n\n" + SlackAnswerGuidance,
	}
	if extra := strings.TrimSpace(extraSystemPrompt); extra != "" {
		req.AdditionalSystemPrompt = strings.TrimRight(req.AdditionalSystemPrompt, "\n") + "\n\n" + extra
	}
	return req
}
