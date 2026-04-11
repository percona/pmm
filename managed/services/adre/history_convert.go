// Copyright (C) 2026 Percona LLC
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

package adre

import (
	"strings"

	"github.com/percona/pmm/managed/models"
)

// AdreMessagesToHolmesHistory converts persisted rows (oldest first) to Holmes conversation_history entries.
func AdreMessagesToHolmesHistory(msgs []models.AdreMessage) []interface{} {
	out := make([]interface{}, 0, len(msgs))
	for _, m := range msgs {
		switch m.Role {
		case "tool":
			content := strings.TrimSpace(m.Content)
			if content == "" && len(m.ToolResultJSON) > 0 {
				content = string(m.ToolResultJSON)
			}
			out = append(out, map[string]interface{}{
				"role":    "tool",
				"content": content,
				"name":    m.ToolName,
			})
		default:
			out = append(out, map[string]interface{}{
				"role":    m.Role,
				"content": m.Content,
			})
		}
	}
	return out
}
