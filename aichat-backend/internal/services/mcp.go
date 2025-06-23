package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
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
}

// NewMCPService creates a new MCP service
func NewMCPService(cfg *config.Config) *MCPService {
	return &MCPService{
		config:  cfg,
		clients: make(map[string]client.MCPClient),
		tools:   make(map[string]models.MCPTool),
		servers: make(map[string]config.MCPServerConfig),
	}
}

// Initialize connects to all enabled MCP servers from JSON file
func (s *MCPService) Initialize(ctx context.Context) error {
	// Load MCP servers from JSON file
	servers, err := s.config.GetEnabledMCPServers()
	if err != nil {
		return fmt.Errorf("failed to load MCP servers: %w", err)
	}

	s.mu.Lock()
	s.servers = servers
	s.mu.Unlock()

	log.Printf("Found %d enabled MCP servers to connect to", len(s.servers))

	// Connect to all configured servers
	connectedCount, connectionErrors := s.connectToAllServers(ctx, s.servers, false)

	log.Printf("Successfully initialized MCP service with %d active servers", connectedCount)

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
				log.Printf("Failed to connect to MCP server %s: %v", name, err)
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
			log.Printf("⚠️ MCP: Failed to close existing connection to %s: %v", name, err)
			lastErr = err
		} else {
			log.Printf("Closed MCP client: %s", name)
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
		log.Printf("❌ MCP: Connection errors occurred for %d servers %s:", len(errors), context)
		for _, err := range errors {
			log.Printf("  - %v", err)
		}
	}
}

// connectToServer establishes connection to an MCP server
func (s *MCPService) connectToServer(ctx context.Context, serverName string, serverConfig config.MCPServerConfig) error {
	log.Printf("Connecting to MCP server: %s (%s)", serverName, serverConfig.Description)

	var mcpClient client.MCPClient
	var err error
	var transport string

	// Auto-detect transport based on configuration
	if serverConfig.URL != "" {
		transport = "sse"
		log.Printf("Creating SSE MCP client for URL: %s", serverConfig.URL)
		mcpSSEClient, err := client.NewSSEMCPClient(serverConfig.URL)
		if err != nil {
			return fmt.Errorf("failed to create SSE MCP client: %w", err)
		}
		mcpSSEClient.Start(ctx)
		mcpClient = mcpSSEClient
	} else if serverConfig.Command != "" {
		transport = "stdio"
		log.Printf("Creating stdio MCP client for command: %s %v", serverConfig.Command, serverConfig.Args)

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
	log.Printf("Loaded %d tools from MCP server %s", toolCount, serverName)

	// Store client (thread-safe)
	s.mu.Lock()
	s.clients[serverName] = mcpClient
	s.mu.Unlock()

	log.Printf("Successfully connected to MCP server: %s using %s transport", serverName, transport)
	return nil
}

// loadToolsFromServer loads tools from a specific MCP server and stores them
func (s *MCPService) loadToolsFromServer(ctx context.Context, serverName string, mcpClient client.MCPClient) int {
	// Get available tools
	toolsResp, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		log.Printf("❌ MCP: Failed to list tools for server %s: %v", serverName, err)
		return 0
	}

	// Store tools with server prefix (thread-safe)
	s.mu.Lock()
	defer s.mu.Unlock()

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

		s.tools[toolKey] = models.MCPTool{
			Name:        tool.Name,
			Description: fmt.Sprintf("%s (%s)", tool.Description, serverName),
			InputSchema: inputSchema,
			Server:      serverName,
		}
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
	log.Printf("🔄 MCP: Force refreshing tools from all MCP servers (with reconnection)")

	// Get list of server configurations
	serverConfigs := s.getServerConfigs()

	if len(serverConfigs) == 0 {
		log.Printf("🔄 MCP: No configured servers to refresh tools from")
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

	log.Printf("🔄 MCP: Force refresh completed. Connected servers: %d, Total tools refreshed: %d", connectedCount, totalRefreshed)
	return nil
}

// ExecuteTool executes a tool call on the appropriate MCP server
func (s *MCPService) ExecuteTool(ctx context.Context, toolCall models.ToolCall) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	log.Printf("🔧 MCP: Attempting to execute tool call: %s with args: %s", toolCall.Function.Name, toolCall.Function.Arguments)

	// Parse tool name to find server
	var serverName, toolName string
	for key, tool := range s.tools {
		if tool.Name == toolCall.Function.Name {
			// Extract server name from key (format: "server/tool")
			serverName = key[:len(key)-len(tool.Name)-1]
			toolName = tool.Name
			log.Printf("🔧 MCP: Found tool %s on server %s", toolName, serverName)
			break
		}
	}

	if serverName == "" {
		log.Printf("❌ MCP: Tool not found: %s. Available tools: %v", toolCall.Function.Name, s.getToolNames())
		return "", fmt.Errorf("tool not found: %s", toolCall.Function.Name)
	}

	mcpClient, exists := s.clients[serverName]
	if !exists {
		log.Printf("❌ MCP: Server not connected: %s", serverName)
		return "", fmt.Errorf("MCP server not connected: %s", serverName)
	}

	// Parse arguments
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		log.Printf("❌ MCP: Failed to parse tool arguments: %v", err)
		return "", fmt.Errorf("failed to parse tool arguments: %w", err)
	}

	log.Printf("🔧 MCP: Executing tool %s on server %s with args: %v", toolName, serverName, args)

	// Execute tool with correct request structure
	request := mcp.CallToolRequest{}
	request.Params.Name = toolName
	request.Params.Arguments = args

	result, err := mcpClient.CallTool(ctx, request)
	if err != nil {
		log.Printf("❌ MCP: Tool execution failed: %v", err)
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

	log.Printf("✅ MCP: Tool %s executed successfully, result length: %d", toolName, len(resultStr))
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
	s.mu.RLock()
	defer s.mu.RUnlock()

	health := make(map[string]bool)
	for name, mcpClient := range s.clients {
		// Simple ping to check if server is responsive
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		_, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
		health[name] = err == nil
		cancel()
	}

	return health
}

// getToolNames returns a list of available tool names for logging
func (s *MCPService) getToolNames() []string {
	names := make([]string, 0, len(s.tools))
	for _, tool := range s.tools {
		names = append(names, tool.Name)
	}
	return names
}
