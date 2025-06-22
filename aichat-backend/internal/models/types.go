package models

import (
	"time"
)

// Attachment represents a file attachment
type Attachment struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	MimeType string `json:"mime_type"`
	Size     int64  `json:"size"`
	Content  string `json:"content,omitempty"` // Base64 encoded content for small files
	Path     string `json:"path,omitempty"`    // File path for larger files
}

// Message represents a chat message
type Message struct {
	ID             string          `json:"id"`
	Role           string          `json:"role"` // user, assistant, system
	Content        string          `json:"content"`
	Timestamp      time.Time       `json:"timestamp"`
	ToolCalls      []ToolCall      `json:"tool_calls,omitempty"`
	ToolExecutions []ToolExecution `json:"tool_executions,omitempty"`
	Attachments    []Attachment    `json:"attachments,omitempty"`
}

// ToolCall represents a tool function call
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// ToolExecution represents the execution details of a tool call
type ToolExecution struct {
	ID        string    `json:"id"`              // Matches ToolCall.ID
	ToolName  string    `json:"tool_name"`       // Name of the tool that was executed
	Arguments string    `json:"arguments"`       // Arguments passed to the tool (JSON string)
	Result    string    `json:"result"`          // Result returned by the tool
	Error     string    `json:"error,omitempty"` // Error message if execution failed
	StartTime time.Time `json:"start_time"`      // When the tool execution started
	EndTime   time.Time `json:"end_time"`        // When the tool execution completed
	Duration  int64     `json:"duration_ms"`     // Execution duration in milliseconds
}

// ChatRequest represents an incoming chat message
type ChatRequest struct {
	Message     string            `json:"message" binding:"required"`
	SessionID   string            `json:"session_id,omitempty"`
	Context     map[string]string `json:"context,omitempty"`
	Attachments []Attachment      `json:"attachments,omitempty"`
}

// ChatResponse represents a chat response
type ChatResponse struct {
	Message   *Message `json:"message"`
	SessionID string   `json:"session_id"`
	Error     string   `json:"error,omitempty"`
}

// ChatHistory represents the chat history
type ChatHistory struct {
	SessionID string     `json:"session_id"`
	Messages  []*Message `json:"messages"`
}

// MCPTool represents an available MCP tool
type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
	Server      string                 `json:"server"`
}

// MCPToolsResponse represents the available MCP tools
type MCPToolsResponse struct {
	Tools        []MCPTool `json:"tools"`
	ForceRefresh bool      `json:"force_refresh,omitempty"`
}

// ToolApprovalRequest represents a request for user approval to execute tools
type ToolApprovalRequest struct {
	SessionID string     `json:"session_id"`
	ToolCalls []ToolCall `json:"tool_calls"`
	RequestID string     `json:"request_id"`
}

// ToolApprovalResponse represents the user's response to tool execution request
type ToolApprovalResponse struct {
	SessionID   string   `json:"session_id"`
	RequestID   string   `json:"request_id"`
	Approved    bool     `json:"approved"`
	ApprovedIDs []string `json:"approved_ids,omitempty"` // Allow selective approval
}

// StreamMessage represents a streaming chat message chunk
type StreamMessage struct {
	Type           string          `json:"type"` // message, tool_call, tool_execution, tool_approval_request, error, done
	Content        string          `json:"content,omitempty"`
	SessionID      string          `json:"session_id"`
	Error          string          `json:"error,omitempty"`
	Done           bool            `json:"done,omitempty"`
	ToolCalls      []ToolCall      `json:"tool_calls,omitempty"`
	ToolExecutions []ToolExecution `json:"tool_executions,omitempty"`
	RequestID      string          `json:"request_id,omitempty"` // For tool approval requests
}
