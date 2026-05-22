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

// HolmesChatLeadingStub is prepended when conversation_history is non-empty but does not start with role=system (Holmes ChatRequest requires it).
const HolmesChatLeadingStub = "PMM session. Full system instructions and Grafana context (if any) are provided via additional_system_prompt."

// EnsureHolmesLeadingSystemMessage ensures the first message in history is role=system when history is non-empty.
func EnsureHolmesLeadingSystemMessage(hist []any) []any {
	if len(hist) == 0 {
		return hist
	}
	first, ok := hist[0].(map[string]any)
	if !ok {
		return append([]any{map[string]any{"role": "system", "content": HolmesChatLeadingStub}}, hist...) //nolint:goconst
	}
	if role, _ := first["role"].(string); role == "system" {
		return hist
	}
	return append([]any{map[string]any{"role": "system", "content": HolmesChatLeadingStub}}, hist...)
}
