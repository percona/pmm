package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/percona/pmm/aichat-backend/internal/config"
	"github.com/percona/pmm/aichat-backend/internal/models"
)

// MockProvider implements the LLMProvider interface for testing and demonstration
type MockProvider struct {
	config config.LLMConfig
}

// NewMockProvider creates a new mock provider for testing
func NewMockProvider(cfg config.LLMConfig) (*MockProvider, error) {
	return &MockProvider{
		config: cfg,
	}, nil
}

// GenerateStreamResponse generates a mock streaming response
func (p *MockProvider) GenerateStreamResponse(ctx context.Context, messages []*models.Message, tools []models.MCPTool) (<-chan *models.StreamMessage, error) {
	// Find the last user message
	var lastUserMessage string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			lastUserMessage = messages[i].Content
			break
		}
	}

	// Create response channel
	responseChan := make(chan *models.StreamMessage, 10)

	// Start goroutine to simulate streaming
	go func() {
		defer close(responseChan)

		mockWords := []string{"Mock", "streaming", "response", "to:", fmt.Sprintf("'%s'.", lastUserMessage), "Available", "tools:", fmt.Sprintf("%d", len(tools))}

		for _, word := range mockWords {
			select {
			case <-ctx.Done():
				responseChan <- &models.StreamMessage{
					Type:  "error",
					Error: ctx.Err().Error(),
				}
				return
			case <-time.After(200 * time.Millisecond):
				responseChan <- &models.StreamMessage{
					Type:    "message",
					Content: word + " ",
				}
			}
		}

		// Send completion signal
		responseChan <- &models.StreamMessage{
			Type: "done",
		}
	}()

	return responseChan, nil
}

// Close closes the mock provider (no-op)
func (p *MockProvider) Close() error {
	// Mock provider doesn't require cleanup
	return nil
}
