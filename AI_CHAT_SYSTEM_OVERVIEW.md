# AI Chat System Overview

A comprehensive AI chat integration for PMM consisting of a standalone Go backend service and React widget UI with Large Language Model (LLM) and Model Context Protocol (MCP) support.

## System Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│                 │    │                 │    │                 │
│   React Widget  │◄──►│  Go Backend     │◄──►│   LLM Service   │
│   (PMM UI)      │    │  (Standalone)   │    │   (OpenAI)      │
│                 │    │                 │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌─────────────────┐
                       │                 │
                       │  MCP Servers    │
                       │  (Multiple)     │
                       │                 │
                       └─────────────────┘
```

## Components

### 1. Go Backend (`aichat-backend/`)

**Standalone application** separate from PMM managed services.

**Key Features:**
- RESTful API with CORS support
- LLM integration with streaming responses
- MCP client for connecting to multiple MCP servers
- In-memory session management
- Health monitoring endpoints

**Configuration:**
- Main config: `config.yaml`
- MCP servers: `mcp-servers.json` (separate JSON file)

**API Endpoints:**
- `POST /v1/chat/send` - Send message and get response
- `GET /v1/chat/stream` - Server-Sent Events streaming
- `GET /v1/chat/history` - Get conversation history
- `DELETE /v1/chat/clear` - Clear conversation history
- `GET /v1/chat/mcp/tools` - List available MCP tools
- `GET /v1/chat/mcp/servers/status` - MCP server status
- `GET /v1/chat/health` - Health check

### 2. React Widget (`ui/src/shared/components/AIChatWidget/`)

**Integrated into PMM UI** as a floating action button widget.

**Key Features:**
- Material-UI design with clean interface
- Real-time streaming responses
- Markdown message rendering
- MCP tools visualization
- Session management
- TypeScript support

**Components:**
- `AIChatWidget.tsx` - Main widget component
- `ChatMessage.tsx` - Message display component
- `MCPToolsDialog.tsx` - Tools viewer dialog
- `api/aichat.ts` - Backend API client

## Configuration Details

### Backend Configuration

**config.yaml** (main configuration):
```yaml
server:
  port: 3001

llm:
  provider: "openai"
  api_key: "${OPENAI_API_KEY}"
  model: "gpt-4o-mini"

mcp:
  servers_file: "mcp-servers.json"
```

**mcp-servers.json** (MCP servers configuration):
```json
{
  "servers": [
    {
      "name": "filesystem",
      "description": "File system operations",
      "command": "npx",
      "args": ["@modelcontextprotocol/server-filesystem", "/path/to/workspace"],
      "timeout": 30,
      "env": {"DEBUG": "mcp*"},
      "enabled": true
    }
  ]
}
```

### Frontend Integration

The React widget is integrated into PMM's main UI layout:

```typescript
// In ui/src/shared/layout/PMM.layout.tsx
import { AIChatWidget } from '../components/AIChatWidget/AIChatWidget';

export const PMMLayout = () => {
  return (
    <div className="pmm-layout">
      {/* ... existing layout ... */}
      <AIChatWidget />
    </div>
  );
};
```

## Data Flow

### Chat Message Flow

1. **User Input**: User types message in React widget
2. **API Call**: Widget sends POST to `/v1/chat/send`
3. **LLM Processing**: Backend processes message with LLM
4. **Tool Execution**: If LLM requests tools, execute via MCP
5. **Response**: Return formatted response to widget
6. **Display**: Widget renders response with markdown

### Streaming Flow

1. **Stream Request**: Widget opens SSE connection to `/v1/chat/stream`
2. **Streaming Response**: Backend streams LLM response chunks
3. **Real-time Display**: Widget updates UI in real-time
4. **Completion**: Stream closes when response complete

## MCP Integration

### Server Configuration

MCP servers are configured in a separate JSON file for better organization:

```json
{
  "servers": [
    {
      "name": "filesystem",
      "description": "File system operations",
      "command": "npx",
      "args": ["@modelcontextprotocol/server-filesystem", "/workspace"],
      "timeout": 30,
      "env": {"DEBUG": "mcp*"},
      "enabled": true
    },
    {
      "name": "clickhouse", 
      "description": "ClickHouse database operations",
      "command": "npx",
      "args": ["@modelcontextprotocol/server-clickhouse"],
      "timeout": 30,
      "env": {
        "CLICKHOUSE_URL": "http://localhost:8123",
        "CLICKHOUSE_USER": "default"
      },
      "enabled": false
    }
  ]
}
```

### Tool Execution

1. **Tool Discovery**: Backend connects to enabled MCP servers
2. **Tool Listing**: Each server exposes available tools
3. **LLM Integration**: Tools are provided to LLM as available functions
4. **Execution**: When LLM calls a tool, backend routes to appropriate MCP server
5. **Result**: Tool result is returned to LLM for final response

## Deployment

### Development

```bash
# Backend
cd aichat-backend
export OPENAI_API_KEY="your-key"
go run cmd/main.go

# Frontend (PMM UI)
cd ui
npm start
```

### Docker Deployment

```bash
# Build backend
docker build -t aichat-backend aichat-backend/

# Run with configuration
docker run -d \
  --name aichat-backend \
  -p 3001:3001 \
  -e OPENAI_API_KEY=your-key \
  -v $(pwd)/aichat-backend/config.yaml:/app/config.yaml \
  -v $(pwd)/aichat-backend/mcp-servers.json:/app/mcp-servers.json \
  aichat-backend
```

## Security Considerations

### Backend Security
- Environment-based API key management
- CORS configuration for frontend access
- Input validation and sanitization
- Rate limiting (to be implemented)

### Frontend Security
- TypeScript for type safety
- Sanitized markdown rendering
- Session isolation
- No sensitive data storage in browser

## Monitoring and Debugging

### Health Monitoring
- `/health` endpoint for service status
- `/api/v1/mcp/servers/status` for MCP server health
- Structured logging throughout application

### Debug Configuration
Enable debugging by adding environment variables to MCP servers:

```json
{
  "env": {
    "DEBUG": "mcp*",
    "LOG_LEVEL": "debug"
  }
}
```

## Integration with PMM

### Standalone Architecture
- **Independent Service**: Backend runs as separate application
- **No PMM Dependencies**: Does not depend on PMM managed services
- **UI Integration**: Widget integrates into existing PMM UI
- **Configuration**: Can be configured for PMM-specific MCP servers

### PMM-Specific MCP Servers
- **ClickHouse**: For query analytics data access
- **Filesystem**: For PMM configuration file access
- **Database**: For PMM metadata queries
- **Monitoring**: For metrics and alerting data

## Future Enhancements

### Planned Features
- **Authentication**: User-based session management
- **Persistent Storage**: Database-backed chat history
- **Rate Limiting**: API rate limiting and quotas
- **Multi-LLM Support**: Support for additional LLM providers
- **Advanced MCP**: More sophisticated MCP server management

### Performance Optimizations
- **Connection Pooling**: Optimize MCP server connections
- **Caching**: Cache frequently used tool results
- **Compression**: Compress large responses
- **Background Processing**: Async tool execution for long-running tasks

This system provides a flexible, scalable foundation for AI chat integration in PMM while maintaining clear separation of concerns and modularity. 