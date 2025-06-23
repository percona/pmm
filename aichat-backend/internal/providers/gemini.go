package providers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/percona/pmm/aichat-backend/internal/config"
	"github.com/percona/pmm/aichat-backend/internal/models"
	"google.golang.org/genai"
)

// GeminiProvider implements the LLMProvider interface for Google Gemini
type GeminiProvider struct {
	client *genai.Client
	config config.LLMConfig
}

// NewGeminiProvider creates a new Gemini provider
func NewGeminiProvider(cfg config.LLMConfig) (*GeminiProvider, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: cfg.APIKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Gemini client: %w", err)
	}

	return &GeminiProvider{
		client: client,
		config: cfg,
	}, nil
}

// convertMCPToolsToGemini converts MCP tools to Gemini function declarations
func (p *GeminiProvider) convertMCPToolsToGemini(tools []models.MCPTool) []*genai.FunctionDeclaration {
	if len(tools) == 0 {
		return nil
	}

	functions := make([]*genai.FunctionDeclaration, 0, len(tools))

	for _, tool := range tools {
		log.Printf("🔧 Gemini: Converting tool %s, schema: %+v", tool.Name, tool.InputSchema)

		// Convert input schema to Gemini format
		parameters := &genai.Schema{
			Type: genai.TypeObject,
		}
		m, err := json.Marshal(tool.InputSchema)
		if err != nil {
			log.Printf("❌ Gemini: Failed to marshal tool input schema: %v", err)
			continue
		}
		log.Printf("🔧 Gemini: Tool input schema: %s", string(m))
		err = json.Unmarshal(m, &parameters)
		if err != nil {
			log.Printf("❌ Gemini: Failed to unmarshal tool input schema: %v", err)
			continue
		}
		log.Printf("🔧 Gemini: Tool input schema: %+v", parameters)

		function := &genai.FunctionDeclaration{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  parameters,
		}

		functions = append(functions, function)
	}

	log.Printf("🔧 Gemini: Converted %d MCP tools to Gemini functions", len(functions))
	return functions
}

