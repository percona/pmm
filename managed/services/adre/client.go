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

// Package adre implements the ADRE (Autonomous Database Reliability Engineer) / HolmesGPT integration.
package adre

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/managed/models"
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

// tlsSkipVerifyWarnOnce guards the per-process "TLS verification disabled" warning, since clients are
// rebuilt on every request.
var tlsSkipVerifyWarnOnce sync.Once

// NewClient creates a new HolmesGPT API client with default TLS verification.
// baseURL may include credentials for Basic Auth: http://user:password@host:port
func NewClient(baseURL string) *Client {
	return newClient(baseURL, false)
}

// NewClientFromSettings creates a HolmesGPT client using ADRE URL and TLS settings from PMM settings.
func NewClientFromSettings(settings *models.Settings) *Client {
	if settings == nil {
		return newClient("", false)
	}
	return newClient(settings.GetAdreURL(), settings.Adre.TLSSkipVerify)
}

func newClient(baseURL string, tlsSkipVerify bool) *Client {
	baseURL = strings.TrimSuffix(baseURL, "/")
	authHeader := ""
	u, err := url.Parse(baseURL)
	if err == nil && u.User != nil {
		password, hasPass := u.User.Password()
		if hasPass {
			user := u.User.Username()
			authHeader = "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+password))
			// Strip credentials from baseURL for logging/requests (we add auth via header)
			u.User = nil
			baseURL = u.String()
		}
	}
	l := logrus.WithField("component", "adre")
	httpClient := &http.Client{Timeout: defaultTimeout}
	// Only override the transport when skipping verification; otherwise reuse the
	// shared http.DefaultTransport so connections are pooled across requests.
	if tlsSkipVerify {
		// Clients are built per request, so warn once per process to avoid log spam.
		tlsSkipVerifyWarnOnce.Do(func() {
			l.Warn("ADRE/HolmesGPT TLS certificate verification is DISABLED (tls_skip_verify); the connection " +
				"is encrypted but the server is not authenticated — only use for trusted internal endpoints")
		})
		transport := http.DefaultTransport.(*http.Transport).Clone()      //nolint:forcetypeassert
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // intentional when admin enables tls_skip_verify
		httpClient.Transport = transport
	}
	return &Client{
		baseURL:    baseURL,
		authHeader: authHeader,
		httpClient: httpClient,
		l:          l,
	}
}

// setAuth adds Authorization header to the request if client has auth configured.
//
//nolint:funcorder // grouped with constructor; reads better than method-visibility ordering
func (c *Client) setAuth(req *http.Request) {
	if c.authHeader != "" {
		req.Header.Set("Authorization", c.authHeader)
	}
}

// url joins baseURL with the given path.
//
//nolint:funcorder // grouped with constructor; reads better than method-visibility ordering
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
	defer func() { _ = resp.Body.Close() }()

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
	if err := json.Unmarshal(rawBody, &out); err == nil { //nolint:noinlineerr
		return out.ModelName, nil
	}

	// Backward compatibility for older Holmes responses where model_name was JSON-encoded as a string.
	var legacy struct {
		ModelName string `json:"model_name"`
	}
	if err := json.Unmarshal(rawBody, &legacy); err != nil { //nolint:noinlineerr
		return nil, err
	}
	if strings.TrimSpace(legacy.ModelName) == "" {
		return []string{}, nil
	}
	var legacyModels []string
	if err := json.Unmarshal([]byte(legacy.ModelName), &legacyModels); err != nil { //nolint:noinlineerr
		return nil, err
	}
	return legacyModels, nil
}

// ChatRequest is the request body for POST /api/chat.
type ChatRequest struct {
	Ask                    string `json:"ask"`
	ConversationHistory    []any  `json:"conversation_history,omitempty"`
	Model                  string `json:"model,omitempty"`
	Stream                 bool   `json:"stream,omitempty"`
	AdditionalSystemPrompt string `json:"additional_system_prompt,omitempty"`
	PageContext            any    `json:"page_context,omitempty"`
	FrontendTools          []any  `json:"frontend_tools,omitempty"`
	FrontendToolResults    []any  `json:"frontend_tool_results,omitempty"`
	ToolDecisions          []any  `json:"tool_decisions,omitempty"`
	// BehaviorControls overrides Holmes prompt components
	// (e.g. {"time_skills": false, "todowrite_instructions": false}).
	// Keys must match holmes/core/prompt.py PromptComponent values. Optional.
	BehaviorControls map[string]bool `json:"behavior_controls,omitempty"`
}

// ChatResponse is the response from POST /api/chat.
type ChatResponse struct {
	Analysis            string          `json:"analysis"`
	ConversationHistory []any           `json:"conversation_history,omitempty"`
	ToolCalls           []any           `json:"tool_calls,omitempty"`
	FollowUpActions     []any           `json:"follow_up_actions,omitempty"`
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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HolmesGPT /api/chat: %s: %s", resp.Status, string(respBody))
	}

	var out ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil { //nolint:noinlineerr
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
		_ = resp.Body.Close()
		return nil, fmt.Errorf("HolmesGPT /api/chat (stream): %s: %s", resp.Status, string(respBody))
	}
	return resp.Body, nil
}
