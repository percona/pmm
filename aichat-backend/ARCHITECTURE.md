# AI Chat Backend Architecture

This document describes the architecture and design decisions for the AI Chat Backend service.

## Overview

The AI Chat Backend is a Go-based service that provides conversational AI capabilities with tool execution support through the Model Context Protocol (MCP). The service integrates with various Language Learning Models (LLMs) and can execute tools in parallel for improved performance.

## Core Components

### Chat Service (`internal/services/chat.go`)

The main orchestrator that handles:
- User message processing
- LLM integration and streaming responses
- **Parallel tool execution** for improved performance
- Session management
- Tool approval workflows

#### Parallel Tool Execution

The service executes multiple tools concurrently when the LLM requests several tools simultaneously. This provides significant performance improvements:

- **Sequential Execution**: Tools executed one after another (total time = sum of all tool execution times)
- **Parallel Execution**: Tools executed simultaneously (total time ≈ longest individual tool execution time)

**Example Performance Improvement:**
```
3 tools, each taking 200ms:
- Sequential: ~600ms total
- Parallel: ~200ms total (3x faster)
```

**Implementation Details:**
- Uses Go goroutines and channels for concurrent execution
- Maintains result ordering to preserve LLM context
- Thread-safe with proper synchronization
- Handles errors gracefully for individual tool failures

**Benefits:**
- Faster query analysis and recommendations
- Better user experience with reduced wait times
- Efficient resource utilization
- Scalable tool execution

### MCP Service (`internal/services/mcp.go`)

Manages connections to MCP servers and tool execution:
- Multiple MCP server connections
- Tool discovery and registration
- Thread-safe tool execution
- Connection health monitoring

### Database Service (`internal/services/database.go`)

Handles data persistence:
- Chat history storage
- Session management
- Tool approval tracking
- Message persistence

### LLM Providers (`internal/providers/`)

Abstraction layer for different LLM services:
- OpenAI integration
- Streaming response support
- Tool calling capabilities
- Error handling and retries

## Data Flow

1. **User Message** → Chat Service
2. **Session Management** → Database Service
3. **LLM Processing** → LLM Provider
4. **Tool Requests** → MCP Service
5. **Parallel Tool Execution** → Multiple MCP tools simultaneously
6. **Results Aggregation** → Chat Service
7. **Response Generation** → LLM Provider
8. **Response Streaming** → User Interface

## Tool Execution Flow

```
LLM Request: [tool1, tool2, tool3]
    ↓
Parallel Execution:
    tool1() ← goroutine 1
    tool2() ← goroutine 2  
    tool3() ← goroutine 3
    ↓
Result Collection (maintains order):
    [result1, result2, result3]
    ↓
Continue LLM Processing
```

## Configuration

The service is configured through:
- Environment variables
- YAML configuration files
- MCP server definitions
- Database connection settings

## Security Considerations

- Tool approval workflows
- Session isolation
- Input validation
- Error sanitization
- Connection security

## Performance Optimizations

- **Parallel tool execution** (primary optimization)
- Connection pooling
- Streaming responses
- Efficient session management
- Concurrent MCP server connections

## Monitoring and Observability

- Structured logging
- Performance metrics
- Tool execution timing
- Error tracking
- Health checks

## Directory Structure
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

## LLMProvider Interface (`internal/providers/interface.go`)
```go
type LLMProvider interface {
    GenerateStreamResponse(ctx context.Context, messages []*models.Message, tools []models.MCPTool) (<-chan *models.StreamMessage, error)
    Close() error
}
```

All LLM providers must implement this interface, ensuring consistent behavior across different providers.

## LLMService (Strategy Context) (`internal/services/llm.go`)
The `LLMService` acts as a strategy context that:
- Selects the appropriate provider based on configuration
- Delegates calls to the selected provider
- Handles provider initialization and cleanup

## Currently Implemented Providers

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
  api_key: "your-api-key-here" # API key (can use env vars)
  base_url: ""               # Optional custom base URL
```

## Error Handling

The architecture includes comprehensive error handling:
llm:
  provider: "openai"          # openai, gemini, mock, claude, ollama
  model: "gpt-4"             # Provider-specific model name
  api_key: "your-api-key-here" # API key (use env vars in production)
  base_url: ""               # Optional custom base URL
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