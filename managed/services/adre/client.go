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

// Package adre implements the ADRE (Autonomous Database Reliability Engineer) / HolmesGPT integration.
package adre

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	defaultTimeout = 60 * time.Second
	streamTimeout  = 5 * time.Minute
)

// Client is an HTTP client for the HolmesGPT API.
type Client struct {
	baseURL    string
	authHeader string // "Authorization: Basic xxx" or "Authorization: Bearer xxx", empty if no auth
	httpClient *http.Client
	l          *logrus.Entry
}

// NewClient creates a new HolmesGPT API client.
// baseURL may include credentials for Basic Auth: http://user:password@host:port
func NewClient(baseURL string) *Client {
	baseURL = strings.TrimSuffix(baseURL, "/")
	authHeader := ""
	if u, err := url.Parse(baseURL); err == nil && u.User != nil {
		password, hasPass := u.User.Password()
		if hasPass {
			user := u.User.Username()
			authHeader = "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+password))
			// Strip credentials from baseURL for logging/requests (we add auth via header)
			u.User = nil
			baseURL = u.String()
		}
	}
	return &Client{
		baseURL:    baseURL,
		authHeader: authHeader,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		l: logrus.WithField("component", "adre"),
	}
}

// setAuth adds Authorization header to the request if client has auth configured.
func (c *Client) setAuth(req *http.Request) {
	if c.authHeader != "" {
		req.Header.Set("Authorization", c.authHeader)
	}
}

// url joins baseURL with the given path.
func (c *Client) url(p string) string {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return c.baseURL + p
	}
	u.Path = path.Join(u.Path, p)
	return u.String()
}

// Models returns the list of available models from HolmesGPT.
func (c *Client) Models(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url("/api/model"), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HolmesGPT /api/model: %s: %s", resp.Status, string(body))
	}

	var out struct {
		ModelName []string `json:"model_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.ModelName, nil
}

// ChatRequest is the request body for POST /api/chat.
type ChatRequest struct {
	Ask                    string        `json:"ask"`
	ConversationHistory    []interface{} `json:"conversation_history,omitempty"`
	Model                  string        `json:"model,omitempty"`
	Stream                 bool          `json:"stream,omitempty"`
	AdditionalSystemPrompt string        `json:"additional_system_prompt,omitempty"`
	PageContext            interface{}   `json:"page_context,omitempty"`
	// ReplaceSystemPrompt: when true, Holmes uses only AdditionalSystemPrompt as system message (PMM Agent mode).
	ReplaceSystemPrompt bool `json:"replace_system_prompt,omitempty"`
	// BehaviorControls overrides prompt components (e.g. {"todowrite_instructions": false}). Optional.
	BehaviorControls map[string]bool `json:"behavior_controls,omitempty"`
}

// ChatResponse is the response from POST /api/chat.
type ChatResponse struct {
	Analysis            string        `json:"analysis"`
	ConversationHistory []interface{} `json:"conversation_history,omitempty"`
	ToolCalls           []interface{} `json:"tool_calls,omitempty"`
	FollowUpActions     []interface{} `json:"follow_up_actions,omitempty"`
}

// Chat sends a chat request to HolmesGPT (non-streaming).
func (c *Client) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url("/api/chat"), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	c.setAuth(httpReq)

	client := *c.httpClient
	client.Timeout = streamTimeout
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HolmesGPT /api/chat: %s: %s", resp.Status, string(respBody))
	}

	var out ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ChatStream sends a chat request and returns the response body for streaming (SSE).
// Caller must close the returned ReadCloser.
func (c *Client) ChatStream(ctx context.Context, req *ChatRequest) (io.ReadCloser, error) {
	// Copy request so we can set Stream=true without mutating the caller's req
	streamReq := *req
	streamReq.Stream = true
	body, err := json.Marshal(&streamReq)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url("/api/chat"), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	c.setAuth(httpReq)

	client := *c.httpClient
	client.Timeout = streamTimeout
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("HolmesGPT /api/chat (stream): %s: %s", resp.Status, string(respBody))
	}
	return resp.Body, nil
}

// contentString extracts a string from a message "content" field (may be string or other type in some APIs).
func contentString(m map[string]interface{}) string {
	if c, ok := m["content"].(string); ok {
		return c
	}
	if c := m["content"]; c != nil {
		return fmt.Sprintf("%v", c)
	}
	return ""
}

// BuildChatRequestForHolmesAgent builds a ChatRequest for the Holmes Agent (default prompt) with multi-turn messages.
// messages is the sub-conversation (e.g. from ask_holmes tool); each item has "role" and "content".
// conversation_history is messages[0:len-1], ask is the last message's content. additionalSystemPrompt is the Chat/Investigation prompt.
func BuildChatRequestForHolmesAgent(messages []map[string]interface{}, additionalSystemPrompt string) *ChatRequest {
	req := &ChatRequest{
		AdditionalSystemPrompt: additionalSystemPrompt,
		Stream:                 false,
	}
	if len(messages) == 0 {
		return req
	}
	if len(messages) == 1 {
		req.Ask = contentString(messages[0])
		return req
	}
	last := messages[len(messages)-1]
	req.Ask = contentString(last)
	history := make([]interface{}, 0, len(messages)-1)
	for i := 0; i < len(messages)-1; i++ {
		history = append(history, messages[i])
	}
	req.ConversationHistory = history
	return req
}

// InvestigateRequest is the request body for POST /api/investigate.
// AdditionalSystemPrompt is sent if Holmes API supports it for investigation.
type InvestigateRequest struct {
	Source                 string      `json:"source"`
	Title                  string      `json:"title"`
	Description            string      `json:"description"`
	Subject                interface{} `json:"subject,omitempty"`
	Context                interface{} `json:"context,omitempty"`
	IncludeToolCalls       bool        `json:"include_tool_calls,omitempty"`
	Model                  string      `json:"model,omitempty"`
	Stream                 bool        `json:"stream,omitempty"`
	AdditionalSystemPrompt string      `json:"additional_system_prompt,omitempty"`
}

// InvestigateResponse is the response from POST /api/investigate.
type InvestigateResponse struct {
	Analysis     string            `json:"analysis"`
	Sections     map[string]string `json:"sections,omitempty"`
	ToolCalls    []interface{}     `json:"tool_calls,omitempty"`
	Instructions []interface{}     `json:"instructions,omitempty"`
}

// Investigate sends an investigate request to HolmesGPT (non-streaming).
func (c *Client) Investigate(ctx context.Context, req *InvestigateRequest) (*InvestigateResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url("/api/investigate"), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	c.setAuth(httpReq)

	// Investigation can take several minutes; use same timeout as stream to avoid 504.
	httpClient := *c.httpClient
	httpClient.Timeout = streamTimeout
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HolmesGPT /api/investigate: %s: %s", resp.Status, string(respBody))
	}

	var out InvestigateResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

// InvestigateStream sends an investigate request and returns the response body for streaming (SSE).
// Uses /api/investigate with stream: true (same pattern as ChatStream).
// Caller must close the returned ReadCloser.
func (c *Client) InvestigateStream(ctx context.Context, req *InvestigateRequest) (io.ReadCloser, error) {
	streamReq := *req
	streamReq.Stream = true
	body, err := json.Marshal(&streamReq)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url("/api/investigate"), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	c.setAuth(httpReq)

	client := *c.httpClient
	client.Timeout = streamTimeout
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("HolmesGPT /api/investigate (stream): %s: %s", resp.Status, string(respBody))
	}
	return resp.Body, nil
}
