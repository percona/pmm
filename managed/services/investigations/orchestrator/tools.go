// Copyright (C) 2025 Percona LLC
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

package orchestrator

// DefaultSystemPrompt is the system prompt for the PMM Investigations assistant.
const DefaultSystemPrompt = `You are the PMM Investigations assistant. You help users investigate incidents and understand their PMM data (metrics, logs, queries, dashboards).

You have access to tools:
- Use get_investigation_context to read the current investigation summary, time range, and block list.
- Use append_block to add a finding, summary, or evidence block to the incident page.
- When the user asks to investigate an issue, analyze a query, explain a metric, diagnose performance, or answer questions about their PMM data, use the holmes_investigate tool (when available) with the user's question and incident context. Do not use it for general knowledge; only for observability/database-related questions.

For general questions not about their data, answer from your own knowledge. Do not forward every message to HolmesGPT; only call it when the request is clearly observability-related.

Be concise and evidence-driven.`

// RunReportSystemPrompt is used when the user explicitly runs "Generate report".
const RunReportSystemPrompt = `You are building a full investigation report for this incident. Use get_investigation_context to read the current state, then use append_block to add blocks (type: summary, markdown, finding) with the report content. Add a short summary block at the top, then any findings or details. When you have added all blocks, respond with a brief final message and do not call more tools.`

// DefaultToolRegistry returns the default set of tools for the orchestrator.
// holmes_investigate is added in phase3 when the HolmesGPT adapter is wired.
func DefaultToolRegistry(includeHolmes bool) []ToolDefinition {
	tools := []ToolDefinition{
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "get_investigation_context",
				Description: "Use to read the current investigation summary, time range, and block list.",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "append_block",
				Description: "Use to add a finding, summary, or evidence block to the incident page. Pass type (e.g. markdown, finding, summary), title, position, and optional data_json with block content.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"type": map[string]interface{}{
							"type":        "string",
							"description": "Block type: markdown, summary, finding, etc.",
						},
						"title": map[string]interface{}{
							"type":        "string",
							"description": "Optional block title",
						},
						"position": map[string]interface{}{
							"type":        "integer",
							"description": "Order position (0-based)",
						},
						"data_json": map[string]interface{}{
							"type":        "object",
							"description": "Block content, e.g. {\"content\": \"...\"} for markdown",
						},
					},
					"required": []interface{}{"type"},
				},
			},
		},
	}
	if includeHolmes {
		tools = append(tools, ToolDefinition{
			Type: "function",
			Function: ToolFunction{
				Name:        "holmes_investigate",
				Description: "Use when the user asks to investigate an issue, analyze a query, explain a metric, diagnose performance, run observability or database diagnostics, or answer questions about their PMM data. Pass the user's question and the current incident context. Do not use for general knowledge or for updating the incident page structure.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"question": map[string]interface{}{
							"type":        "string",
							"description": "The specific question or focus for this investigation request",
						},
					},
					"required": []interface{}{"question"},
				},
			},
		})
	}
	return tools
}
