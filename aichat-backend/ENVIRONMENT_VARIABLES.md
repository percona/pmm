# Environment Variables

The AI Chat Backend uses Kong for configuration management, which provides a unified approach to handling environment variables, CLI flags, and configuration files.

## Configuration Precedence

Configuration values are applied in the following order (highest to lowest priority):

1. **CLI flags** (highest priority)
2. **Environment variables** 
3. **Configuration file**
4. **Default values** (lowest priority)

## Available Environment Variables

All environment variables correspond to CLI flags and are automatically handled by Kong.

### Configuration
- `AICHAT_CONFIG` - Path to configuration file (default: `config.yaml`)
- `AICHAT_ENV_ONLY` - Load configuration only from environment variables (default: `false`)

### Server Configuration
- `AICHAT_PORT` - Server port (default: `3001`)

### Logging Configuration
- `AICHAT_LOG_LEVEL` - Log level: `debug`, `info`, `warn`, `error` (default: `info`)
- `AICHAT_LOG_JSON` - Output logs in JSON format (default: `false`)

### Database Configuration
- `AICHAT_DATABASE_URL` - Complete PostgreSQL connection string (default: `postgres://ai_chat_user:ai_chat_secure_password@127.0.0.1:5432/ai_chat?sslmode=disable`)

### LLM Configuration
- `AICHAT_LLM_PROVIDER` - LLM provider: `openai`, `gemini`, `claude`, `mock` (default: `openai`)
- `AICHAT_LLM_MODEL` - LLM model name (default: `gpt-4o-mini`)
- `AICHAT_API_KEY` - API key for the LLM provider
- `AICHAT_LLM_BASE_URL` - Custom base URL for LLM API
- `AICHAT_SYSTEM_PROMPT` - Custom system prompt for the AI assistant

### MCP Configuration
- `AICHAT_MCP_SERVERS_FILE` - Path to MCP servers configuration file (default: `mcp-servers.json`)

## Usage Examples

### Basic Usage with Environment Variables
```bash
export AICHAT_PORT=8080
export AICHAT_LLM_PROVIDER=openai
export AICHAT_API_KEY=your-api-key
export AICHAT_DATABASE_URL="postgres://user:pass@localhost:5432/dbname"
./aichat-backend
```

### Environment Variables Only Mode
```bash
export AICHAT_ENV_ONLY=true
export AICHAT_LLM_PROVIDER=mock
export AICHAT_LOG_LEVEL=debug
./aichat-backend
```

### CLI Flags Override Environment Variables
```bash
export AICHAT_PORT=8080
./aichat-backend --port=9090  # Port 9090 will be used, not 8080
```

### JSON Logging
```bash
export AICHAT_LOG_JSON=true
export AICHAT_LOG_LEVEL=debug
./aichat-backend
```

## Docker Usage

### Using Environment Variables in Docker
```bash
docker run -e AICHAT_PORT=8080 \
           -e AICHAT_LLM_PROVIDER=openai \
           -e AICHAT_API_KEY=your-api-key \
           -e AICHAT_DATABASE_URL="postgres://user:pass@db:5432/ai_chat" \
           aichat-backend
```

### Docker Compose Example
```yaml
version: '3.8'
services:
  aichat-backend:
    image: aichat-backend
    environment:
      AICHAT_PORT: 8080
      AICHAT_LLM_PROVIDER: openai
      AICHAT_API_KEY: ${OPENAI_API_KEY}
      AICHAT_DATABASE_URL: postgres://ai_chat_user:password@postgres:5432/ai_chat
      AICHAT_LOG_LEVEL: info
      AICHAT_LOG_JSON: "true"
    ports:
      - "8080:8080"
```

## Kubernetes Usage

### ConfigMap Example
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: aichat-config
data:
  AICHAT_PORT: "8080"
  AICHAT_LLM_PROVIDER: "openai"
  AICHAT_LOG_LEVEL: "info"
  AICHAT_LOG_JSON: "true"
```

### Secret Example
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: aichat-secrets
type: Opaque
stringData:
  AICHAT_API_KEY: "your-openai-api-key"
  AICHAT_DATABASE_URL: "postgres://user:password@postgres:5432/ai_chat"
```

### Deployment Example
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: aichat-backend
spec:
  replicas: 1
  selector:
    matchLabels:
      app: aichat-backend
  template:
    metadata:
      labels:
        app: aichat-backend
    spec:
      containers:
      - name: aichat-backend
        image: aichat-backend:latest
        envFrom:
        - configMapRef:
            name: aichat-config
        - secretRef:
            name: aichat-secrets
        ports:
        - containerPort: 8080
```

## Migration Notes

### From Legacy Environment Variables

The following legacy environment variables are **no longer supported**:

- `AICHAT_DB_HOST` - Use `AICHAT_DATABASE_URL` instead
- `AICHAT_DB_PORT` - Use `AICHAT_DATABASE_URL` instead  
- `AICHAT_DB_NAME` - Use `AICHAT_DATABASE_URL` instead
- `AICHAT_DB_USERNAME` - Use `AICHAT_DATABASE_URL` instead
- `AICHAT_DB_PASSWORD` - Use `AICHAT_DATABASE_URL` instead
- `AICHAT_DB_SSL_MODE` - Use `AICHAT_DATABASE_URL` instead
- `PORT` - Use `AICHAT_PORT` instead
- `OPENAI_API_KEY` - Use `AICHAT_API_KEY` instead
- `GEMINI_API_KEY` - Use `AICHAT_API_KEY` instead
- `GOOGLE_API_KEY` - Use `AICHAT_API_KEY` instead

### Migration Example

**Before (legacy):**
```bash
export AICHAT_DB_HOST=localhost
export AICHAT_DB_PORT=5432
export AICHAT_DB_NAME=ai_chat
export AICHAT_DB_USERNAME=user
export AICHAT_DB_PASSWORD=pass
export OPENAI_API_KEY=your-key
```

**After (Kong-based):**
```bash
export AICHAT_DATABASE_URL="postgres://user:pass@localhost:5432/ai_chat"
export AICHAT_API_KEY=your-key
```

## Validation

Kong automatically validates environment variables:

- **Enum validation**: `AICHAT_LOG_LEVEL` accepts only `debug`, `info`, `warn`, `error`
- **Type validation**: `AICHAT_PORT` must be a valid integer
- **Boolean validation**: `AICHAT_LOG_JSON` accepts `true`, `false`, `1`, `0`

Invalid values will result in clear error messages with usage information. 