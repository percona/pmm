package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Server ServerConfig `yaml:"server"`
	LLM    LLMConfig    `yaml:"llm"`
	MCP    MCPConfig    `yaml:"mcp"`
}

// ServerConfig contains server configuration
type ServerConfig struct {
	Port int `yaml:"port"`
}

// LLMConfig contains LLM service configuration
type LLMConfig struct {
	Provider     string            `yaml:"provider"` // openai, anthropic, etc.
	APIKey       string            `yaml:"api_key"`
	Model        string            `yaml:"model"`
	BaseURL      string            `yaml:"base_url,omitempty"`
	SystemPrompt string            `yaml:"system_prompt,omitempty"`
	Options      map[string]string `yaml:"options,omitempty"`
}

// MCPConfig contains MCP client configuration
type MCPConfig struct {
	ServersFile string `yaml:"servers_file"`
}

// MCPServerConfig contains individual MCP server configuration
type MCPServerConfig struct {
	Description string            `json:"description,omitempty"`
	Command     string            `json:"command,omitempty"` // for stdio transport
	Args        []string          `json:"args,omitempty"`    // for stdio transport
	URL         string            `json:"url,omitempty"`     // for SSE transport
	Env         map[string]string `json:"env,omitempty"`
	Timeout     int               `json:"timeout"` // in seconds
	Enabled     bool              `json:"enabled"`
}

// MCPServersConfig represents the structure with mcpServers object
type MCPServersConfig struct {
	MCPServers map[string]MCPServerConfig `json:"mcpServers"`
}

// Load loads configuration from file and environment variables
func Load(path string) (*Config, error) {
	// Set defaults
	config := &Config{
		Server: ServerConfig{
			Port: 3001,
		},
		LLM: LLMConfig{
			Provider:     "openai",
			Model:        "gpt-4o-mini",
			SystemPrompt: "You are an AI assistant for PMM (Percona Monitoring and Management), a comprehensive database monitoring and management platform. PMM supports MySQL, PostgreSQL, MongoDB, and other database technologies. You help users with database monitoring, performance optimization, query analysis, backup management, and troubleshooting database issues. When providing assistance, focus on PMM-specific features, best practices for database monitoring, and actionable insights for database performance optimization.",
		},
		MCP: MCPConfig{
			ServersFile: "mcp-servers.json",
		},
	}

	// Try to read config file if it exists
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, err
			}
			// Config file doesn't exist, use defaults and env vars
		} else {
			// Parse YAML if file exists
			if err := yaml.Unmarshal(data, config); err != nil {
				return nil, err
			}
		}
	}

	// Override with environment variables
	loadFromEnvironment(config)

	return config, nil
}

// loadFromEnvironment loads configuration from environment variables
func loadFromEnvironment(config *Config) {
	// Server configuration
	if port := os.Getenv("AICHAT_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Server.Port = p
		}
	}
	if port := os.Getenv("PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Server.Port = p
		}
	}

	// LLM configuration
	if provider := os.Getenv("AICHAT_LLM_PROVIDER"); provider != "" {
		config.LLM.Provider = provider
	}
	if apiKey := os.Getenv("AICHAT_API_KEY"); apiKey != "" {
		config.LLM.APIKey = apiKey
	}
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		config.LLM.APIKey = apiKey
	}
	if apiKey := os.Getenv("GEMINI_API_KEY"); apiKey != "" {
		config.LLM.APIKey = apiKey
	}
	if apiKey := os.Getenv("GOOGLE_API_KEY"); apiKey != "" {
		config.LLM.APIKey = apiKey
	}
	if model := os.Getenv("AICHAT_LLM_MODEL"); model != "" {
		config.LLM.Model = model
	}
	if baseURL := os.Getenv("AICHAT_LLM_BASE_URL"); baseURL != "" {
		config.LLM.BaseURL = baseURL
	}
	if systemPrompt := os.Getenv("AICHAT_SYSTEM_PROMPT"); systemPrompt != "" {
		config.LLM.SystemPrompt = systemPrompt
	}

	// LLM options from environment
	if config.LLM.Options == nil {
		config.LLM.Options = make(map[string]string)
	}
	if temp := os.Getenv("AICHAT_LLM_TEMPERATURE"); temp != "" {
		config.LLM.Options["temperature"] = temp
	}
	if maxTokens := os.Getenv("AICHAT_LLM_MAX_TOKENS"); maxTokens != "" {
		config.LLM.Options["max_tokens"] = maxTokens
	}
	if topP := os.Getenv("AICHAT_LLM_TOP_P"); topP != "" {
		config.LLM.Options["top_p"] = topP
	}

	// MCP configuration
	if serversFile := os.Getenv("AICHAT_MCP_SERVERS_FILE"); serversFile != "" {
		config.MCP.ServersFile = serversFile
	}
	if serversFile := os.Getenv("AICHAT_CONFIG_FILE"); serversFile != "" {
		// Legacy support
		config.MCP.ServersFile = filepath.Join(filepath.Dir(serversFile), "mcp-servers.json")
	}
}

// GetEnabledMCPServers returns only the enabled MCP servers as a map
func (c *Config) GetEnabledMCPServers() (map[string]MCPServerConfig, error) {
	// Parse as mcpServers format
	serversFilePath := c.MCP.ServersFile
	if !filepath.IsAbs(serversFilePath) {
		serversFilePath = filepath.Join(".", serversFilePath)
	}

	// Check if servers file exists
	if _, err := os.Stat(serversFilePath); os.IsNotExist(err) {
		return map[string]MCPServerConfig{}, nil
	}

	serversData, err := os.ReadFile(serversFilePath)
	if err != nil {
		return nil, err
	}

	var serversConfig MCPServersConfig
	if err := json.Unmarshal(serversData, &serversConfig); err != nil {
		return nil, err
	}

	if serversConfig.MCPServers == nil {
		return map[string]MCPServerConfig{}, nil
	}

	// Set defaults for missing fields
	enabledServers := make(map[string]MCPServerConfig)
	for name, serverConfig := range serversConfig.MCPServers {
		if serverConfig.Timeout == 0 {
			serverConfig.Timeout = 30 // Default timeout
		}
		serverConfig.Enabled = true // All servers are enabled
		enabledServers[name] = serverConfig
	}

	return enabledServers, nil
}

// GetConfigFromEnv loads configuration entirely from environment variables
func GetConfigFromEnv() *Config {
	config := &Config{
		Server: ServerConfig{
			Port: 3001,
		},
		LLM: LLMConfig{
			Provider:     "openai",
			Model:        "gpt-4o-mini",
			SystemPrompt: "You are an AI assistant for PMM (Percona Monitoring and Management), a comprehensive database monitoring and management platform. PMM supports MySQL, PostgreSQL, MongoDB, and other database technologies. You help users with database monitoring, performance optimization, query analysis, backup management, and troubleshooting database issues. When providing assistance, focus on PMM-specific features, best practices for database monitoring, and actionable insights for database performance optimization.",
		},
		MCP: MCPConfig{
			ServersFile: "mcp-servers.json",
		},
	}

	loadFromEnvironment(config)
	return config
}
