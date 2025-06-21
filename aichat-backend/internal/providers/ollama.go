package providers

import (
	"context"
	"fmt"

	"github.com/percona/pmm/aichat-backend/internal/config"
	"github.com/percona/pmm/aichat-backend/internal/models"
)

// OllamaProvider implements the LLMProvider interface for local Ollama models
type OllamaProvider struct {
	config  config.LLMConfig
	baseURL string
	// In a real implementation:
	// client *http.Client
}

// NewOllamaProvider creates a new Ollama provider
func NewOllamaProvider(cfg config.LLMConfig) (*OllamaProvider, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434" // Default Ollama URL
	}

	return &OllamaProvider{
		config:  cfg,
		baseURL: baseURL,
	}, nil
}

// GenerateResponse generates a response using Ollama
func (p *OllamaProvider) GenerateResponse(ctx context.Context, messages []*models.Message, tools []models.MCPTool) (*models.Message, error) {
	// In a real implementation, you would:
	// 1. Convert messages to Ollama chat format
	// 2. Make HTTP request to Ollama API
	// 3. Parse response and convert to our format

	return nil, fmt.Errorf("Ollama provider not implemented yet")
}

// GenerateStreamResponse generates a streaming response using Ollama
func (p *OllamaProvider) GenerateStreamResponse(ctx context.Context, messages []*models.Message, tools []models.MCPTool) (<-chan *models.StreamMessage, error) {
	// In a real implementation, you would implement Ollama streaming
	return nil, fmt.Errorf("Ollama streaming not implemented yet")
}

// Close closes the Ollama provider
func (p *OllamaProvider) Close() error {
	// Ollama HTTP client doesn't require explicit cleanup
	return nil
}
