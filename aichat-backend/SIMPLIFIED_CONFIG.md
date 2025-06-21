# Simplified MCP Configuration

The MCP client configuration has been simplified to automatically detect transport type based on the presence of `url` or `command` fields.

## Auto-Detection Rules

- **SSE Transport**: When `url` field is present
- **stdio Transport**: When `command` field is present
- **Error**: When neither or both fields are present

## Configuration Examples

### stdio Transport (Local Servers)

```json
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
```

### SSE Transport (Remote Servers)

```json
{
  "name": "remote-api", 
  "description": "Remote API server",
  "url": "https://api.example.com/mcp/sse",
  "timeout": 60,
  "enabled": true
}
```

## Complete Configuration File

```json
{
  "servers": [
    {
      "name": "filesystem",
      "description": "Local file operations",
      "command": "npx", 
      "args": ["@modelcontextprotocol/server-filesystem", "/workspace"],
      "timeout": 30,
      "enabled": true
    },
    {
      "name": "remote-tools",
      "description": "Remote tool server",
      "url": "http://localhost:8080/mcp",
      "timeout": 60,
      "enabled": true
    },
    {
      "name": "database",
      "description": "Database operations", 
      "command": "python",
      "args": ["-m", "mcp_server_database", "--connection-string", "postgresql://user:pass@localhost/db"],
      "env": {
        "PGPASSWORD": "secret"
      },
      "timeout": 45,
      "enabled": false
    }
  ]
}
```

## Field Reference

| Field | Type | Required | Transport | Description |
|-------|------|----------|-----------|-------------|
| `name` | string | Yes | Both | Unique server identifier |
| `description` | string | No | Both | Human-readable description |
| `command` | string | stdio only | stdio | Command to execute |
| `args` | array | No | stdio | Command arguments |
| `url` | string | sse only | SSE | Base URL for SSE endpoint |
| `env` | object | No | stdio | Environment variables |
| `timeout` | number | No | Both | Connection timeout (seconds) |
| `enabled` | boolean | Yes | Both | Enable/disable server |

## Benefits

- ✅ **Simpler configuration**: No redundant transport field
- ✅ **Clear intent**: URL = remote, Command = local
- ✅ **Auto-detection**: Transport determined automatically
- ✅ **Less verbose**: Fewer fields to specify
- ✅ **Intuitive**: Configuration matches usage pattern

## Migration

If you have existing configurations with `transport` field:

1. **Remove** the `transport` field from all server configurations
2. **Keep** `url` for SSE servers
3. **Keep** `command` for stdio servers
4. **Test** that servers still connect correctly

The system will automatically detect the correct transport type based on which field is present. 