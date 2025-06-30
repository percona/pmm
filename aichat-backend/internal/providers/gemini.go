package providers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/genai"

	"github.com/google/uuid"
	"github.com/percona/pmm/aichat-backend/internal/config"
	"github.com/percona/pmm/aichat-backend/internal/models"
	"github.com/sirupsen/logrus"
)

// GeminiProvider implements the LLMProvider interface for Google Gemini
type GeminiProvider struct {
	client *genai.Client
	config config.LLMConfig
	l      *logrus.Entry
}

// NewGeminiProvider creates a new Gemini provider
func NewGeminiProvider(cfg config.LLMConfig) (*GeminiProvider, error) {
	l := logrus.WithField("component", "gemini-provider")

	l.WithField("model", cfg.Model).Info("Initializing Gemini provider")

	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: cfg.APIKey,
	})
	if err != nil {
		l.WithError(err).Error("Failed to create Gemini client")
		return nil, fmt.Errorf("failed to initialize Gemini client: %w", err)
	}

	return &GeminiProvider{
		client: client,
		config: cfg,
		l:      l,
	}, nil
}

// convertMCPToolsToGemini converts MCP tools to Gemini function declarations
func (p *GeminiProvider) convertMCPToolsToGemini(tools []models.MCPTool) []*genai.FunctionDeclaration {
	if len(tools) == 0 {
		return nil
	}

	functions := make([]*genai.FunctionDeclaration, 0, len(tools))

	for _, tool := range tools {
		// Convert input schema to Gemini format
		parameters := &genai.Schema{
			Type: genai.TypeObject,
		}
		m, err := json.Marshal(tool.InputSchema)
		if err != nil {
			p.l.WithError(err).Error("Failed to marshal tool input schema")
			continue
		}
		err = json.Unmarshal(m, &parameters)
		if err != nil {
			p.l.WithError(err).Error("Failed to unmarshal tool input schema")
			continue
		}

		function := &genai.FunctionDeclaration{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  parameters,
		}

		functions = append(functions, function)
	}
	return functions
}

// convertMessagesToGemini converts []*models.Message to []*genai.Content for Gemini API
func (p *GeminiProvider) convertMessagesToGemini(messages []*models.Message) []*genai.Content {
	var contents []*genai.Content
	for _, msg := range messages {
		role := genai.RoleUser
		if msg.Role == "assistant" {
			role = genai.RoleModel
		}

		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			parts := []*genai.Part{}
			if msg.Content != "" {
				parts = append(parts, genai.NewPartFromText(msg.Content))
			}

			for _, toolCall := range msg.ToolCalls {
				var args map[string]interface{}
				if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
					p.l.WithError(err).Error("Failed to parse function arguments")
					continue
				}

				functionCallPart := genai.NewPartFromFunctionCall(toolCall.Function.Name, args)
				parts = append(parts, functionCallPart)
			}

			content := &genai.Content{
				Role:  role,
				Parts: parts,
			}
			contents = append(contents, content)
		} else if msg.Role == "tool" {
			if msg.Content == "" {
				continue
			}

			parts := []*genai.Part{genai.NewPartFromText(msg.Content)}

			content := &genai.Content{
				Role:  genai.RoleUser,
				Parts: parts,
			}
			contents = append(contents, content)
		} else {
			if msg.Content == "" {
				continue
			}

			parts := []*genai.Part{genai.NewPartFromText(msg.Content)}

			if msg.Role == "user" && len(msg.Attachments) > 0 {
				attachmentParts := p.convertAttachmentsToGeminiParts(msg.Attachments)
				parts = append(parts, attachmentParts...)
			}

			content := &genai.Content{
				Role:  role,
				Parts: parts,
			}
			contents = append(contents, content)
		}
	}
	return contents
}

// GenerateStreamResponse generates a streaming response using Google Gemini API
func (p *GeminiProvider) GenerateStreamResponse(ctx context.Context, messages []*models.Message, tools []models.MCPTool) (<-chan *models.StreamMessage, error) {
	// Convert MCP tools to Gemini functions
	var toolConfig *genai.GenerateContentConfig
	functions := p.convertMCPToolsToGemini(tools)
	if len(functions) > 0 {
		toolConfig = &genai.GenerateContentConfig{
			Tools: []*genai.Tool{{
				FunctionDeclarations: functions,
			}},
			Temperature: genai.Ptr(float32(0.7)),
		}
	} else {
		toolConfig = &genai.GenerateContentConfig{
			Temperature: genai.Ptr(float32(0.7)),
		}
	}

	// Use the new helper to convert messages
	contents := p.convertMessagesToGemini(messages)

	// Create response channel
	responseChan := make(chan *models.StreamMessage, 10)

	// Start goroutine to handle streaming
	go func() {
		defer close(responseChan)

		var totalContent string
		var detectedToolCalls []models.ToolCall

		// Use GenerateContentStream for streaming
		streamIter := p.client.Models.GenerateContentStream(ctx, p.config.Model, contents, toolConfig)

		// Process streaming response
		for resp, err := range streamIter {
			if err != nil {
				p.l.WithError(err).Error("Streaming error")
				responseChan <- &models.StreamMessage{
					Type:    "error",
					Content: "",
					Done:    true,
					Error:   fmt.Sprintf("Gemini streaming error: %v", err),
				}
				return
			}

			if len(resp.Candidates) == 0 {
				continue
			}

			candidate := resp.Candidates[0]
			if candidate.Content == nil || len(candidate.Content.Parts) == 0 {
				continue
			}

			// Process parts in this chunk
			for _, part := range candidate.Content.Parts {
				if part.Text != "" {
					totalContent += part.Text
					responseChan <- &models.StreamMessage{
						Type:    "message",
						Content: part.Text,
						Done:    false,
					}
				}
				if part.FunctionCall != nil {
					// Convert to tool call
					argsBytes, err := json.Marshal(part.FunctionCall.Args)
					if err != nil {
						p.l.WithError(err).Error("Failed to marshal function arguments")
						continue
					}

					toolCall := models.ToolCall{
						ID:   fmt.Sprintf("call_%s", uuid.New().String()),
						Type: "function",
						Function: struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						}{
							Name:      part.FunctionCall.Name,
							Arguments: string(argsBytes),
						},
					}
					detectedToolCalls = append(detectedToolCalls, toolCall)
				}
			}
		}

		// Send final message with tool calls if any were detected
		finalMessage := &models.StreamMessage{
			Type:      "done",
			Content:   totalContent,
			Done:      true,
			ToolCalls: detectedToolCalls,
		}

		responseChan <- finalMessage
	}()

	return responseChan, nil
}

// Close closes the Gemini client
func (p *GeminiProvider) Close() error {
	// The new unified SDK doesn't require explicit closing
	return nil
}

// convertAttachmentsToGeminiParts converts message attachments to Gemini parts
func (p *GeminiProvider) convertAttachmentsToGeminiParts(attachments []models.Attachment) []*genai.Part {
	var parts []*genai.Part

	for _, attachment := range attachments {
		// Only handle image files for Gemini
		if strings.HasPrefix(attachment.MimeType, "image/") && attachment.Content != "" {
			// Decode base64 content
			imageBytes, err := base64.StdEncoding.DecodeString(attachment.Content)
			if err != nil {
				p.l.WithFields(logrus.Fields{
					"filename": attachment.Filename,
					"error":    err,
				}).Error("Failed to decode base64 content")
				continue
			}

			// Create image part using the correct Gemini method
			imagePart := genai.NewPartFromBytes(imageBytes, attachment.MimeType)
			parts = append(parts, imagePart)
		}
	}

	return parts
}
