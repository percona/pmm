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

// Package orchestrator provides the LLM orchestrator for PMM AI Investigations.
// The orchestrator calls a configurable LLM (e.g. Ollama) with a tool registry;
// one of the tools is "call HolmesGPT for investigation."
package orchestrator

import (
	"context"
)

// Message represents a single chat message (user, assistant, or tool).
type Message struct {
	Role    string `json:"role"` // "user", "assistant", "system", "tool"
	Content string `json:"content"`
	// Name is set when role is "tool" to identify which tool produced the content.
	Name string `json:"name,omitempty"`
}

// ToolDefinition describes a tool the LLM can call.
type ToolDefinition struct {
	Type     string       `json:"type"` // "function"
	Function ToolFunction `json:"function"`
}

// ToolFunction describes the function name and parameters for a tool.
type ToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// ToolCall represents a tool invocation requested by the LLM.
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// CompleteResult is the result of an LLM completion.
type CompleteResult struct {
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// LLMProvider is the interface for the configurable orchestrator LLM (e.g. Ollama, OpenAI).
// The orchestrator calls Complete with the conversation history and available tools;
// the provider returns the model response and any tool calls.
type LLMProvider interface {
	Complete(ctx context.Context, messages []Message, tools []ToolDefinition) (*CompleteResult, error)
}
