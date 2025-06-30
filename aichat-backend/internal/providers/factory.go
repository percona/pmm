package providers

import (
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/aichat-backend/internal/config"
)

// CreateLLMProvider creates the appropriate LLM provider based on configuration
func CreateLLMProvider(cfg config.LLMConfig) (LLMProvider, error) {
	l := logrus.WithField("component", "llm-provider")

	var provider LLMProvider
	var err error

	switch cfg.Provider {
	case "openai":
		provider, err = NewOpenAIProvider(cfg)
	case "gemini", "google":
		provider, err = NewGeminiProvider(cfg)
	case "mock":
		provider, err = NewMockProvider(cfg)
	case "claude", "anthropic":
		provider, err = NewClaudeProvider(cfg)
	case "ollama":
		provider, err = NewOllamaProvider(cfg)
	default:
		// Default to OpenAI
		provider, err = NewOpenAIProvider(cfg)
	}

	if err != nil {
		l.WithFields(logrus.Fields{
			"provider": cfg.Provider,
			"error":    err,
		}).Error("Failed to initialize LLM provider")
		return nil, err
	}

	l.WithField("provider", cfg.Provider).Info("LLM provider initialized")
	return provider, nil
}
