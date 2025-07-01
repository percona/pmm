package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/aichat-backend/internal/config"
	"github.com/percona/pmm/aichat-backend/internal/models"
)

// MCPService handles MCP client connections and tool execution
type MCPService struct {
	config  *config.Config
	clients map[string]client.MCPClient
	tools   map[string]models.MCPTool
	servers map[string]config.MCPServerConfig
	mu      sync.RWMutex
	l       *logrus.Entry
}

// NewMCPService creates a new MCP service
func NewMCPService(cfg *config.Config) *MCPService {
	return &MCPService{
		config:  cfg,
		clients: make(map[string]client.MCPClient),
		tools:   make(map[string]models.MCPTool),
		servers: make(map[string]config.MCPServerConfig),
		l:       logrus.WithField("component", "mcp-service"),
	}
}

// Initialize connects to all MCP servers from JSON file
func (s *MCPService) Initialize(ctx context.Context) error {
	// Load MCP servers from JSON file
	servers, err := s.config.GetMCPServers()
	if err != nil {
		return fmt.Errorf("failed to load MCP servers: %w", err)
	}

	s.mu.Lock()
	s.servers = servers
	s.mu.Unlock()

	s.l.WithField("servers_count", len(s.servers)).Info("Initializing MCP service")

	// Connect to all configured servers
	connectedCount, connectionErrors := s.connectToAllServers(ctx, s.servers, false)

	s.l.WithField("connected_servers_count", connectedCount).Info("MCP service initialization completed")

	// Log connection errors if any
	s.logConnectionErrors(connectionErrors, "during initialization")

	return nil
}

// connectToAllServers connects to multiple MCP servers in parallel
// closeExisting: if true, closes existing connections before connecting
// Returns: (connectedCount, connectionErrors)
func (s *MCPService) connectToAllServers(ctx context.Context, servers map[string]config.MCPServerConfig, closeExisting bool) (int, []error) {
	if len(servers) == 0 {
		return 0, nil
	}

	// Close existing connections if requested
	if closeExisting {
		s.mu.Lock()
		s.closeAllClients(true)
		s.mu.Unlock()
	}

	// Connect to servers in parallel
	var wg sync.WaitGroup
	var mu sync.Mutex
	var connectionErrors []error

	for serverName, serverConfig := range servers {
		wg.Add(1)
		go func(name string, config config.MCPServerConfig) {
			defer wg.Done()

			if err := s.connectToServer(ctx, name, config); err != nil {
				s.l.WithFields(logrus.Fields{
					"server_name": name,
					"error":       err,
				}).Error("Failed to connect to MCP server")
				mu.Lock()
				connectionErrors = append(connectionErrors, fmt.Errorf("server %s: %w", name, err))
				mu.Unlock()
			}
		}(serverName, serverConfig)
	}

	// Wait for all connections to complete
	wg.Wait()

	// Get final connected count
	s.mu.RLock()
	connectedCount := len(s.clients)
	s.mu.RUnlock()

	return connectedCount, connectionErrors
}

// closeAllClients closes all existing client connections and optionally clears tools
// Must be called with mutex held
func (s *MCPService) closeAllClients(clearTools bool) error {
	var lastErr error

	if clearTools {
		s.tools = make(map[string]models.MCPTool)
	}

	for name, client := range s.clients {
		if err := client.Close(); err != nil {
			s.l.WithFields(logrus.Fields{
				"server_name": name,
				"error":       err,
			}).Warn("Failed to close existing MCP connection")
			lastErr = err
		} else {
			s.l.WithField("server_name", name).Debug("Closed MCP client")
		}
	}
	s.clients = make(map[string]client.MCPClient)

	return lastErr
}

// getServerConfigs returns a copy of all server configurations
func (s *MCPService) getServerConfigs() map[string]config.MCPServerConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	serverConfigs := make(map[string]config.MCPServerConfig)
	for name, config := range s.servers {
		serverConfigs[name] = config
	}
	return serverConfigs
}

// logConnectionErrors logs a list of connection errors in a consistent format
func (s *MCPService) logConnectionErrors(errors []error, context string) {
	if len(errors) > 0 {
		s.l.WithFields(logrus.Fields{
			"error_count": len(errors),
			"context":     context,
		}).Error("MCP connection errors occurred")
		for _, err := range errors {
			s.l.WithField("error", err).Debug("MCP connection error detail")
		}
	}
}

