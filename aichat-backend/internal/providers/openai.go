package providers

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	openai "github.com/sashabaranov/go-openai"

	"github.com/percona/pmm/aichat-backend/internal/config"
	"github.com/percona/pmm/aichat-backend/internal/models"
)

// OpenAIProvider implements the LLMProvider interface for OpenAI
type OpenAIProvider struct {
	client *openai.Client
	config config.LLMConfig
	l      *logrus.Entry
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(cfg config.LLMConfig) (*OpenAIProvider, error) {
	clientConfig := openai.DefaultConfig(cfg.APIKey)
	if cfg.BaseURL != "" {
		clientConfig.BaseURL = cfg.BaseURL
	}

	client := openai.NewClientWithConfig(clientConfig)

	l := logrus.WithField("component", "openai-provider")
	l.WithFields(logrus.Fields{
		"model":    cfg.Model,
		"base_url": cfg.BaseURL,
	}).Info("Initializing OpenAI provider")

	return &OpenAIProvider{
		client: client,
		config: cfg,
		l:      l,
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
		p.l.WithField("tool_call_count", len(choice.Message.ToolCalls)).Debug("OpenAI returned tool calls in response")

		for i, tc := range choice.Message.ToolCalls {
			p.l.WithFields(logrus.Fields{
				"call_index": i + 1,
				"call_id":    tc.ID,
				"type":       tc.Type,
				"function":   tc.Function.Name,
				"arguments":  tc.Function.Arguments,
			}).Debug("OpenAI tool call details")

			message.ToolCalls = append(message.ToolCalls, models.ToolCall{
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
	} else {
		p.l.WithField("content", choice.Message.Content).Debug("OpenAI response (no tool calls)")
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

		p.l.Debug("Starting streaming response processing")
		var messageCount int
		var totalContent string

		for {
			response, err := stream.Recv()
			if err != nil {
				if err.Error() == "EOF" {
					// Stream ended normally
					p.l.WithFields(logrus.Fields{
						"total_chunks":   messageCount,
						"content_length": len(totalContent),
					}).Debug("Stream completed normally")
					responseChan <- &models.StreamMessage{
						Type: "done",
					}
					return
				}
				// Error occurred
				p.l.WithFields(logrus.Fields{
					"chunks_processed": messageCount,
					"error":            err,
				}).Error("Stream error after processing chunks")
				responseChan <- &models.StreamMessage{
					Type:  "error",
					Error: err.Error(),
				}
				return
			}

			messageCount++
			p.l.WithField("chunk_number", messageCount).Debug("Received chunk")

			if len(response.Choices) > 0 {
				choice := response.Choices[0]

				// Log choice details
				if choice.Delta.Content != "" {
					totalContent += choice.Delta.Content
					p.l.WithFields(logrus.Fields{
						"chunk_number":   messageCount,
						"content_length": len(choice.Delta.Content),
						"content":        choice.Delta.Content,
					}).Debug("Content chunk received")
					responseChan <- &models.StreamMessage{
						Type:    "message",
						Content: choice.Delta.Content,
					}
				}

				// Check for tool calls in streaming (OpenAI supports this)
				if len(choice.Delta.ToolCalls) > 0 {
					p.l.WithFields(logrus.Fields{
						"chunk_number":    messageCount,
						"tool_call_count": len(choice.Delta.ToolCalls),
					}).Debug("Tool calls detected in streaming chunk")

					for i, tc := range choice.Delta.ToolCalls {
						p.l.WithFields(logrus.Fields{
							"call_index": i + 1,
							"call_id":    tc.ID,
							"function":   tc.Function.Name,
						}).Debug("Stream tool call details")
						responseChan <- &models.StreamMessage{
							Type:    "tool_call",
							Content: fmt.Sprintf("Tool call: %s", tc.Function.Name),
						}
					}
				}

				// Log finish reason if present
				if choice.FinishReason != "" {
					p.l.WithField("finish_reason", choice.FinishReason).Debug("Stream finished")
				}
			} else {
				p.l.WithField("chunk_number", messageCount).Debug("Chunk has no choices")
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
