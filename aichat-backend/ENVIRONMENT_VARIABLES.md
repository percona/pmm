# AI Chat Backend Environment Variables

The AI Chat Backend supports comprehensive configuration through environment variables, making it suitable for containerized and cloud deployments.

## Usage

You can run the backend in two modes:

### 1. File + Environment (Default)
```bash
# Load config from file with environment variable overrides
./aichat-backend -config config.yaml
```

### 2. Environment Only
```bash
# Load configuration entirely from environment variables
./aichat-backend --env-only
```

## Server Configuration

| Environment Variable | Description | Default | Example |
|---------------------|-------------|---------|---------|
| `AICHAT_PORT` | Server port | `3001` | `8080` |
| `PORT` | Alternative port variable | `3001` | `8080` |
| `AICHAT_CORS_ORIGINS` | Allowed CORS origins (comma-separated) | `http://localhost:8080,http://localhost:8443` | `https://pmm.example.com` |
| `GIN_MODE` | Gin framework mode | `debug` | `release` |

## LLM Configuration

| Environment Variable | Description | Default | Example |
|---------------------|-------------|---------|---------|
| `AICHAT_LLM_PROVIDER` | LLM provider | `openai` | `openai` |
| `AICHAT_API_KEY` | API key for LLM service | - | `sk-...` |
| `OPENAI_API_KEY` | OpenAI API key (alternative) | - | `sk-...` |
| `AICHAT_LLM_MODEL` | LLM model to use | `gpt-4o-mini` | `gpt-4` |
| `AICHAT_LLM_BASE_URL` | Custom LLM API base URL | - | `https://api.example.com/v1` |

## LLM Options

| Environment Variable | Description | Default | Example |
|---------------------|-------------|---------|---------|
| `AICHAT_LLM_TEMPERATURE` | Response creativity (0.0-2.0) | - | `0.7` |
| `AICHAT_LLM_MAX_TOKENS` | Maximum response tokens | - | `2000` |
| `AICHAT_LLM_TOP_P` | Top-p sampling parameter | - | `0.9` |

## MCP Configuration

| Environment Variable | Description | Default | Example |
|---------------------|-------------|---------|---------|
| `AICHAT_MCP_SERVERS_FILE` | Path to MCP servers JSON file | `mcp-servers.json` | `/srv/aichat-backend/mcp-servers.json` |

## Application Configuration

| Environment Variable | Description | Default | Example |
|---------------------|-------------|---------|---------|
| `AICHAT_LOG_LEVEL` | Logging level | `info` | `debug` |
| `AICHAT_VERSION` | Application version | `dev` | `pmm-3.x` |

## PMM Integration Variables

When deployed in PMM via ansible, these variables are automatically set:

```bash
# Set via ansible templates
AICHAT_PORT=3001
AICHAT_LLM_PROVIDER=openai
AICHAT_LLM_MODEL=gpt-4o-mini
AICHAT_API_KEY=<from_ansible_vault>
AICHAT_MCP_SERVERS_FILE=/srv/aichat-backend/mcp-servers.json
AICHAT_LOG_LEVEL=info
AICHAT_CORS_ORIGINS=http://localhost:8080,http://localhost:8443
AICHAT_VERSION=pmm-3.x
GIN_MODE=release
```

## Example Configurations

### Development Environment
```bash
export AICHAT_PORT=3001
export OPENAI_API_KEY=sk-your-api-key
export AICHAT_LLM_MODEL=gpt-4o-mini
export AICHAT_LLM_TEMPERATURE=0.7
export AICHAT_LOG_LEVEL=debug
export GIN_MODE=debug

./aichat-backend --env-only
```

### Production Environment
```bash
export AICHAT_PORT=3001
export OPENAI_API_KEY=${OPENAI_API_KEY}
export AICHAT_LLM_MODEL=gpt-4
export AICHAT_LLM_TEMPERATURE=0.3
export AICHAT_LLM_MAX_TOKENS=4000
export AICHAT_MCP_SERVERS_FILE=/srv/aichat-backend/mcp-servers.json
export AICHAT_LOG_LEVEL=info
export AICHAT_CORS_ORIGINS=https://pmm.company.com
export GIN_MODE=release

./aichat-backend --env-only
```

### Docker Environment
```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o aichat-backend main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates nodejs npm
WORKDIR /app
COPY --from=builder /app/aichat-backend .

ENV AICHAT_PORT=3001
ENV AICHAT_LLM_PROVIDER=openai
ENV AICHAT_LLM_MODEL=gpt-4o-mini
ENV GIN_MODE=release

CMD ["./aichat-backend", "--env-only"]
```

### Kubernetes Deployment
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: aichat-backend
spec:
  template:
    spec:
      containers:
      - name: aichat-backend
        image: aichat-backend:latest
        env:
        - name: AICHAT_PORT
          value: "3001"
        - name: OPENAI_API_KEY
          valueFrom:
            secretKeyRef:
              name: aichat-secrets
              key: openai-api-key
        - name: AICHAT_LLM_MODEL
          value: "gpt-4o-mini"
        - name: AICHAT_MCP_SERVERS_FILE
          value: "/config/mcp-servers.json"
        - name: GIN_MODE
          value: "release"
        args: ["--env-only"]
```

## Configuration Priority

The configuration is loaded in the following priority order (highest to lowest):

1. **Environment Variables** (highest priority)
2. **Configuration File** (YAML)
3. **Default Values** (lowest priority)

This means environment variables will always override file-based configuration.

## Validation

The application validates the following required settings:

- **API Key**: Required when using OpenAI provider (`OPENAI_API_KEY` or `AICHAT_API_KEY`)
- **Port**: Must be between 1 and 65535
- **MCP Servers File**: Must be readable JSON file (if specified)

## Security Considerations

### Environment Variables Security

- **Never commit API keys** to version control
- Use **secrets management** systems (Kubernetes secrets, Docker secrets, etc.)
- Restrict **environment variable visibility** in production systems
- Use **least privilege** access for service accounts

### Best Practices

```bash
# ✅ Good: Use secrets management
export OPENAI_API_KEY=$(vault kv get -field=api_key secret/openai)

# ✅ Good: Load from file with restricted permissions
export OPENAI_API_KEY=$(cat /etc/secrets/openai-api-key)

# ❌ Bad: Hardcoded in scripts
export OPENAI_API_KEY=sk-hardcoded-key-here

# ❌ Bad: Visible in process list
./aichat-backend --api-key sk-visible-in-ps
```

## Troubleshooting

### Configuration Issues

```bash
# Test configuration loading
./aichat-backend --env-only --version

# Check if environment variables are set
env | grep AICHAT_

# Validate configuration
./aichat-backend --env-only &
curl http://localhost:3001/health
```

### Common Problems

1. **Missing API Key**
   ```
   Error: OpenAI API key is required (set OPENAI_API_KEY or AICHAT_API_KEY environment variable)
   ```
   **Solution**: Set the API key environment variable

2. **Port in Use**
   ```
   Error: Failed to start server: listen tcp :3001: bind: address already in use
   ```
   **Solution**: Change `AICHAT_PORT` or stop conflicting service

3. **Invalid MCP Servers File**
   ```
   Warning: Failed to initialize MCP service: no such file or directory
   ```
   **Solution**: Ensure `AICHAT_MCP_SERVERS_FILE` points to valid JSON file

This environment variable approach provides maximum flexibility for deployment across different environments while maintaining security best practices. 