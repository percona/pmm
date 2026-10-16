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

import "encoding/json"

// holmesHistoryMaxBytes caps the replayed Holmes history (system message excluded).
const holmesHistoryMaxBytes = 400 << 10

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

// decodeHolmesHistory parses a stored Holmes conversation_history; nil unless it starts with the system message.
func decodeHolmesHistory(raw json.RawMessage) []any {
	if len(raw) == 0 {
		return nil
	}
	var hist []any
	err := json.Unmarshal(raw, &hist)
	if err != nil || len(hist) < 2 || holmesMessageRole(hist[0]) != "system" {
		return nil
	}
	return hist
}

func holmesMessageRole(m any) string {
	mm, _ := m.(map[string]any)
	role, _ := mm["role"].(string)
	return role
}

// TrimHolmesHistory drops the oldest whole turns (a turn starts at a user message, so tool calls keep their results)
// once the history exceeds maxMsgs messages or maxBytes, down to half of both caps. Trimming in steps instead of a
// sliding window keeps the prompt prefix, and so the provider's prompt cache, stable between trims.
func TrimHolmesHistory(hist []any, maxMsgs, maxBytes int) []any {
	suffix := make([]int, len(hist)+1)
	for i := len(hist) - 1; i >= 1; i-- {
		b, err := json.Marshal(hist[i])
		if err != nil {
			b = nil
		}
		suffix[i] = suffix[i+1] + len(b)
	}
	if len(hist) <= maxMsgs && suffix[1] <= maxBytes {
		return hist
	}
	cut := 0
	for i := 1; i < len(hist); i++ {
		if holmesMessageRole(hist[i]) != "user" {
			continue
		}
		cut = i
		if len(hist)-i <= maxMsgs/2 && suffix[i] <= maxBytes/2 {
			break
		}
	}
	if cut <= 1 {
		return hist
	}
	return append([]any{hist[0]}, hist[cut:]...)
}
