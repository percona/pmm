package services

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

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
	l        *logrus.Entry
}

// NewLLMService creates a new LLM service with the appropriate provider
func NewLLMService(cfg config.LLMConfig) *LLMService {
	l := logrus.WithField("component", "llm-service")

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
		l.WithFields(logrus.Fields{
			"provider": cfg.Provider,
			"error":    err,
		}).Warn("Failed to initialize LLM provider")
		// Return service with nil provider - will fail gracefully at runtime
	} else {
		l.WithField("provider", cfg.Provider).Info("LLM service initialized")
	}

	return &LLMService{
		provider: provider,
		config:   cfg,
		l:        l,
	}
}

// GenerateResponse generates a response from the configured LLM provider
func (s *LLMService) GenerateResponse(ctx context.Context, messages []*models.Message, tools []models.MCPTool) (*models.Message, error) {
	s.l.WithFields(logrus.Fields{
		"provider":      s.config.Provider,
		"message_count": len(messages),
		"tool_count":    len(tools),
	}).Debug("Generating response using LLM provider")

	if s.provider == nil {
		return nil, fmt.Errorf("LLM provider not initialized")
	}

	response, err := s.provider.GenerateResponse(ctx, messages, tools)
	if err != nil {
		s.l.WithError(err).Error("Response generation failed")
		return nil, err
	}

	s.l.WithFields(logrus.Fields{
		"content_length": len(response.Content),
		"tool_calls":     len(response.ToolCalls),
	}).Debug("Response generated successfully")
	return response, nil
}

// GenerateStreamResponse generates a streaming response from the configured LLM provider
func (s *LLMService) GenerateStreamResponse(ctx context.Context, messages []*models.Message, tools []models.MCPTool) (<-chan *models.StreamMessage, error) {
	s.l.WithFields(logrus.Fields{
		"provider":      s.config.Provider,
		"message_count": len(messages),
		"tool_count":    len(tools),
	}).Debug("Starting streaming response using LLM provider")

	if s.provider == nil {
		return nil, fmt.Errorf("LLM provider not initialized")
	}

	streamChan, err := s.provider.GenerateStreamResponse(ctx, messages, tools)
	if err != nil {
		s.l.WithError(err).Error("Streaming response failed to start")
		return nil, err
	}

	s.l.Debug("Streaming response started successfully")
	return streamChan, nil
}

// Close closes the LLM service and cleans up resources
func (s *LLMService) Close() error {
	if s.provider != nil {
		return s.provider.Close()
	}
	return nil
}
