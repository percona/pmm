# AI Chat Backend

A sophisticated AI chat backend service for PMM (Percona Monitoring and Management) with support for multiple LLM providers, MCP (Model Context Protocol) server integration, and persistent chat history.

## Features

- **Multiple LLM Providers**: OpenAI, Anthropic, and other providers
- **MCP Server Integration**: Connect to external tools and data sources via MCP protocol
- **PMM Authentication Integration**: Seamless user authentication via PMM's auth server
- **Persistent Chat History**: Database-backed chat sessions with user isolation
- **Session Management**: Create, update, delete, and list chat sessions
- **File Attachments**: Support for uploading and processing files in chat
- **Streaming Responses**: Real-time streaming chat responses via Server-Sent Events
- **Tool Execution with Approval**: Request user approval before executing tools
- **Parallel MCP Connections**: Efficient parallel connection to multiple MCP servers

## Authentication

The AI Chat Backend integrates with PMM's authentication system:

1. **PMM Auth Server**: All `/v1/chat/` endpoints (except `/v1/chat/health`) require PMM authentication with at least `viewer` role
2. **User ID Header**: PMM auth server sets `X-User-ID` header containing the authenticated user's ID
3. **Data Isolation**: Each user's chat sessions and history are completely isolated
4. **Security**: Requests without valid authentication will receive 401 Unauthorized responses

### Authentication Flow

```
User Request → nginx → PMM Auth Server → AI Chat Backend (with X-User-ID header)
```

The auth server validates the user's session and forwards the request with the user ID, ensuring secure access to user-specific chat data.

**Note**: There is no fallback to a default user. All endpoints require valid PMM authentication.

## Configuration

### Environment Variables

```bash
# Server configuration
export AICHAT_PORT="3001"

# LLM configuration
export AICHAT_LLM_PROVIDER="openai"
export AICHAT_LLM_MODEL="gpt-4o-mini"
export AICHAT_API_KEY="your-api-key-here"

# MCP servers configuration
export AICHAT_MCP_SERVERS_FILE="/etc/aichat-backend/mcp-servers.json"

# Database configuration (uses dedicated AI Chat database)
export AICHAT_DATABASE_URL="postgres://ai_chat_user:ai_chat_secure_password@127.0.0.1:5432/ai_chat?sslmode=disable"

# Alternative: Individual database parameters (for backward compatibility)
# export AICHAT_DB_HOST="127.0.0.1"
# export AICHAT_DB_PORT="5432"
# export AICHAT_DB_NAME="ai_chat"
# export AICHAT_DB_USERNAME="ai_chat_user"
# export AICHAT_DB_PASSWORD="ai_chat_secure_password"
# export AICHAT_DB_SSL_MODE="disable"

# CORS configuration
export AICHAT_CORS_ORIGINS="http://localhost:8080,http://localhost:8443"

# Logging
export AICHAT_LOG_LEVEL="info"
```

## API Endpoints

### Authentication

All endpoints (except `/v1/chat/health`) require PMM authentication. The user ID is automatically extracted from the `X-User-ID` header set by PMM's auth server.

### Chat Operations

- `POST /v1/chat/send` - Send a chat message
- `POST /v1/chat/send-with-files` - Send a chat message with file attachments
- `DELETE /v1/chat/clear?session_id=<id>` - Clear chat history for a session
- `GET /v1/chat/stream?session_id=<id>&message=<text>` - Stream chat responses (SSE)

### Session Management

- `POST /v1/chat/sessions` - Create a new chat session
- `GET /v1/chat/sessions` - List user's chat sessions (paginated)
- `GET /v1/chat/sessions/:id` - Get specific session details
- `PUT /v1/chat/sessions/:id` - Update session title
- `DELETE /v1/chat/sessions/:id` - Delete session and all messages
- `GET /v1/chat/sessions/:id/messages` - Get session messages (paginated)

### MCP Operations

- `GET /v1/chat/mcp/tools` - Get available MCP tools
- `GET /v1/chat/mcp/tools?force_refresh=true` - Force refresh MCP tools

### Health Check

- `GET /v1/chat/health` - Health check (no authentication required)

## Usage Examples

### Send a Chat Message

```bash
curl -X POST http://localhost:3001/v1/chat/send \
  -H "Content-Type: application/json" \
  -H "X-User-ID: 123" \
  -d '{
    "message": "Hello, how can you help me with database monitoring?",
    "session_id": "session-123"
  }'
```

### Create a New Session

```bash
curl -X POST http://localhost:3001/v1/chat/sessions \
  -H "Content-Type: application/json" \
  -H "X-User-ID: 123" \
  -d '{
    "title": "Database Performance Discussion"
  }'
```

### List User Sessions

```bash
curl -X GET "http://localhost:3001/v1/chat/sessions?limit=10&offset=0" \
  -H "X-User-ID: 123"
```

### Stream Chat Response

```bash
curl -X GET "http://localhost:3001/v1/chat/stream?session_id=session-123&message=Hello" \
  -H "X-User-ID: 123" \
  -H "Accept: text/event-stream"
```

## Development

### Running Locally

```bash
# Install dependencies
go mod download

# Set environment variables
export AICHAT_LLM_PROVIDER="openai"
export AICHAT_API_KEY="your-openai-api-key"
export AICHAT_DATABASE_URL="postgres://ai_chat_user:password@localhost:5432/ai_chat?sslmode=disable"

# Run the server
go run main.go

# Or build and run
go build -o aichat-backend main.go
./aichat-backend
```

### Testing with Authentication

For local development without PMM's auth server, the application falls back to a default user ID:

```bash
# Test with explicit user ID header
curl -X POST http://localhost:3001/v1/chat/send \
  -H "Content-Type: application/json" \
  -H "X-User-ID: 123" \
  -d '{"message": "Test message", "session_id": "test-session"}'

# Test without header (uses default-user)
curl -X POST http://localhost:3001/v1/chat/send \
  -H "Content-Type: application/json" \
  -d '{"message": "Test message", "session_id": "test-session"}'
```

## Database Schema

The service uses PostgreSQL with the following tables:

- `chat_sessions` - User chat sessions
- `chat_messages` - Individual messages within sessions  
- `chat_attachments` - File attachments for messages

All data is isolated per user ID for security and privacy.

## Integration with PMM

When deployed as part of PMM:

1. **nginx** handles routing and authentication
2. **PMM Auth Server** validates user credentials and sets headers
3. **AI Chat Backend** receives authenticated requests with user context
4. **Database** stores user-specific chat data securely

This ensures that users can only access their own chat sessions and history, maintaining proper data isolation and security within the PMM ecosystem.

## Database Migrations

The AI Chat Backend uses [go-migrate](https://github.com/golang-migrate/migrate) for database schema management with embedded migrations.

### Automatic Migrations

Migrations are embedded into the Go binary and run automatically on application startup:

```bash
# Migrations run automatically when starting the application
./aichat-backend

# Example startup output:
# Running database migrations...
# ✅ Database migration version: 1
```

### Migration Files

Migration files are stored in the `migrations/` directory and embedded at build time:

- `000001_create_chat_tables.up.sql` - Forward migration
- `000001_create_chat_tables.down.sql` - Rollback migration

### Production Benefits

This approach provides several advantages for production deployments:

- **Self-contained**: No need to distribute migration files separately
- **Version consistency**: Migrations always match the binary version
- **Reliability**: No risk of missing migration files
- **Security**: Migration logic is compiled into the binary
- **Simplicity**: No manual migration commands needed