// connectToServer establishes connection to an MCP server
func (s *MCPService) connectToServer(ctx context.Context, serverName string, serverConfig config.MCPServerConfig) error {
	s.l.WithFields(logrus.Fields{
		"server_name":        serverName,
		"server_description": serverConfig.Description,
	}).Info("Connecting to MCP server")

	var mcpClient client.MCPClient
	var err error
	var transport string

	// Auto-detect transport based on configuration
	if serverConfig.URL != "" {
		transport = "sse"
		s.l.WithFields(logrus.Fields{
			"server_name": serverName,
			"url":         serverConfig.URL,
		}).Debug("Creating SSE MCP client")
		mcpSSEClient, err := client.NewSSEMCPClient(serverConfig.URL)
		if err != nil {
			return fmt.Errorf("failed to create SSE MCP client: %w", err)
		}
		mcpSSEClient.Start(ctx)
		mcpClient = mcpSSEClient
	} else if serverConfig.Command != "" {
		transport = "stdio"
		s.l.WithFields(logrus.Fields{
			"server_name": serverName,
			"command":     serverConfig.Command,
			"args":        serverConfig.Args,
		}).Debug("Creating stdio MCP client")

		// Convert environment map to slice of strings in KEY=VALUE format
		var envVars []string
		for key, value := range serverConfig.Env {
			envVars = append(envVars, fmt.Sprintf("%s=%s", key, value))
		}

		mcpClient, err = client.NewStdioMCPClient(serverConfig.Command, envVars, serverConfig.Args...)
		if err != nil {
			return fmt.Errorf("failed to create stdio MCP client: %w", err)
		}
	} else {
		return fmt.Errorf("server configuration must specify either URL (for SSE) or Command (for stdio)")
	}

	// Initialize the client with timeout
	initCtx := ctx
	if serverConfig.Timeout > 0 {
		var cancel context.CancelFunc
		initCtx, cancel = context.WithTimeout(ctx, time.Duration(serverConfig.Timeout)*time.Second)
		defer cancel()
	}

	// Create initialize request
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = "2024-11-05"
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "aichat-backend",
		Version: "1.0.0",
	}
	initRequest.Params.Capabilities = mcp.ClientCapabilities{
		Roots: &struct {
			ListChanged bool `json:"listChanged,omitempty"`
		}{
			ListChanged: true,
		},
		Sampling: &struct{}{},
	}

	if _, err := mcpClient.Initialize(initCtx, initRequest); err != nil {
		mcpClient.Close()
		return fmt.Errorf("failed to initialize MCP client: %w", err)
	}

	// Get available tools
	toolCount := s.loadToolsFromServer(ctx, serverName, mcpClient)
	s.l.WithFields(logrus.Fields{
		"server_name": serverName,
		"tool_count":  toolCount,
	}).Debug("Loaded tools from MCP server")

	// Store client (thread-safe)
	s.mu.Lock()
	s.clients[serverName] = mcpClient
	s.mu.Unlock()

	s.l.WithFields(logrus.Fields{
		"server_name": serverName,
		"transport":   transport,
	}).Info("Successfully connected to MCP server")
	return nil
}

// loadToolsFromServer loads tools from a specific MCP server and stores them
func (s *MCPService) loadToolsFromServer(ctx context.Context, serverName string, mcpClient client.MCPClient) int {
	// Get available tools
	toolsResp, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		s.l.WithFields(logrus.Fields{
			"server_name": serverName,
			"error":       err,
		}).Error("Failed to list tools for MCP server")
		return 0
	}

	// Store tools with server prefix (thread-safe)

	toolCount := 0
	for _, tool := range toolsResp.Tools {
		toolKey := fmt.Sprintf("%s/%s", serverName, tool.Name)

		// Convert ToolInputSchema to map[string]interface{}
		inputSchema := map[string]interface{}{
			"type": tool.InputSchema.Type,
		}
		if tool.InputSchema.Properties != nil {
			inputSchema["properties"] = tool.InputSchema.Properties
		}
		if len(tool.InputSchema.Required) > 0 {
			inputSchema["required"] = tool.InputSchema.Required
		}

		s.mu.Lock()
		s.tools[toolKey] = models.MCPTool{
			Name:        tool.Name,
			Description: fmt.Sprintf("%s (%s)", tool.Description, serverName),
			InputSchema: inputSchema,
			Server:      serverName,
		}
		s.mu.Unlock()
		toolCount++
	}

	return toolCount
}

// GetTools returns all available tools from connected MCP servers
func (s *MCPService) GetTools() []models.MCPTool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tools := make([]models.MCPTool, 0, len(s.tools))
	for _, tool := range s.tools {
		tools = append(tools, tool)
	}

	return tools
}

