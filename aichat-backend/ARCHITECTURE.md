# AI Chat Backend Architecture

## Overview

The AI Chat Backend is built using a clean, interface-based architecture that follows the Strategy pattern for LLM provider implementations. This design makes it easy to add new LLM providers and maintain existing ones.

## Core Components

### 1. LLM Service Layer

#### Directory Structure
```
internal/
├── providers/
│   ├── interface.go    # LLMProvider interface definition
│   ├── openai.go      # OpenAI provider implementation
│   ├── gemini.go      # Google Gemini provider implementation
│   ├── mock.go        # Mock provider for testing
│   ├── claude.go      # Claude provider skeleton
│   └── ollama.go      # Ollama provider skeleton
└── services/
    └── llm.go         # LLM service (strategy context)
```

#### LLMProvider Interface (`internal/providers/interface.go`)
```go
type LLMProvider interface {
    GenerateResponse(ctx context.Context, messages []*models.Message, tools []models.MCPTool) (*models.Message, error)
    GenerateStreamResponse(ctx context.Context, messages []*models.Message, tools []models.MCPTool) (<-chan *models.StreamMessage, error)
    Close() error
}
```

All LLM providers must implement this interface, ensuring consistent behavior across different providers.

#### LLMService (Strategy Context) (`internal/services/llm.go`)
The `LLMService` acts as a strategy context that:
- Selects the appropriate provider based on configuration
- Delegates calls to the selected provider
- Handles provider initialization and cleanup

### 2. Provider Implementations

#### Currently Implemented Providers

1. **OpenAIProvider** (`internal/providers/openai.go`)
   - Full implementation for OpenAI GPT models
   - Supports streaming and function calling
   - Uses `github.com/sashabaranov/go-openai`

2. **GeminiProvider** (`internal/providers/gemini.go`)
   - Full implementation for Google Gemini models
   - Supports streaming and conversation history
   - Uses `github.com/google/generative-ai-go/genai`

3. **MockProvider** (`internal/providers/mock.go`)
   - Testing/demonstration provider
   - Simulates API calls with mock responses
   - Useful for development and testing

#### Partially Implemented (Examples)

4. **ClaudeProvider** (`internal/providers/claude.go`)
   - Skeleton implementation for Anthropic Claude
   - Shows structure for adding Claude support

5. **OllamaProvider** (`internal/providers/ollama.go`)
   - Skeleton implementation for local Ollama models
   - Shows structure for local model integration

## Adding a New LLM Provider

### Step 1: Create the Provider File

Create a new file `internal/providers/myprovider.go` with your provider implementation:

```go
package providers

import (
    "context"
    "fmt"

    "github.com/percona/pmm/aichat-backend/internal/config"
    "github.com/percona/pmm/aichat-backend/internal/models"
)

type MyProvider struct {
    client *myapi.Client
    config config.LLMConfig
}

func NewMyProvider(cfg config.LLMConfig) (*MyProvider, error) {
    client := myapi.NewClient(cfg.APIKey)
    return &MyProvider{
        client: client,
        config: cfg,
    }, nil
}

func (p *MyProvider) GenerateResponse(ctx context.Context, messages []*models.Message, tools []models.MCPTool) (*models.Message, error) {
    // 1. Convert messages to provider format
    // 2. Convert tools to provider format
    // 3. Make API call
    // 4. Convert response back to our format
}

func (p *MyProvider) GenerateStreamResponse(ctx context.Context, messages []*models.Message, tools []models.MCPTool) (<-chan *models.StreamMessage, error) {
    // Implement streaming response
}

func (p *MyProvider) Close() error {
    // Cleanup resources
}
```

### Step 2: Register the Provider

Add the provider to the switch statement in `internal/services/llm.go`:

```go
switch cfg.Provider {
case "myprovider":
    provider, err = providers.NewMyProvider(cfg)
// ... existing cases
}
```

### Step 3: Update Configuration

Add validation for the new provider in `main.go` if needed:

```go
case "myprovider":
    if cfg.LLM.APIKey == "" {
        return fmt.Errorf("MyProvider API key is required")
    }
```

## Architecture Benefits

### 1. Separation of Concerns
- Each provider handles its own API specifics
- Common interface ensures consistent behavior
- Easy to test individual providers

### 2. Easy Extension
- New providers can be added without modifying existing code
- Clear contract defined by the interface
- Minimal changes required to support new providers

### 3. Maintainability
- Provider-specific code is isolated
- Bug fixes only affect specific providers
- Easier to update dependencies

### 4. Testability
- Mock provider available for testing
- Easy to unit test individual providers
- Interface allows for easy mocking

## Configuration

Each provider can be configured via:

1. **Configuration File** (`config.yaml`)
2. **Environment Variables**
3. **Command Line Flags**

Example configurations:

```yaml
llm:
  provider: "openai"          # openai, gemini, mock, claude, ollama
  model: "gpt-4"             # Provider-specific model name
  api_key: "${OPENAI_API_KEY}" # API key (can use env vars)
  base_url: ""               # Optional custom base URL
```

## Error Handling

The architecture includes comprehensive error handling:

- Provider initialization errors are logged but don't crash the application
- Runtime errors are propagated with context
- Graceful degradation when providers are unavailable
- Proper resource cleanup on shutdown

## Performance Considerations

- Providers are initialized once at startup
- Connection pooling is handled by individual provider implementations
- Streaming responses use channels for efficient memory usage
- Context cancellation is supported for request timeouts

## Future Enhancements

Possible future improvements:

1. **Provider Health Checks**: Regular health monitoring of providers
2. **Fallback Providers**: Automatic fallback when primary provider fails
3. **Load Balancing**: Distribute requests across multiple provider instances
4. **Provider Metrics**: Detailed metrics collection per provider
5. **Dynamic Provider Registration**: Runtime provider loading 