package providers

import (
	"context"

	"github.com/percona/pmm/aichat-backend/internal/models"
)

// LLMProvider defines the interface that all LLM providers must implement
type LLMProvider interface {
	GenerateResponse(ctx context.Context, messages []*models.Message, tools []models.MCPTool) (*models.Message, error)
	GenerateStreamResponse(ctx context.Context, messages []*models.Message, tools []models.MCPTool) (<-chan *models.StreamMessage, error)
	Close() error
}
