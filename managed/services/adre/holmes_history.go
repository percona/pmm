// Copyright (C) 2026 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package adre

// HolmesChatLeadingStub is prepended when conversation_history is non-empty but does not start with role=system (Holmes ChatRequest requires it).
const HolmesChatLeadingStub = "PMM session. Full system instructions and Grafana context (if any) are provided via additional_system_prompt."

// EnsureHolmesLeadingSystemMessage ensures the first message in history is role=system when history is non-empty.
func EnsureHolmesLeadingSystemMessage(hist []interface{}) []interface{} {
	if len(hist) == 0 {
		return hist
	}
	first, ok := hist[0].(map[string]interface{})
	if !ok {
		return append([]interface{}{map[string]interface{}{"role": "system", "content": HolmesChatLeadingStub}}, hist...)
	}
	if role, _ := first["role"].(string); role == "system" {
		return hist
	}
	return append([]interface{}{map[string]interface{}{"role": "system", "content": HolmesChatLeadingStub}}, hist...)
}
