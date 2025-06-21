# LLM Providers

This directory contains all LLM (Large Language Model) provider implementations for the AI Chat Backend. Each provider is organized into its own file for better maintainability and easier extension.

## File Organization

| File | Description | Status |
|------|-------------|--------|
| `interface.go` | Defines the `LLMProvider` interface that all providers must implement | ✅ Complete |
| `openai.go` | OpenAI GPT models implementation | ✅ Complete |
| `gemini.go` | Google Gemini models implementation | ✅ Complete |
| `mock.go` | Mock provider for testing and development | ✅ Complete |
| `claude.go` | Anthropic Claude models implementation | 🚧 Skeleton |
| `ollama.go` | Local Ollama models implementation | 🚧 Skeleton |

## Provider Interface

All providers must implement the `LLMProvider` interface defined in `interface.go`:

```go
type LLMProvider interface {
    GenerateResponse(ctx context.Context, messages []*models.Message, tools []models.MCPTool) (*models.Message, error)
    GenerateStreamResponse(ctx context.Context, messages []*models.Message, tools []models.MCPTool) (<-chan *models.StreamMessage, error)
    Close() error
}
```

## Adding a New Provider

1. **Create a new file**: `internal/providers/myprovider.go`
2. **Implement the interface**: All three methods are required
3. **Register the provider**: Add it to the switch statement in `internal/services/llm.go`
4. **Add validation**: Update `main.go` if special API key validation is needed

## Provider Features

### OpenAI Provider (`openai.go`)
- ✅ Text generation
- ✅ Streaming responses
- ✅ Function calling / tool usage
- ✅ Multiple models (GPT-3.5, GPT-4, etc.)
- ✅ Custom base URL support

### Gemini Provider (`gemini.go`)
- ✅ Text generation
- ✅ Streaming responses
- ✅ Conversation history
- ✅ Multiple models (gemini-1.5-flash, gemini-1.5-pro, etc.)
- ⚠️ Limited tool/function calling support

### Mock Provider (`mock.go`)
- ✅ Simulated responses
- ✅ Streaming simulation
- ✅ Tool count reporting
- ✅ Configurable delays
- 💡 Perfect for testing and development

### Claude Provider (`claude.go`)
- 🚧 Skeleton implementation
- 📝 Ready for Anthropic Claude API integration
- 📝 Structured for Claude's conversation format

### Ollama Provider (`ollama.go`)
- 🚧 Skeleton implementation
- 📝 Ready for local Ollama integration
- 📝 HTTP-based communication with local models

## Configuration

Each provider can be configured via:
- **Provider name**: Set via `AICHAT_LLM_PROVIDER` environment variable
- **Model**: Set via `AICHAT_LLM_MODEL` environment variable
- **API Key**: Provider-specific environment variables
- **Base URL**: For custom endpoints (OpenAI-compatible APIs, etc.)

## Usage Examples

```bash
# Use OpenAI
AICHAT_LLM_PROVIDER=openai OPENAI_API_KEY=sk-... ./aichat-backend

# Use Gemini
AICHAT_LLM_PROVIDER=gemini GEMINI_API_KEY=... ./aichat-backend

# Use Mock for testing
AICHAT_LLM_PROVIDER=mock ./aichat-backend
```

## Best Practices

1. **Error Handling**: Always provide meaningful error messages
2. **Context Support**: Respect context cancellation for timeouts
3. **Resource Cleanup**: Implement proper cleanup in the `Close()` method
4. **Streaming**: Use channels for streaming responses with proper error handling
5. **Configuration**: Support both API keys and custom endpoints where applicable
6. **Testing**: Consider adding unit tests for your provider implementation 