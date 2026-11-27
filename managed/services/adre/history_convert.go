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
	"github.com/percona/pmm/managed/models"
)

// MessagesToHolmesHistory converts persisted rows (oldest first) to Holmes conversation_history entries.
func MessagesToHolmesHistory(msgs []models.AdreMessage) []any {
	out := make([]any, 0, len(msgs))
	for _, m := range msgs {
		switch m.Role {
		case "tool":
			// Do not replay persisted tool rows into conversation_history.
			// OpenAI (and LiteLLM) require each role=tool message to follow an assistant message
			// that includes matching tool_calls; we only persist plain assistant text plus
			// separate tool result rows, so replaying "tool" here causes 400 errors.
		default:
			out = append(out, map[string]any{
				"role":    m.Role,    //nolint:goconst
				"content": m.Content, //nolint:goconst
			})
		}
	}
	return out
}
