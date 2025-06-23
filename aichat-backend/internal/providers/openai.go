package providers

import (
	"context"
	"fmt"
	"log"

	openai "github.com/sashabaranov/go-openai"

	"github.com/percona/pmm/aichat-backend/internal/config"
	"github.com/percona/pmm/aichat-backend/internal/models"
)

// OpenAIProvider implements the LLMProvider interface for OpenAI
type OpenAIProvider struct {
	client *openai.Client
	config config.LLMConfig
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(cfg config.LLMConfig) (*OpenAIProvider, error) {
	clientConfig := openai.DefaultConfig(cfg.APIKey)
	if cfg.BaseURL != "" {
		clientConfig.BaseURL = cfg.BaseURL
	}

	client := openai.NewClientWithConfig(clientConfig)

	return &OpenAIProvider{
		client: client,
		config: cfg,
	}, nil
}

// GenerateResponse generates a response using OpenAI API
func (p *OpenAIProvider) GenerateResponse(ctx context.Context, messages []*models.Message, tools []models.MCPTool) (*models.Message, error) {
	// Convert messages to OpenAI format
	openaiMessages := make([]openai.ChatCompletionMessage, 0, len(messages))

	for _, msg := range messages {
		openaiMsg := openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}

		// Handle tool calls if present
		if len(msg.ToolCalls) > 0 {
			toolCalls := make([]openai.ToolCall, 0, len(msg.ToolCalls))
			for _, tc := range msg.ToolCalls {
				toolCalls = append(toolCalls, openai.ToolCall{
					ID:   tc.ID,
					Type: openai.ToolTypeFunction,
					Function: openai.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				})
			}
			openaiMsg.ToolCalls = toolCalls
		}

		openaiMessages = append(openaiMessages, openaiMsg)
	}

	// Convert MCP tools to OpenAI tools
	openaiTools := make([]openai.Tool, 0, len(tools))
	for _, tool := range tools {
		openaiTools = append(openaiTools, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.InputSchema,
			},
		})
	}

	// Create chat completion request
	req := openai.ChatCompletionRequest{
		Model:       p.config.Model,
		Messages:    openaiMessages,
		Tools:       openaiTools,
		Temperature: 0.7,
	}

	// Make the request
	resp, err := p.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate response: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response generated")
	}

	choice := resp.Choices[0]

	// Convert response to our message format
	message := &models.Message{
		ID:      fmt.Sprintf("openai_%s", resp.ID),
		Role:    choice.Message.Role,
		Content: choice.Message.Content,
	}

	// Handle tool calls in response
	if len(choice.Message.ToolCalls) > 0 {
		log.Printf("🔧 OpenAI returned %d tool call(s) in response", len(choice.Message.ToolCalls))
		toolCalls := make([]models.ToolCall, 0, len(choice.Message.ToolCalls))
		for i, tc := range choice.Message.ToolCalls {
			log.Printf("🔧 OpenAI tool call %d: ID=%s, Type=%s, Function=%s, Args=%s",
				i+1, tc.ID, tc.Type, tc.Function.Name, tc.Function.Arguments)

			toolCalls = append(toolCalls, models.ToolCall{
				ID:   tc.ID,
				Type: string(tc.Type),
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}
		message.ToolCalls = toolCalls
	} else {
		log.Printf("📝 OpenAI response (no tool calls): %s", choice.Message.Content)
	}

	return message, nil
}

// GenerateStreamResponse generates a streaming response using OpenAI API
func (p *OpenAIProvider) GenerateStreamResponse(ctx context.Context, messages []*models.Message, tools []models.MCPTool) (<-chan *models.StreamMessage, error) {
	// Convert messages to OpenAI format
	openaiMessages := make([]openai.ChatCompletionMessage, 0, len(messages))

	for _, msg := range messages {
		openaiMessages = append(openaiMessages, openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Convert MCP tools to OpenAI tools
	openaiTools := make([]openai.Tool, 0, len(tools))
	for _, tool := range tools {
		openaiTools = append(openaiTools, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.InputSchema,
			},
		})
	}

	// Create streaming chat completion request
	req := openai.ChatCompletionRequest{
		Model:       p.config.Model,
		Messages:    openaiMessages,
		Tools:       openaiTools,
		Temperature: 0.7,
		Stream:      true,
	}

	stream, err := p.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create stream: %w", err)
	}

	// Create response channel
	responseChan := make(chan *models.StreamMessage, 10)

	// Start goroutine to handle streaming
	go func() {
		defer close(responseChan)
		defer stream.Close()

		log.Printf("🔄 OpenAI: Starting streaming response processing")
		var messageCount int
		var totalContent string

		for {
			response, err := stream.Recv()
			if err != nil {
				if err.Error() == "EOF" {
					// Stream ended normally
					log.Printf("✅ OpenAI: Stream completed normally. Total chunks: %d, content length: %d", messageCount, len(totalContent))
					responseChan <- &models.StreamMessage{
						Type: "done",
					}
					return
				}
				// Error occurred
				log.Printf("❌ OpenAI: Stream error after %d chunks: %v", messageCount, err)
				responseChan <- &models.StreamMessage{
					Type:  "error",
					Error: err.Error(),
				}
				return
			}

			messageCount++
			log.Printf("📦 OpenAI: Received chunk %d", messageCount)

			if len(response.Choices) > 0 {
				choice := response.Choices[0]

				// Log choice details
				if choice.Delta.Content != "" {
					totalContent += choice.Delta.Content
					log.Printf("📝 OpenAI: Content chunk %d (length: %d): %q", messageCount, len(choice.Delta.Content), choice.Delta.Content)
					responseChan <- &models.StreamMessage{
						Type:    "message",
						Content: choice.Delta.Content,
					}
				}

				// Check for tool calls in streaming (OpenAI supports this)
				if len(choice.Delta.ToolCalls) > 0 {
					log.Printf("🔧 OpenAI: Tool calls detected in streaming chunk %d: %d calls", messageCount, len(choice.Delta.ToolCalls))
					for i, tc := range choice.Delta.ToolCalls {
						log.Printf("🔧 OpenAI: Stream tool call %d: ID=%s, Function=%s", i+1, tc.ID, tc.Function.Name)
						responseChan <- &models.StreamMessage{
							Type:    "tool_call",
							Content: fmt.Sprintf("Tool call: %s", tc.Function.Name),
						}
					}
				}

				// Log finish reason if present
				if choice.FinishReason != "" {
					log.Printf("🏁 OpenAI: Stream finished with reason: %s", choice.FinishReason)
				}
			} else {
				log.Printf("⚠️  OpenAI: Chunk %d has no choices", messageCount)
			}
		}
	}()

	return responseChan, nil
}

// Close closes the OpenAI provider (no-op for OpenAI)
func (p *OpenAIProvider) Close() error {
	// OpenAI client doesn't require explicit cleanup
	return nil
}
