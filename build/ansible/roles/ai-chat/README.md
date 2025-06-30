# AI Chat Backend Ansible Role

This role deploys the AI Chat Backend service as a managed service within PMM using manual binary deployment and supervisord management.

## Overview

The AI Chat Backend is a standalone Go service that provides:
- Large Language Model (LLM) integration with multiple providers (OpenAI, Gemini, Claude, Ollama)
- Model Context Protocol (MCP) client support for tool integration
- RESTful API with streaming responses
- Session management and health monitoring
- File upload and processing capabilities

## Role Structure

```
roles/ai-chat/
├── tasks/main.yml              # Main deployment tasks
├── files/
│   ├── ai-chat.ini             # Static supervisord configuration
│   └── mcp-servers.json        # Static MCP servers configuration
├── handlers/main.yml           # Service restart handlers
├── vars/main.yml               # Role variables
└── README.md                   # This file
```

## Dependencies

- `supervisord` role (for service management)
- `nginx` role (for reverse proxy configuration)
- Python 3.13+ with uv (for MCP servers)
- aichat-backend binary (provided via files/)

## Configuration

The service is configured entirely through environment variables set in the supervisord configuration.

### Default Configuration

The service runs with these default settings:
- **Port**: 3001
- **LLM Provider**: OpenAI
- **LLM Model**: gpt-4o-mini
- **Environment Only**: Uses `--env-only` flag for configuration
- **MCP Servers**: Configured via mcp-servers.json

### Required Setup

1. **OpenAI API Key**: Must be provided via environment variable
2. **Binary**: The `aichat-backend` binary must be available in the role files/
3. **uv**: Python package manager for MCP servers

## Deployment

### Including the Role

```yaml
# In your playbook
- hosts: pmm-servers
  roles:
    - role: ai-chat
```

### Manual Deployment

```bash
# Deploy the role
ansible-playbook -i inventory deploy-ai-chat.yml

# Check service status
ansible pmm-servers -m shell -a "supervisorctl status aichat-backend"
```

## Service Management

### Supervisord Integration

The service is managed via supervisord with static configuration in `/etc/supervisord.d/ai-chat.ini`:

```ini
[program:aichat-backend]
command=/usr/sbin/aichat-backend
args=--env-only
directory=/srv/aichat-backend
user=pmm
environment=
    AICHAT_PORT="3001",
    AICHAT_LLM_PROVIDER="openai",
    AICHAT_LLM_MODEL="gpt-4o-mini",
    ...
```

### Service Commands

```bash
# Check status
supervisorctl status aichat-backend

# Start/stop/restart
supervisorctl start aichat-backend
supervisorctl stop aichat-backend
supervisorctl restart aichat-backend

# View logs
tail -f /srv/logs/aichat-backend.log
```

## API Endpoints

The service exposes the following endpoints through nginx reverse proxy:
- `POST /v1/chat/send` - Send chat message
- `POST /v1/chat/send-with-files` - Send chat message with files
- `POST /v1/chat/sessions` - Create new session
- `GET /v1/chat/sessions` - List user sessions
- `GET /v1/chat/sessions/:id/messages` - Get session messages
- `GET /v1/chat/stream` - Server-Sent Events streaming
- `GET /v1/chat/mcp/tools` - List available MCP tools
- `GET /v1/chat/health` - Health check
## File Locations

### Service Files (RPM Package)
- **Binary**: `/usr/sbin/aichat-backend` (RPM installation - standard deployment)
- **Configuration**: `/etc/aichat-backend/config.yaml` (generated from template)
- **MCP Config**: `/etc/aichat-backend/mcp-servers.json`
- **Logs**: `/srv/logs/aichat-backend.log`

### Manual Installation Files
- **Binary**: `/usr/local/bin/aichat-backend` (manual installation/development)
- **Configuration**: Same as RPM package

> **Note**: The standard PMM deployment uses RPM packages with binaries in `/usr/sbin/`. Manual installations for development or custom deployments use `/usr/local/bin/`.

### System Files
- **Supervisord Config**: `/etc/supervisord.d/ai-chat.ini`
- **Nginx Config**: Integrated into `/etc/nginx/conf.d/pmm.conf`

## Configuration Customization

### Environment Variables

To customize the configuration, edit the environment variables in `/etc/supervisord.d/ai-chat.ini`:

```ini
environment=
    AICHAT_PORT="3001",
    AICHAT_LLM_PROVIDER="openai",
    AICHAT_LLM_MODEL="gpt-4",
    AICHAT_API_KEY="your-api-key",
    AICHAT_LLM_TEMPERATURE="0.7",
    AICHAT_MCP_SERVERS_FILE="/srv/aichat-backend/mcp-servers.json",
    GIN_MODE="release"
```

### MCP Servers

To customize MCP servers, edit `/srv/aichat-backend/mcp-servers.json`:

```json
{
  "servers": [
    {
      "name": "filesystem",
      "description": "File system operations",
      "command": "npx",
      "args": ["@modelcontextprotocol/server-filesystem", "/srv"],
      "enabled": true
    }
  ]
}
```

## Security

### Authentication
- All API endpoints use PMM's existing authentication system
- Health check endpoint is publicly accessible (no auth)

### Permissions
- Service runs as `pmm` user
- Configuration files owned by appropriate users
- Logs accessible to `pmm` user

### API Key Management
- API keys should be set in supervisord environment variables
- Avoid hardcoding keys in configuration files
- Use secure deployment practices for secrets

## Troubleshooting

### Service Issues

```bash
# Check if service is running
supervisorctl status aichat-backend

# View service logs
tail -f /srv/logs/aichat-backend.log

# Check environment variables
supervisorctl status aichat-backend | grep environment
```

### Configuration Issues

```bash
# Test backend directly
curl http://localhost:3001/health

# Test through nginx
curl http://localhost/v1/chat/health

# Check nginx configuration
nginx -t
```

### MCP Server Issues

```bash
# Test MCP server manually
npx @modelcontextprotocol/server-filesystem /srv
```

## Monitoring

### Log Monitoring
- Service logs: `/srv/logs/aichat-backend.log`
- Supervisord logs: `/var/log/supervisord/supervisord.log`
- Nginx logs: `/var/log/nginx/access.log`, `/var/log/nginx/error.log`

### Health Checks
- Application health: `GET /v1/chat/health`
- Service status: `supervisorctl status aichat-backend`
- Process monitoring: `ps aux | grep aichat-backend`

## Updates

### Updating the Binary

#### RPM Package Updates (Standard)
```bash
# Update via package manager (recommended)
yum update pmm-aichat-backend

# Or manually replace RPM-installed binary
supervisorctl stop aichat-backend
cp new-aichat-backend /usr/sbin/aichat-backend
chmod +x /usr/sbin/aichat-backend
supervisorctl start aichat-backend
```

#### Manual Installation Updates
```bash
# Stop service
supervisorctl stop aichat-backend

# Replace binary
cp new-aichat-backend /usr/local/bin/aichat-backend
chmod +x /usr/local/bin/aichat-backend

# Start service
supervisorctl start aichat-backend
```

### Configuration Updates

```bash
# Update environment variables in supervisord config
nano /etc/supervisord.d/ai-chat.ini

# Update MCP servers
nano /srv/aichat-backend/mcp-servers.json

# Restart service
supervisorctl restart aichat-backend
```

This simplified approach uses static configuration files and environment variables, eliminating the need for ansible templates while providing a robust deployment solution for the AI Chat Backend within PMM's infrastructure. 