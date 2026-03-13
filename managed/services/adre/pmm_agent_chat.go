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

package adre

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

const pmmAgentMaxTurns = 10

// RunPMMAgentChatStream runs the PMM Agent flow (Holmes with replace_system_prompt), optionally looping on tool_calls (ask_holmes, generate_investigation_report), and streams the final response as SSE.
func RunPMMAgentChatStream(
	w http.ResponseWriter,
	r *http.Request,
	db reform.DBTX,
	l *logrus.Entry,
	settings *models.Settings,
	ask string,
	conversationHistory []interface{},
	streamTimeout time.Duration,
) {
	ctx, cancel := context.WithTimeout(r.Context(), streamTimeout)
	defer cancel()

	trimmed := trimConversationHistory(conversationHistory, settings.Adre.ChatHistoryLength)
	// Avoid duplicate current user message: frontend often sends it in both conversation_history and ask.
	if ask != "" && len(trimmed) > 0 {
		if last, ok := trimmed[len(trimmed)-1].(map[string]interface{}); ok {
			if role, _ := last["role"].(string); role == "user" {
				if content, _ := last["content"].(string); content == ask {
					trimmed = trimmed[:len(trimmed)-1]
				}
			}
		}
	}
	agentPrompt := resolvePMMAgentPrompt(settings)
	client := NewClient(settings.GetAdreURL())

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}
	writeSSE := func(event, data string) {
		if event != "" {
			fmt.Fprintf(w, "event: %s\n", event)
		}
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	history := ensureSystemFirst(trimmed, agentPrompt)
	currentAsk := ask
	investigationPrompt := settings.Adre.InvestigationPrompt
	if investigationPrompt == "" {
		investigationPrompt = DefaultInvestigationPrompt
	}

	for turn := 0; turn < pmmAgentMaxTurns; turn++ {
		req := &ChatRequest{
			Ask:                    currentAsk,
			ConversationHistory:   history,
			ReplaceSystemPrompt:   true,
			AdditionalSystemPrompt: agentPrompt,
			Stream:                false,
		}
		resp, err := client.Chat(ctx, req)
		if err != nil {
			l.Errorf("PMM Agent Chat: %v", err)
			contentJSON, _ := json.Marshal(map[string]string{"content": "PMM Agent failed: " + err.Error()})
			writeSSE("", string(contentJSON))
			return
		}

		if resp.Analysis != "" {
			contentJSON, _ := json.Marshal(map[string]string{"content": resp.Analysis})
			writeSSE("", string(contentJSON))
		}

		toolCalls := parseToolCalls(resp.ToolCalls)
		if len(toolCalls) == 0 {
			return
		}

		// Emit tool call progress and execute tools.
		var toolResults []map[string]interface{}
		for _, tc := range toolCalls {
			startPayload, _ := json.Marshal(map[string]string{
				"id":          tc.ID,
				"tool_name":   tc.Name,
				"description": tc.Name,
			})
			writeSSE("start_tool_calling", string(startPayload))

			result := executePMMAgentTool(ctx, db, l, tc, client, investigationPrompt)
			toolResults = append(toolResults, map[string]interface{}{
				"id":      tc.ID,
				"name":    tc.Name,
				"result":  result,
			})
			resultPayload, _ := json.Marshal(map[string]interface{}{
				"tool_call_id": tc.ID,
				"name":         tc.Name,
				"result":       map[string]string{"data": result},
			})
			writeSSE("tool_calling_result", string(resultPayload))
		}

		// Build next turn: append user message (if any), then assistant message (content + tool_calls), then tool results.
		if currentAsk != "" {
			history = append(history, map[string]interface{}{"role": "user", "content": currentAsk})
		}
		assistantMsg := map[string]interface{}{"role": "assistant", "content": resp.Analysis}
		if len(toolCalls) > 0 {
			tcList := make([]map[string]interface{}, 0, len(toolCalls))
			for _, tc := range toolCalls {
				tcList = append(tcList, map[string]interface{}{
					"id":   tc.ID,
					"type": "function",
					"function": map[string]interface{}{
						"name":      tc.Name,
						"arguments": tc.Arguments,
					},
				})
			}
			assistantMsg["tool_calls"] = tcList
		}
		history = append(history, assistantMsg)
		for _, tr := range toolResults {
			history = append(history, map[string]interface{}{
				"role":        "tool",
				"tool_call_id": tr["id"],
				"content":     tr["result"],
			})
		}
		currentAsk = "" // Continuation: no new user message
	}
}