// GenerateResponse generates a response using Google Gemini API with function calling
func (p *GeminiProvider) GenerateResponse(ctx context.Context, messages []*models.Message, tools []models.MCPTool) (*models.Message, error) {
	// Convert MCP tools to Gemini functions
	var toolConfig *genai.GenerateContentConfig
	functions := p.convertMCPToolsToGemini(tools)
	if len(functions) > 0 {
		log.Printf("🔧 Gemini: Enabling function calling with %d functions", len(functions))
		toolConfig = &genai.GenerateContentConfig{
			Tools: []*genai.Tool{{
				FunctionDeclarations: functions,
			}},
			Temperature: genai.Ptr(float32(0.7)),
		}
	} else {
		log.Printf("🔧 Gemini: No tools available for function calling")
		toolConfig = &genai.GenerateContentConfig{
			Temperature: genai.Ptr(float32(0.7)),
		}
	}

	// Convert messages to Gemini format
	var contents []*genai.Content

	for _, msg := range messages {
		role := genai.RoleUser
		if msg.Role == "assistant" {
			role = genai.RoleModel
		}

		// Handle tool calls in assistant messages
		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			// Create content with function calls
			parts := []*genai.Part{}

			// Add text content if any
			if msg.Content != "" {
				parts = append(parts, genai.NewPartFromText(msg.Content))
			}

			// Add function calls
			for _, toolCall := range msg.ToolCalls {
				var args map[string]interface{}
				if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
					log.Printf("❌ Gemini: Failed to parse function arguments: %v", err)
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
			// Tool results should be formatted as function responses
			// For now, we'll convert them to text parts but ensure they have proper content
			if msg.Content == "" {
				log.Printf("⚠️  Gemini: Skipping empty tool result message")
				continue
			}

			parts := []*genai.Part{genai.NewPartFromText(fmt.Sprintf("Tool result: %s", msg.Content))}

			content := &genai.Content{
				Role:  genai.RoleUser, // Tool results should be from user perspective
				Parts: parts,
			}
			contents = append(contents, content)
		} else {
			// Handle regular user or system messages
			if msg.Content == "" {
				log.Printf("⚠️  Gemini: Skipping empty message with role: %s", msg.Role)
				continue
			}

			// Create parts starting with text content
			parts := []*genai.Part{genai.NewPartFromText(msg.Content)}

			// Add attachment parts for user messages
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

	// Generate content
	resp, err := p.client.Models.GenerateContent(ctx, p.config.Model, contents, toolConfig)
	if err != nil {
		return nil, fmt.Errorf("Gemini API request failed: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("no response candidates from Gemini")
	}

	candidate := resp.Candidates[0]
	if candidate.Content == nil || len(candidate.Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from Gemini")
	}

	// Parse response content and tool calls
	var content string
	var toolCalls []models.ToolCall

	for i, part := range candidate.Content.Parts {
		if part.Text != "" {
			log.Printf("🔧 Gemini: Found text part %d: %s", i, part.Text)
			content += part.Text
		}
		if part.FunctionCall != nil {
			log.Printf("🔧 Gemini: Found function call part %d: %s with args %+v", i, part.FunctionCall.Name, part.FunctionCall.Args)
			// Convert to tool call
			argsBytes, err := json.Marshal(part.FunctionCall.Args)
			if err != nil {
				log.Printf("❌ Gemini: Failed to marshal function arguments: %v", err)
				continue
			}

			toolCall := models.ToolCall{
				ID:   fmt.Sprintf("call_%d", len(toolCalls)),
				Type: "function",
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{
					Name:      part.FunctionCall.Name,
					Arguments: string(argsBytes),
				},
			}
			toolCalls = append(toolCalls, toolCall)
		}
	}

	if len(toolCalls) > 0 {
		log.Printf("🔧 Gemini: Detected %d tool calls in response", len(toolCalls))
	}

	return &models.Message{
		Role:      "assistant",
		Content:   content,
		ToolCalls: toolCalls,
	}, nil
}

// GenerateStreamResponse generates a streaming response using Google Gemini API
func (p *GeminiProvider) GenerateStreamResponse(ctx context.Context, messages []*models.Message, tools []models.MCPTool) (<-chan *models.StreamMessage, error) {
	log.Printf("🔄 Gemini: Starting streaming response generation")

	// Convert MCP tools to Gemini functions
	var toolConfig *genai.GenerateContentConfig
	functions := p.convertMCPToolsToGemini(tools)
	if len(functions) > 0 {
		log.Printf("🔧 Gemini: Enabling function calling with %d functions", len(functions))
		toolConfig = &genai.GenerateContentConfig{
			Tools: []*genai.Tool{{
				FunctionDeclarations: functions,
			}},
			Temperature: genai.Ptr(float32(0.7)),
		}
	} else {
		log.Printf("🔧 Gemini: No tools available for function calling")
		toolConfig = &genai.GenerateContentConfig{
			Temperature: genai.Ptr(float32(0.7)),
		}
	}

	// Convert messages to Gemini format (same as non-streaming)
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
					log.Printf("❌ Gemini: Failed to parse function arguments: %v", err)
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
			// Tool results should be formatted as function responses
			// For now, we'll convert them to text parts but ensure they have proper content
			if msg.Content == "" {
				log.Printf("⚠️  Gemini: Skipping empty tool result message")
				continue
			}

			parts := []*genai.Part{genai.NewPartFromText(fmt.Sprintf("Tool result: %s", msg.Content))}

			content := &genai.Content{
				Role:  genai.RoleUser, // Tool results should be from user perspective
				Parts: parts,
			}
			contents = append(contents, content)
		} else {
			// Regular user or system messages
			if msg.Content == "" {
				log.Printf("⚠️  Gemini: Skipping empty message with role: %s", msg.Role)
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

	// Create response channel
	responseChan := make(chan *models.StreamMessage, 10)

	// Start goroutine to handle streaming
	go func() {
		defer close(responseChan)

		log.Printf("🔄 Gemini: Starting streaming response processing")
		var messageCount int
		var totalContent string
		var detectedToolCalls []models.ToolCall

		// Use GenerateContentStream for streaming
		streamIter := p.client.Models.GenerateContentStream(ctx, p.config.Model, contents, toolConfig)

		// Process streaming response
		for resp, err := range streamIter {
			if err != nil {
				log.Printf("❌ Gemini: Streaming error: %v", err)
				responseChan <- &models.StreamMessage{
					Type:    "error",
					Content: "",
					Done:    true,
					Error:   fmt.Sprintf("Gemini streaming error: %v", err),
				}
				return
			}

			messageCount++
			log.Printf("📦 Gemini: Processing chunk %d with %d candidates", messageCount, len(resp.Candidates))

			if len(resp.Candidates) == 0 {
				log.Printf("⚠️  Gemini: Chunk %d has no candidates", messageCount)
				continue
			}

			candidate := resp.Candidates[0]
			if candidate.Content == nil || len(candidate.Content.Parts) == 0 {
				log.Printf("⚠️  Gemini: Chunk %d has no content parts", messageCount)
				continue
			}

			// Process parts in this chunk
			for partIndex, part := range candidate.Content.Parts {
				if part.Text != "" {
					log.Printf("📝 Gemini: Chunk %d, part %d - Text content (length: %d): %s",
						messageCount, partIndex+1, len(part.Text),
						func() string {
							if len(part.Text) > 50 {
								return part.Text[:50] + "..."
							}
							return part.Text
						}())

					totalContent += part.Text
					responseChan <- &models.StreamMessage{
						Type:    "message",
						Content: part.Text,
						Done:    false,
					}
				}
				if part.FunctionCall != nil {
					log.Printf("🔧 Gemini: Chunk %d, part %d - Function call: %s with args %+v",
						messageCount, partIndex+1, part.FunctionCall.Name, part.FunctionCall.Args)

					// Convert to tool call
					argsBytes, err := json.Marshal(part.FunctionCall.Args)
					if err != nil {
						log.Printf("❌ Gemini: Failed to marshal function arguments: %v", err)
						continue
					}

					toolCall := models.ToolCall{
						ID:   fmt.Sprintf("call_%d", len(detectedToolCalls)),
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

			// Check for finish reason
			if candidate.FinishReason != "" {
				log.Printf("🏁 Gemini: Chunk %d finish reason: %s", messageCount, candidate.FinishReason)
			}
		}

		// Send final message with tool calls if any were detected
		finalMessage := &models.StreamMessage{
			Type:      "done",
			Content:   totalContent,
			Done:      true,
			ToolCalls: detectedToolCalls,
		}

		if len(detectedToolCalls) > 0 {
			log.Printf("🔧 Gemini: Final message includes %d tool calls", len(detectedToolCalls))
		}

		responseChan <- finalMessage
		log.Printf("✅ Gemini: Successfully completed streaming response")
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
				log.Printf("❌ Gemini: Failed to decode base64 content for %s: %v", attachment.Filename, err)
				continue
			}

			// Create image part using the correct Gemini method
			imagePart := genai.NewPartFromBytes(imageBytes, attachment.MimeType)
			parts = append(parts, imagePart)

			log.Printf("🖼️  Gemini: Added image attachment %s (%s, %d bytes)",
				attachment.Filename, attachment.MimeType, len(imageBytes))
		} else {
			log.Printf("⚠️  Gemini: Skipping non-image attachment %s (%s)",
				attachment.Filename, attachment.MimeType)
		}
	}

	return parts
}
