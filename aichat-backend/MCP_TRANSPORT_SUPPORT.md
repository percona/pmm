# MCP Transport Support

The AI Chat Backend supports both **stdio** and **SSE** (Server-Sent Events) transport methods for connecting to MCP servers with automatic transport detection.

## Transport Types

### 1. stdio Transport
- **Use case**: Local MCP servers that run as separate processes
- **Communication**: Standard input/output pipes
- **Configuration**: Specify `command` and `args` fields
- **Examples**: Filesystem servers, local database tools, git operations

### 2. SSE Transport  
- **Use case**: Remote MCP servers accessible via HTTP
- **Communication**: Server-Sent Events over HTTP
- **Configuration**: Specify `url` field
- **Examples**: Web APIs, cloud services, remote databases

## Configuration Schema

```json
{
  "servers": [
    {
      "name": "string",           // Required: Unique server identifier
      "description": "string",    // Optional: Human-readable description
      "command": "string",        // stdio only: Command to execute
      "args": ["string"],         // stdio only: Command arguments
      "url": "string",            // sse only: Base URL for SSE endpoint
      "env": {},                  // stdio only: Environment variables
      "timeout": 30,              // Optional: Connection timeout (seconds)
      "enabled": true             // Required: Enable/disable server
    }
  ]
}
```

## Auto-Detection Logic

Transport type is automatically determined:
- If `url` is present → **SSE transport**
- If `command` is present → **stdio transport**
- If neither is present → **Configuration error**
- Both fields cannot be specified for the same server

## Implementation Details

### Client Creation
```go
// Auto-detect transport based on configuration
if serverConfig.URL != "" {
    transport = "sse"
    mcpClient, err = client.NewSSEMCPClient(serverConfig.URL)
} else if serverConfig.Command != "" {
    transport = "stdio"
    mcpClient, err = client.NewStdioMCPClient(serverConfig.Command, serverConfig.Args...)
} else {
    return fmt.Errorf("server configuration must specify either URL (for SSE) or Command (for stdio)")
}
```

### Connection Flow
1. **Transport Detection**: Determine transport type from URL vs Command presence
2. **Client Creation**: Create appropriate MCP client (SSE or stdio)
3. **Initialization**: Send MCP initialize request with capabilities
4. **Tool Discovery**: List available tools from the server
5. **Registration**: Register tools in the service registry

### Error Handling
- **Missing configuration**: Validation for required fields per transport
- **Connection failures**: Graceful handling with detailed logging
- **Timeout handling**: Configurable timeouts per server

## Configuration Examples

### stdio Transport Examples

#### Filesystem Server
```json
{
  "name": "filesystem",
  "description": "Local file operations",
  "command": "npx",
  "args": ["@modelcontextprotocol/server-filesystem", "/workspace"],
  "timeout": 30,
  "enabled": true
}
```

#### Database Server
```json
{
  "name": "postgres",
  "description": "PostgreSQL operations",
  "command": "python",
  "args": ["-m", "mcp_server_postgres", "--connection-string", "postgresql://user:pass@localhost/db"],
  "env": {
    "PGPASSWORD": "secret"
  },
  "timeout": 45,
  "enabled": true
}
```

### SSE Transport Examples

#### Remote API Server
```json
{
  "name": "remote-api",
  "description": "Remote tool server",
  "url": "https://api.example.com/mcp/sse",
  "timeout": 60,
  "enabled": true
}
```

#### Local Web Server
```json
{
  "name": "web-tools",
  "description": "Local web-based MCP server",
  "url": "http://localhost:8080/mcp",
  "timeout": 30,
  "enabled": true
}
```

## API Endpoints

The backend exposes these endpoints for MCP management:

- `GET /api/v1/mcp/tools` - List all available tools from all servers
- `GET /api/v1/mcp/servers/status` - Get connection status of all servers

### Server Status Response
```json
{
  "filesystem": {
    "name": "filesystem",
    "description": "File operations",
    "transport": "stdio",
    "enabled": true,
    "connected": true,
    "tool_count": 5
  },
  "remote-api": {
    "name": "remote-api", 
    "description": "Remote tools",
    "transport": "sse",
    "enabled": true,
    "connected": false,
    "tool_count": 0
  }
}
```

## Benefits

### stdio Transport
- ✅ Direct process control
- ✅ Environment variable support
- ✅ Local resource access
- ✅ No network dependencies

### SSE Transport  
- ✅ Remote server support
- ✅ Web-based integration
- ✅ Scalable architecture
- ✅ HTTP-based debugging

## Migration Guide

### Configuration simplification:

1. **Remove transport field**: No longer needed - auto-detected from URL vs Command
2. **stdio servers**: Ensure `command` field is present
3. **SSE servers**: Ensure `url` field is present
4. **Test connections**: Verify both transport types work correctly
5. **Monitor status**: Use status endpoint to check server health

### Configuration validation:
- stdio servers must have `command` field
- SSE servers must have `url` field  
- Cannot specify both `url` and `command` for the same server

## Troubleshooting

### Common Issues

1. **stdio Transport**
   - Command not found → Check PATH and command availability
   - Permission denied → Verify execute permissions
   - Timeout errors → Increase timeout or check server startup time

2. **SSE Transport**
   - Connection refused → Verify URL and server availability
   - HTTP errors → Check server logs and network connectivity
   - Timeout errors → Increase timeout for slow servers

### Debug Logging

Enable debug logging for MCP connections:
```json
{
  "env": {
    "DEBUG": "mcp*",
    "LOG_LEVEL": "debug"
  }
}
```

## Future Enhancements

- **Authentication**: Support for API keys and tokens in SSE transport
- **Load balancing**: Multiple URLs for SSE servers
- **Health checks**: Periodic connection validation
- **Metrics**: Connection and performance monitoring 