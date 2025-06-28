package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config contains all application configuration
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	LLM      LLMConfig      `yaml:"llm"`
	MCP      MCPConfig      `yaml:"mcp"`
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

// Load loads configuration from file only (no environment variable handling)
func Load(path string) (*Config, error) {
	// Set defaults
	config := GetConfigFromDefaults()

	// Try to read config file if it exists
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, err
			}
			// Config file doesn't exist, use defaults
		} else {
			// Parse YAML if file exists
			if err := yaml.Unmarshal(data, config); err != nil {
				return nil, err
			}
		}
	}

	return config, nil
}

// GetConfigFromDefaults returns configuration with default values only
func GetConfigFromDefaults() *Config {
	return &Config{
		Server: ServerConfig{
			Port: 3001,
		},
		Database: DatabaseConfig{
			DSN: "postgres://ai_chat_user:ai_chat_secure_password@127.0.0.1:5432/ai_chat?sslmode=disable",
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
