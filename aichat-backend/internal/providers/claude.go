package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/aichat-backend/internal/config"
	"github.com/percona/pmm/aichat-backend/internal/models"
)

// ClaudeProvider implements the LLMProvider interface for Anthropic Claude
type ClaudeProvider struct {
	config config.LLMConfig
	apiKey string
	model  string
	client *http.Client
	l      *logrus.Entry
}

const (
	claudeAPIURL       = "https://api.anthropic.com/v1/messages"
	defaultClaudeModel = "claude-3-sonnet-20240229"
)

// NewClaudeProvider creates a new Claude provider
func NewClaudeProvider(cfg config.LLMConfig) (*ClaudeProvider, error) {
	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("Claude API key is required")
	}
	model := cfg.Model
	if model == "" {
		model = defaultClaudeModel
	}

	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &ClaudeProvider{
		config: cfg,
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: timeout},
		l:      logrus.WithField("component", "claude-provider"),
	}, nil
}

// Claude API request/response structures
type claudeTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

type claudeToolUse struct {
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
	Type  string                 `json:"type"`
}

type claudeToolResult struct {
	ToolUseID string      `json:"tool_use_id"`
	Content   interface{} `json:"content"`
	Type      string      `json:"type"`
}

type claudeRequest struct {
	Model     string          `json:"model"`
	Messages  []claudeMessage `json:"messages"`
	Tools     []claudeTool    `json:"tools,omitempty"`
	MaxTokens int             `json:"max_tokens,omitempty"`
	System    string          `json:"system,omitempty"`
	Stream    bool            `json:"stream,omitempty"`
}

type claudeMessage struct {
	Role    string        `json:"role"`
	Content []interface{} `json:"content"`
}

type claudeResponse struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Role       string          `json:"role"`
	Content    []claudeContent `json:"content"`
	Model      string          `json:"model"`
	StopReason string          `json:"stop_reason"`
}

type claudeContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	// For tool use
	ID    string                 `json:"id,omitempty"`
	Name  string                 `json:"name,omitempty"`
	Input map[string]interface{} `json:"input,omitempty"`
	// For image source
	Source *claudeImageSource `json:"source,omitempty"`
}

type claudeImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

// Streaming-specific structures
type claudeStreamEvent struct {
	Type         string          `json:"type"`
	Index        int             `json:"index"`
	Delta        json.RawMessage `json:"delta"`
	Message      claudeResponse  `json:"message"`
	ContentBlock claudeContent   `json:"content_block"`
}

// GenerateStreamResponse generates a streaming response using Anthropic Claude API
func (p *ClaudeProvider) GenerateStreamResponse(ctx context.Context, messages []*models.Message, tools []models.MCPTool) (<-chan *models.StreamMessage, error) {
	// Inline convertMessagesToClaude
	claudeMessages := make([]claudeMessage, 0, len(messages))
	for _, m := range messages {
		role := "user"
		if m.Role == "assistant" {
			role = "assistant"
		}

		var contentParts []interface{}

		if m.Role == "tool" {
			results := make([]interface{}, len(m.ToolExecutions))
			for i, te := range m.ToolExecutions {
				results[i] = claudeToolResult{
					ToolUseID: te.ID,
					Content:   te.Result,
					Type:      "tool_result",
				}
			}
			contentParts = append(contentParts, results...)
		} else {
			if m.Content != "" {
				contentParts = append(contentParts, map[string]string{"type": "text", "text": m.Content})
			}
			if len(m.Attachments) > 0 {
				// Inline convertAttachmentsToClaudeParts
				for _, attachment := range m.Attachments {
					if strings.HasPrefix(attachment.MimeType, "image/") && attachment.Content != "" {
						contentParts = append(contentParts, map[string]interface{}{
							"type": "image",
							"source": &claudeImageSource{
								Type:      "base64",
								MediaType: attachment.MimeType,
								Data:      attachment.Content,
							},
						})
					}
				}
			}
		}

		claudeMessages = append(claudeMessages, claudeMessage{
			Role:    role,
			Content: contentParts,
		})
	}

	// Inline convertToolsToClaude
	claudeTools := make([]claudeTool, len(tools))
	for i, t := range tools {
		claudeTools[i] = claudeTool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
		}
	}

	reqBody := claudeRequest{
		Model:     p.model,
		Messages:  claudeMessages,
		Tools:     claudeTools,
		MaxTokens: 4096,
		Stream:    true,
	}
	if p.config.SystemPrompt != "" {
		reqBody.System = p.config.SystemPrompt
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		p.l.WithError(err).Error("Failed to marshal Claude streaming request")
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", claudeAPIURL, bytes.NewReader(bodyBytes))
	if err != nil {
		p.l.WithError(err).Error("Failed to create Claude API streaming request")
		return nil, err
	}
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("accept", "text/event-stream")
	req.Header.Set("cache-control", "no-cache")
	req.Header.Set("connection", "keep-alive")

	resp, err := p.client.Do(req)
	if err != nil {
		p.l.WithError(err).Error("Claude API streaming request failed")
		return nil, err
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		p.l.WithField("status", resp.StatusCode).WithField("body", string(body)).Error("Claude API streaming error")
		return nil, fmt.Errorf("Claude API streaming error: %s", string(body))
	}

	responseChan := make(chan *models.StreamMessage, 10)
	go p.handleStreamingResponse(resp, responseChan)
	return responseChan, nil
}

func (p *ClaudeProvider) handleStreamingResponse(resp *http.Response, responseChan chan<- *models.StreamMessage) {
	defer close(responseChan)
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		var event claudeStreamEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			p.l.WithError(err).WithField("data", data).Error("Failed to unmarshal stream event")
			continue
		}

		switch event.Type {
		case "content_block_delta":
			var delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}
			if err := json.Unmarshal(event.Delta, &delta); err == nil && delta.Type == "text_delta" {
				responseChan <- &models.StreamMessage{
					Type:    "message",
					Content: delta.Text,
				}
			}
		case "message_stop":
			responseChan <- &models.StreamMessage{
				Type: "done",
				Done: true,
			}
			return
		}
	}

	if err := scanner.Err(); err != nil {
		p.l.WithError(err).Error("Error reading stream")
		responseChan <- &models.StreamMessage{
			Type:  "error",
			Error: err.Error(),
			Done:  true,
		}
	}
}

// Close closes the Claude provider
func (p *ClaudeProvider) Close() error {
	return nil
}