// toolCall holds a single tool call from the LLM response.
type toolCall struct {
	ID        string
	Name      string
	Arguments string
}

func parseToolCalls(raw []interface{}) []toolCall {
	var out []toolCall
	for _, v := range raw {
		m, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		id, _ := m["id"].(string)
		if id == "" {
			if fc, ok := m["tool_call_id"].(string); ok {
				id = fc
			}
		}
		name, _ := m["name"].(string)
		if name == "" {
			if fn, ok := m["function"].(map[string]interface{}); ok {
				name, _ = fn["name"].(string)
			}
		}
		args, _ := m["arguments"].(string)
		if args == "" {
			if fn, ok := m["function"].(map[string]interface{}); ok {
				args, _ = fn["arguments"].(string)
			}
		}
		if name != "" {
			out = append(out, toolCall{ID: id, Name: name, Arguments: args})
		}
	}
	return out
}

func trimConversationHistory(history []interface{}, maxMessages int) []interface{} {
	if maxMessages <= 0 {
		maxMessages = 20
	}
	if len(history) <= maxMessages {
		return history
	}
	return history[len(history)-maxMessages:]
}

// ensureSystemFirst returns a copy of history that starts with a system message, so Holmes ChatRequest validation passes.
// Holmes requires: "The first item in conversation_history must contain 'role': 'system'".
func ensureSystemFirst(history []interface{}, systemContent string) []interface{} {
	if len(history) > 0 {
		if m, ok := history[0].(map[string]interface{}); ok {
			if role, _ := m["role"].(string); role == "system" {
				return history
			}
		}
	}
	out := make([]interface{}, 0, len(history)+1)
	out = append(out, map[string]interface{}{"role": "system", "content": systemContent})
	out = append(out, history...)
	return out
}

func executePMMAgentTool(ctx context.Context, db reform.DBTX, l *logrus.Entry, tc toolCall, client *Client, investigationPrompt string) string {
	switch tc.Name {
	case "ask_holmes":
		var args struct {
			Messages []map[string]interface{} `json:"messages"`
		}
		if err := json.Unmarshal([]byte(tc.Arguments), &args); err != nil {
			l.Warnf("ask_holmes unmarshal: %v", err)
			return `{"error": "invalid arguments: messages required"}`
		}
		messages := args.Messages
		if len(messages) == 0 {
			return `{"error": "messages cannot be empty"}`
		}
		req := BuildChatRequestForHolmesAgent(messages, investigationPrompt)
		resp, err := client.Chat(ctx, req)
		if err != nil {
			l.Warnf("Holmes Agent (ask_holmes): %v", err)
			b, _ := json.Marshal(map[string]string{"error": "Holmes request failed: " + err.Error()})
			return string(b)
		}
		return resp.Analysis
	case "generate_investigation_report":
		// Stub: return placeholder JSON until Holmes report generator is implemented.
		var args struct {
			Messages []map[string]interface{} `json:"messages"`
			Summary  string                   `json:"summary"`
		}
		_ = json.Unmarshal([]byte(tc.Arguments), &args)
		report := map[string]interface{}{
			"summary":      "Investigation report (placeholder; report generator not yet implemented).",
			"findings":     []interface{}{},
			"recommendations": []interface{}{},
		}
		b, _ := json.Marshal(report)
		return string(b)
	default:
		return fmt.Sprintf(`{"error": "unknown tool: %s"}`, tc.Name)
	}
}
