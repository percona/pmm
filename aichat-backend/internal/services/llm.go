package services

import (
	"context"
	"fmt"
	"log"

	"github.com/percona/pmm/aichat-backend/internal/config"
	"github.com/percona/pmm/aichat-backend/internal/models"
	"github.com/percona/pmm/aichat-backend/internal/providers"
)

// LLMProvider defines the interface that all LLM providers must implement
type LLMProvider interface {
	GenerateResponse(ctx context.Context, messages []*models.Message, tools []models.MCPTool) (*models.Message, error)
	GenerateStreamResponse(ctx context.Context, messages []*models.Message, tools []models.MCPTool) (<-chan *models.StreamMessage, error)
	Close() error
}

// LLMService handles communication with LLM providers using the strategy pattern
type LLMService struct {
	provider providers.LLMProvider
	config   config.LLMConfig
}

// NewLLMService creates a new LLM service with the appropriate provider
func NewLLMService(cfg config.LLMConfig) *LLMService {
	var provider providers.LLMProvider
	var err error

	switch cfg.Provider {
	case "openai":
		provider, err = providers.NewOpenAIProvider(cfg)
	case "gemini", "google":
		provider, err = providers.NewGeminiProvider(cfg)
	case "mock":
		provider, err = providers.NewMockProvider(cfg)
	case "claude", "anthropic":
		provider, err = providers.NewClaudeProvider(cfg)
	case "ollama":
		provider, err = providers.NewOllamaProvider(cfg)
	default:
		// Default to OpenAI
		provider, err = providers.NewOpenAIProvider(cfg)
	}

	if err != nil {
		fmt.Printf("Warning: Failed to initialize %s provider: %v\n", cfg.Provider, err)
		// Return service with nil provider - will fail gracefully at runtime
	}

	return &LLMService{
		provider: provider,
		config:   cfg,
	}
}

// GenerateResponse generates a response from the configured LLM provider
func (s *LLMService) GenerateResponse(ctx context.Context, messages []*models.Message, tools []models.MCPTool) (*models.Message, error) {
	log.Printf("🤖 LLM: Generating response using provider: %s, messages: %d, tools: %d", s.config.Provider, len(messages), len(tools))

	if s.provider == nil {
		return nil, fmt.Errorf("LLM provider not initialized")
	}

	response, err := s.provider.GenerateResponse(ctx, messages, tools)
	if err != nil {
		log.Printf("❌ LLM: Response generation failed: %v", err)
		return nil, err
	}

	log.Printf("✅ LLM: Response generated successfully, content length: %d, tool calls: %d", len(response.Content), len(response.ToolCalls))
	return response, nil
}

// GenerateStreamResponse generates a streaming response from the configured LLM provider
func (s *LLMService) GenerateStreamResponse(ctx context.Context, messages []*models.Message, tools []models.MCPTool) (<-chan *models.StreamMessage, error) {
	log.Printf("🌊 LLM: Starting streaming response using provider: %s, messages: %d, tools: %d", s.config.Provider, len(messages), len(tools))

	if s.provider == nil {
		return nil, fmt.Errorf("LLM provider not initialized")
	}

	streamChan, err := s.provider.GenerateStreamResponse(ctx, messages, tools)
	if err != nil {
		log.Printf("❌ LLM: Streaming response failed to start: %v", err)
		return nil, err
	}

	log.Printf("✅ LLM: Streaming response started successfully")
	return streamChan, nil
}

// Close closes the LLM service and cleans up resources
func (s *LLMService) Close() error {
	if s.provider != nil {
		return s.provider.Close()
	}
	return nil
}
