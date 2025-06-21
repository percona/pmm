# Tool Call Logging System

## Overview

The AI Chat Backend now includes comprehensive logging to detect and track when LLMs attempt to call tools, whether successful or not. This helps with debugging and monitoring tool usage.

## Recent Updates - Gemini Function Calling

✅ **FIXED**: Gemini provider now uses Google's native function calling API instead of text-based tool descriptions
✅ **IMPLEMENTED**: Proper MCP tool to Gemini function conversion
✅ **ADDED**: Function call detection and parsing for Gemini responses
✅ **ENHANCED**: Streaming support for Gemini function calls

## Logging Levels

### 🔧 Tool Call Detection
- **OpenAI Provider**: Logs when tool calls are detected in responses
- **Gemini Provider**: Now logs function calling setup and detected function calls
- **Chat Service**: Logs tool call attempts and execution results

### ⚠️ Missing Tool Call Detection
- Detects when LLM responses suggest tool usage but don't include proper tool calls
- Common patterns: "I'll use the", "Let me use", "I'll execute", etc.

### ✅ Successful Operations
- Tool execution success with result length
- Follow-up response generation after tool execution
- MCP server connections and tool loading
- Function calling setup for both providers

### ❌ Error Conditions  
- Tool execution failures
- MCP server connection issues
- Tool parsing errors
- Missing tools or servers
- Function argument parsing errors

## Log Format Examples

### OpenAI Tool Calls
```
🔧 OpenAI returned 1 tool call(s) in response
🔧 OpenAI tool call 1: ID=call_abc, Type=function, Function=list_directory, Args={"path":"."}
🔧 LLM attempted to call 1 tool(s) in response openai_chatcmpl-123
✅ Tool execution successful for list_directory, result length: 1024
```

### Gemini Function Calls (NEW!)
```
🔧 Gemini: Converted 11 MCP tools to Gemini functions
🔧 Gemini: Enabling function calling with 11 functions
🔧 Gemini function call detected: list_directory with args: map[path:.]
🔧 Gemini returned 1 function call(s) in response
🔧 Gemini function call 1: Function=list_directory, Args={"path":"."}
🔧 LLM attempted to call 1 tool(s) in response gemini_1234
✅ Tool execution successful for list_directory, result length: 1024
```

### Error Cases
```
⚠️  LLM response suggests tool usage but no tool calls were detected. Response: I'll use the database tool to check performance
❌ MCP: Tool not found: unknown_tool. Available tools: [mysql_query, postgres_query, clickhouse_query]
❌ Gemini: Failed to marshal function arguments: invalid character 'x'
```

## Implementation Details

### Chat Service (`internal/services/chat.go`)
- Added comprehensive logging for tool call detection and execution
- Added `detectToolCallAttempts()` function to identify text-based tool usage
- Enhanced streaming support with tool call detection
- Logs all tool execution attempts with detailed information

### Provider Logging
- **OpenAI** (`internal/providers/openai.go`): Logs detected tool calls from API responses
- **Gemini** (`internal/providers/gemini.go`): 
  - ✅ **NEW**: Logs function calling setup and conversion
  - ✅ **NEW**: Logs detected function calls with full details
  - ✅ **NEW**: Proper MCP tool → Gemini function conversion
  - ✅ **NEW**: Function call parsing and error handling

### MCP Service (`internal/services/mcp.go`)
- Logs available tools when requested
- Detailed tool execution logging with success/failure status
- Server connection status and tool loading information

## Key Features

1. **Tool Call Detection**: Identifies when LLMs attempt to use tools
2. **Missing Tool Detection**: Warns when responses suggest tool usage without proper calls
3. **Execution Tracking**: Monitors tool execution success/failure
4. **Provider Parity**: Both OpenAI and Gemini now support native function calling
5. **Debug Information**: Provides detailed logs for troubleshooting tool issues

## Gemini Function Calling Implementation

### What Changed
- **Before**: Gemini only received tools as text context
- **After**: Gemini uses Google's native function calling API

### MCP Tool Conversion
The provider now converts MCP tools to Gemini function declarations:
```go
// MCP Tool Schema → Gemini Function Declaration
{
  "name": "list_directory",
  "description": "List files in directory",
  "input_schema": {
    "type": "object",
    "properties": {
      "path": {"type": "string", "description": "Directory path"}
    },
    "required": ["path"]
  }
}
```

### Function Call Detection
- Gemini responses are parsed for `*genai.FunctionCall` parts
- Function arguments are properly marshaled to JSON
- Tool calls are converted to our standard `models.ToolCall` format

## Usage for Debugging

When an LLM "tries to call a tool but doesn't", check the logs for:

1. **🔧 Tool availability**: Are tools being passed to the LLM?
2. **🔧 Function calling setup**: Is the provider enabling function calling?
3. **📝 Response content**: What did the LLM actually return?
4. **⚠️ Pattern detection**: Did the response suggest tool usage without proper calls?
5. **❌ Conversion errors**: Are there issues converting MCP tools to provider format?

## Testing

Run the backend and make a request that should trigger tool usage:
```bash
AICHAT_LLM_PROVIDER=gemini AICHAT_LLM_MODEL=gemini-2.0-flash-exp GEMINI_API_KEY=... ./bin/aichat-backend
```

Try queries like:
- "What files are in the current directory?"
- "List the contents of the project root"
- "Read the README file"

## Next Steps

- [x] ~~Implement proper tool calling support for Gemini provider~~
- [ ] Add tool execution support for streaming responses  
- [ ] Add metrics collection for tool usage patterns
- [ ] Implement tool call retry mechanisms
- [ ] Test Claude provider function calling implementation 