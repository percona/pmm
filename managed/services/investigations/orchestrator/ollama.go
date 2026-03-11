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

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// OllamaProvider implements LLMProvider by calling Ollama's /api/chat endpoint.
type OllamaProvider struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewOllamaProvider creates an Ollama provider. baseURL is e.g. "http://localhost:11434".
func NewOllamaProvider(baseURL, model string) *OllamaProvider {
	if model == "" {
		model = "llama3.2"
	}
	return &OllamaProvider{
		baseURL: baseURL,
		model:   model,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// ollamaChatRequest is the request body for POST /api/chat.
type ollamaChatRequest struct {
	Model    string            `json:"model"`
	Messages []ollamaMessage   `json:"messages"`
	Tools    []ToolDefinition  `json:"tools,omitempty"`
	Stream   bool              `json:"stream"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

// ollamaChatResponse is the non-streaming response from /api/chat.
type ollamaChatResponse struct {
	Message struct {
		Role      string `json:"role"`
		Content   string `json:"content"`
		ToolCalls []struct {
			Function struct {
				Name      string          `json:"name"`
				Arguments json.RawMessage `json:"arguments"`
			} `json:"function"`
		} `json:"tool_calls"`
	} `json:"message"`
	Done bool `json:"done"`
}

// Complete calls Ollama /api/chat and returns the assistant message and any tool calls.
func (p *OllamaProvider) Complete(ctx context.Context, messages []Message, tools []ToolDefinition) (*CompleteResult, error) {
	ollamaMsgs := make([]ollamaMessage, 0, len(messages))
	for _, m := range messages {
		ollamaMsgs = append(ollamaMsgs, ollamaMessage{
			Role:    m.Role,
			Content: m.Content,
			Name:    m.Name,
		})
	}
	body := ollamaChatRequest{
		Model:    p.model,
		Messages: ollamaMsgs,
		Tools:    tools,
		Stream:   false,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/api/chat", bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama returned %d", resp.StatusCode)
	}
	var chatResp ollamaChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	out := &CompleteResult{
		Content: chatResp.Message.Content,
	}
	for i, tc := range chatResp.Message.ToolCalls {
		id := fmt.Sprintf("call_%d", i)
		if tc.Function.Name == "" {
			continue
		}
		args := string(tc.Function.Arguments)
		if args == "" {
			args = "{}"
		}
		out.ToolCalls = append(out.ToolCalls, ToolCall{
			ID:        id,
			Name:      tc.Function.Name,
			Arguments: args,
		})
	}
	return out, nil
}
