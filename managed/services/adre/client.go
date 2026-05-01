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

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var out struct {
		ModelName []string `json:"model_name"`
	}
	if err := json.Unmarshal(rawBody, &out); err == nil {
		return out.ModelName, nil
	}

	// Backward compatibility for older Holmes responses where model_name was JSON-encoded as a string.
	var legacy struct {
		ModelName string `json:"model_name"`
	}
	if err := json.Unmarshal(rawBody, &legacy); err != nil {
		return nil, err
	}
	if strings.TrimSpace(legacy.ModelName) == "" {
		return []string{}, nil
	}
	var legacyModels []string
	if err := json.Unmarshal([]byte(legacy.ModelName), &legacyModels); err != nil {
		return nil, err
	}
	return legacyModels, nil
}

// ChatRequest is the request body for POST /api/chat.
type ChatRequest struct {
	Ask                    string        `json:"ask"`
	ConversationHistory    []interface{} `json:"conversation_history,omitempty"`
	Model                  string        `json:"model,omitempty"`
	Stream                 bool          `json:"stream,omitempty"`
	AdditionalSystemPrompt string        `json:"additional_system_prompt,omitempty"`
	PageContext            interface{}   `json:"page_context,omitempty"`
	FrontendTools          []interface{} `json:"frontend_tools,omitempty"`
	FrontendToolResults    []interface{} `json:"frontend_tool_results,omitempty"`
	ToolDecisions          []interface{} `json:"tool_decisions,omitempty"`
	// BehaviorControls overrides Holmes prompt components (e.g. {"time_skills": false, "todowrite_instructions": false}). Keys must match holmes/core/prompt.py PromptComponent values. Optional.
	BehaviorControls map[string]bool `json:"behavior_controls,omitempty"`
}

// ChatResponse is the response from POST /api/chat.
type ChatResponse struct {
	Analysis            string          `json:"analysis"`
	ConversationHistory []interface{}   `json:"conversation_history,omitempty"`
	ToolCalls           []interface{}   `json:"tool_calls,omitempty"`
	FollowUpActions     []interface{}   `json:"follow_up_actions,omitempty"`
	Metadata            json.RawMessage `json:"metadata,omitempty"`
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
