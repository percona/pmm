# AI Chat Backend

A standalone Go backend service for AI chat functionality with Large Language Model (LLM) integration and Model Context Protocol (MCP) client support.

## Features

- **LLM Integration**: Support for OpenAI API with streaming responses
- **MCP Client**: Connect to multiple MCP servers and use their tools
- **RESTful API**: Clean HTTP API for chat operations
- **Session Management**: In-memory chat session handling
- **Health Monitoring**: Health check endpoints for service monitoring

## Project Structure

```
aichat-backend/
├── cmd/
│   └── main.go                 # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go          # Configuration management
│   ├── models/
│   │   └── models.go          # Data models and types
│   ├── services/
│   │   ├── llm.go             # LLM service implementation
│   │   ├── mcp.go             # MCP client service
│   │   └── chat.go            # Chat orchestration service
│   └── handlers/
│       └── handlers.go        # HTTP request handlers
├── config.yaml                # Main configuration file
├── mcp-servers.json           # MCP servers configuration
├── go.mod
├── go.sum
├── Dockerfile
└── README.md
```

## Configuration

### Main Configuration (config.yaml)

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

### MCP Servers Configuration (mcp-servers.json)

Configure your MCP servers in a separate JSON file:

```json
{
  "servers": [
    {
      "name": "filesystem",
      "description": "File system operations",
      "command": "npx",
      "args": ["@modelcontextprotocol/server-filesystem", "/path/to/workspace"],
      "timeout": 30,
      "env": {
        "DEBUG": "mcp*"
      },
      "enabled": true
    },
    {
      "name": "remote-api",
      "description": "Remote API MCP server",
      "url": "https://api.example.com/mcp/sse",
      "timeout": 60,
      "enabled": true
    }
  ]
}
```

## API Endpoints

The backend provides the following RESTful API endpoints:

### Chat Operations
- `POST /v1/chat/send` - Send a chat message and get response
- `GET /v1/chat/history` - Get chat history for a session
- `DELETE /v1/chat/clear` - Clear chat history for a session
- `GET /v1/chat/stream` - Server-Sent Events streaming for real-time responses

### MCP Operations
- `GET /v1/chat/mcp/tools` - List all available MCP tools
- `GET /v1/chat/mcp/servers/status` - Get status of all MCP servers

### Health Check
- `GET /v1/chat/health` - Health check endpoint (no authentication required)

## Quick Start

### 1. Prerequisites

- Go 1.23 or later
- OpenAI API key
- Node.js (for MCP servers)

### 2. Installation

```bash
# Clone the repository (if part of PMM)
git clone https://github.com/percona/pmm.git
cd pmm/aichat-backend

# Or create standalone
mkdir aichat-backend && cd aichat-backend
# Copy the source files
```

### 3. Install Dependencies

```bash
go mod init github.com/percona/pmm/aichat-backend
go mod tidy
```

### 4. Environment Setup

```bash
export OPENAI_API_KEY="your-openai-api-key"
```

### 5. Configure MCP Servers

Create `mcp-servers.json`:

```json
{
  "servers": [
    {
      "name": "filesystem",
      "description": "File system operations",
      "command": "npx",
      "args": ["@modelcontextprotocol/server-filesystem", "/workspace"],
      "timeout": 30,
      "env": {
        "DEBUG": "mcp*"
      },
      "enabled": true
    }
  ]
}
```

### 6. Run the Service

```bash
go run cmd/main.go
```

The service will start on port 3001 (configurable).

## Usage Examples

### Send a Chat Message

```bash
curl -X POST http://localhost:3001/v1/chat/send \
  -H "Content-Type: application/json" \
  -d '{
    "message": "List the files in the current directory",
    "session_id": "user123"
  }'
```

### Get Available Tools

```bash
curl http://localhost:3001/api/v1/mcp/tools
```

### Check Server Status

```bash
curl http://localhost:3001/api/v1/mcp/servers/status
```

## Docker Deployment

### Build Docker Image

```bash
docker build -t aichat-backend .
```

### Run with Docker

```bash
docker run -d \
  --name aichat-backend \
  -p 3001:3001 \
  -e OPENAI_API_KEY=your-api-key \
  -v $(pwd)/config.yaml:/app/config.yaml \
  -v $(pwd)/mcp-servers.json:/app/mcp-servers.json \
  aichat-backend
```

## MCP Server Configuration

The AI Chat Backend supports connecting to multiple MCP (Model Context Protocol) servers simultaneously. MCP servers provide tools and capabilities that can be used during chat conversations.

### Transport Types

The backend supports two transport types for MCP servers:

1. **stdio** - Standard input/output for local MCP servers (when `command` is specified)
2. **sse** - Server-Sent Events for remote HTTP-based MCP servers (when `url` is specified)

### Configuration File

MCP servers are configured in `mcp-servers.json`:

```json
{
  "servers": [
    {
      "name": "filesystem",
      "description": "File system operations",
      "command": "npx",
      "args": ["@modelcontextprotocol/server-filesystem", "/path/to/workspace"],
      "timeout": 30,
      "env": {
        "DEBUG": "mcp*"
      },
      "enabled": true
    },
    {
      "name": "remote-api",
      "description": "Remote API MCP server",
      "url": "https://api.example.com/mcp/sse",
      "timeout": 60,
      "enabled": true
    }
  ]
}
```

### Configuration Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Unique identifier for the server |
| `description` | string | No | Human-readable description |
| `command` | string | stdio only | Command to execute for stdio transport |
| `args` | array | stdio only | Command arguments for stdio transport |
| `url` | string | sse only | Base URL for SSE transport |
| `env` | object | stdio only | Environment variables for stdio transport |
| `timeout` | number | No | Connection timeout in seconds (default: 30) |
| `enabled` | boolean | Yes | Whether the server is enabled |

### Transport Auto-Detection

Transport type is automatically detected based on configuration:
- If `url` is present → **SSE transport**
- If `command` is present → **stdio transport**
- Both fields cannot be specified for the same server

### Example Configurations

#### stdio Transport (Local MCP Server)
```json
{
  "name": "filesystem",
  "description": "Local filesystem operations",
  "command": "npx",
  "args": ["@modelcontextprotocol/server-filesystem", "/workspace"],
  "timeout": 30,
  "enabled": true
}
```

#### SSE Transport (Remote MCP Server)
```json
{
  "name": "remote-tools",
  "description": "Remote tool server",
  "url": "http://localhost:8080/mcp",
  "timeout": 60,
  "enabled": true
}
```

## Development

### Running Tests

```bash
go test ./...
```