// RefreshTools forces a refresh of tools from all MCP servers, reconnecting if necessary
func (s *MCPService) RefreshTools() error {
	s.l.Info("Force refreshing tools from all MCP servers (with reconnection)")

	// Get list of server configurations
	serverConfigs := s.getServerConfigs()

	if len(serverConfigs) == 0 {
		s.l.Warn("No configured servers to refresh tools from")
		return nil
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Reconnect to all servers (this will close existing connections and clear tools)
	connectedCount, connectionErrors := s.connectToAllServers(ctx, serverConfigs, true)

	// Log connection errors if any
	s.logConnectionErrors(connectionErrors, "during refresh")

	// Count total tools refreshed
	s.mu.RLock()
	totalRefreshed := len(s.tools)
	s.mu.RUnlock()

	s.l.WithFields(logrus.Fields{
		"connected_servers": connectedCount,
		"total_tools":       totalRefreshed,
	}).Info("Force refresh completed")
	return nil
}

// ExecuteTool executes a tool call on the appropriate MCP server
func (s *MCPService) ExecuteTool(ctx context.Context, toolCall models.ToolCall) (string, error) {
	s.l.WithFields(logrus.Fields{
		"tool_name": toolCall.Function.Name,
		"tool_args": toolCall.Function.Arguments,
	}).Debug("Attempting to execute tool call")

	// Parse tool name to find server (with read lock)
	var serverName, toolName string
	var mcpClient client.MCPClient
	var clientExists bool

	s.mu.RLock()
	for key, tool := range s.tools {
		if tool.Name == toolCall.Function.Name {
			// Extract server name from key (format: "server/tool")
			serverName = key[:len(key)-len(tool.Name)-1]
			toolName = tool.Name
			s.l.WithFields(logrus.Fields{
				"tool_name":   toolName,
				"server_name": serverName,
			}).Debug("Found tool on server")
			break
		}
	}

	if serverName == "" {
		toolNames := s.getToolNamesUnsafe() // Already under lock
		s.mu.RUnlock()
		s.l.WithFields(logrus.Fields{
			"tool_name":       toolCall.Function.Name,
			"available_tools": toolNames,
		}).Error("Tool not found")
		return "", fmt.Errorf("tool not found: %s", toolCall.Function.Name)
	}

	// Get client reference while under lock
	mcpClient, clientExists = s.clients[serverName]
	s.mu.RUnlock()

	if !clientExists {
		s.l.WithField("server_name", serverName).Error("Server not connected")
		return "", fmt.Errorf("MCP server not connected: %s", serverName)
	}

	// Parse arguments
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		s.l.WithError(err).Error("Failed to parse tool arguments")
		return "", fmt.Errorf("failed to parse tool arguments: %w", err)
	}

	s.l.WithFields(logrus.Fields{
		"tool_name":   toolName,
		"server_name": serverName,
		"args":        args,
	}).Debug("Executing tool on server")

	// Execute tool with correct request structure
	request := mcp.CallToolRequest{}
	request.Params.Name = toolName
	request.Params.Arguments = args

	result, err := mcpClient.CallTool(ctx, request)
	if err != nil {
		s.l.WithFields(logrus.Fields{
			"tool_name":   toolName,
			"server_name": serverName,
			"error":       err,
		}).Error("Tool execution failed")
		return "", fmt.Errorf("failed to execute tool: %w", err)
	}

	// Format result
	var resultStr string
	if len(result.Content) > 0 {
		switch content := result.Content[0].(type) {
		case mcp.TextContent:
			resultStr = content.Text
		default:
			resultBytes, _ := json.Marshal(content)
			resultStr = string(resultBytes)
		}
	}

	s.l.WithFields(logrus.Fields{
		"tool_name":     toolName,
		"server_name":   serverName,
		"result_length": len(resultStr),
	}).Debug("Tool executed successfully")
	return resultStr, nil
}

// Close closes all MCP client connections
func (s *MCPService) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.closeAllClients(false)
}

// HealthCheck checks the health of all connected MCP servers
func (s *MCPService) HealthCheck(ctx context.Context) map[string]bool {
	// Create a copy of clients map to avoid holding lock during network calls
	s.mu.RLock()
	clientsCopy := make(map[string]client.MCPClient)
	for name, client := range s.clients {
		clientsCopy[name] = client
	}
	s.mu.RUnlock()

	health := make(map[string]bool)
	for name, mcpClient := range clientsCopy {
		// Simple ping to check if server is responsive
		timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		_, err := mcpClient.ListTools(timeoutCtx, mcp.ListToolsRequest{})
		health[name] = err == nil
		cancel()

		if err != nil {
			s.l.WithFields(logrus.Fields{
				"server_name": name,
				"error":       err,
			}).Debug("Health check failed for MCP server")
		}
	}

	return health
}

// getToolNamesUnsafe returns a list of available tool names - must be called with lock held
func (s *MCPService) getToolNamesUnsafe() []string {
	names := make([]string, 0, len(s.tools))
	for _, tool := range s.tools {
		names = append(names, tool.Name)
	}
	return names
}
