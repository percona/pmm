package main

import (
	"context"
	"embed"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/alecthomas/kong"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/aichat-backend/internal/config"
	"github.com/percona/pmm/aichat-backend/internal/handlers"
	"github.com/percona/pmm/aichat-backend/internal/services"
	"github.com/percona/pmm/version"
)

//go:embed migrations/*.sql
var embeddedMigrations embed.FS

// CLI represents the command line interface structure
type CLI struct {
	// Configuration
	Config  string `short:"c" long:"config" default:"config.yaml" help:"Path to configuration file" type:"path" group:"Configuration" env:"AICHAT_CONFIG"`
	EnvOnly bool   `long:"env-only" help:"Load configuration only from environment variables" group:"Configuration" env:"AICHAT_ENV_ONLY"`
	Version bool   `short:"v" long:"version" help:"Show version information" group:"Configuration"`

	// Server options
	Port int `long:"port" help:"Override server port from config" placeholder:"PORT" group:"Server" env:"AICHAT_PORT"`

	// Logging options
	LogLevel string `long:"log-level" enum:"debug,info,warn,error" default:"info" help:"Set log level" group:"Logging" env:"AICHAT_LOG_LEVEL"`
	LogJSON  bool   `long:"log-json" help:"Output logs in JSON format" group:"Logging" env:"AICHAT_LOG_JSON"`

	// Database options
	DatabaseURL string `long:"database-url" help:"Override database URL from config" placeholder:"URL" group:"Database" env:"AICHAT_DATABASE_URL"`

	// LLM options
	Provider     string `long:"llm-provider" help:"Override LLM provider from config" placeholder:"PROVIDER" group:"LLM" env:"AICHAT_LLM_PROVIDER"`
	Model        string `long:"llm-model" help:"Override LLM model from config" placeholder:"MODEL" group:"LLM" env:"AICHAT_LLM_MODEL"`
	APIKey       string `long:"api-key" help:"Override API key from config" placeholder:"KEY" group:"LLM" env:"AICHAT_API_KEY"`
	BaseURL      string `long:"base-url" help:"Override LLM base URL from config" placeholder:"URL" group:"LLM" env:"AICHAT_LLM_BASE_URL"`
	SystemPrompt string `long:"system-prompt" help:"Override system prompt from config" placeholder:"PROMPT" group:"LLM" env:"AICHAT_SYSTEM_PROMPT"`

	// MCP options
	MCPServersFile string `long:"mcp-servers-file" help:"Override MCP servers file from config" placeholder:"FILE" group:"MCP" env:"AICHAT_MCP_SERVERS_FILE"`
}

func main() {
	// Parse command line arguments with Kong
	var cli CLI
	ctx := kong.Parse(&cli,
		kong.Name("aichat-backend"),
		kong.Description("AI Chat Backend server for PMM"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
			Summary: true,
		}),
	)

	// Initialize logger with component field
	l := logrus.WithField("component", "aichat-backend")

	// Configure logging based on CLI options
	if cli.LogJSON {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	}

	// Set log level (Kong ensures it's valid due to enum constraint)
	if level, err := logrus.ParseLevel(cli.LogLevel); err != nil {
		l.WithError(err).Fatal("Invalid log level")
	} else {
		logrus.SetLevel(level)
	}

	// Handle version flag
	if cli.Version {
		fmt.Printf("AI Chat Backend\n%s\n", version.FullInfo())
		ctx.Exit(0)
	}

	// Load configuration
	var cfg *config.Config
	var err error

	if cli.EnvOnly {
		l.Info("Loading configuration from defaults only (environment variables handled by Kong)")
		cfg = config.GetConfigFromDefaults()
	} else {
		l.WithField("config_path", cli.Config).Info("Loading configuration from file")
		cfg, err = config.Load(cli.Config)
		if err != nil {
			l.WithError(err).Fatal("Failed to load configuration")
		}
	}

	// Apply CLI overrides to configuration
	applyCliOverrides(cfg, &cli, l)

	// Validate required configuration
	if err := validateConfig(cfg); err != nil {
		l.WithError(err).Fatal("Configuration validation failed")
	}

	l.WithFields(logrus.Fields{
		"port":         cfg.Server.Port,
		"llm_provider": cfg.LLM.Provider,
		"llm_model":    cfg.LLM.Model,
		"mcp_servers":  cfg.MCP.ServersFile,
	}).Info("Starting AI Chat Backend")

	// Create database service
	l.Debug("Creating database service")
	databaseService, err := services.NewDatabaseService(&cfg.Database)
	if err != nil {
		l.WithError(err).Fatal("Failed to initialize database service")
	}

	// Create migration service and run migrations
	l.Info("Running database migrations")
	migrationService := services.NewMigrationService(databaseService.DB(), embeddedMigrations)

	if err := migrationService.RunMigrations(); err != nil {
		l.WithError(err).Fatal("Failed to run database migrations")
	}

	// Log current migration version
	migrationVersion, dirty, err := migrationService.GetMigrationVersion()
	if err != nil {
		l.WithError(err).Warn("Failed to get migration version")
	} else {
		if dirty {
			l.WithField("version", migrationVersion).Warn("Database is in dirty state")
		} else {
			l.WithField("version", migrationVersion).Info("Database migration completed")
		}
	}

	// Initialize services
	l.Debug("Initializing services")
	llmService := services.NewLLMService(cfg.LLM)
	mcpService := services.NewMCPService(cfg)

	// Initialize MCP service (connect to servers)
	serviceCtx := context.Background()
	if err := mcpService.Initialize(serviceCtx); err != nil {
		l.WithError(err).Warn("Failed to initialize MCP service")
	}

	chatService := services.NewChatService(llmService, mcpService, databaseService)

	// Set system prompt from configuration
	if cfg.LLM.SystemPrompt != "" {
		chatService.SetSystemPrompt(cfg.LLM.SystemPrompt)
		l.WithField("prompt_length", len(cfg.LLM.SystemPrompt)).Info("System prompt configured")
	}

	// Initialize HTTP handlers
	l.Debug("Initializing HTTP handlers")
	chatHandler := handlers.NewChatHandler(chatService)
	sessionHandler := handlers.NewSessionHandler(databaseService)

	// Setup router
	router := gin.Default()

	// Configure CORS
	corsConfig := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		AllowCredentials: true,
		AllowAllOrigins:  true,
	}

	router.Use(cors.New(corsConfig))

	// Health check endpoint
	router.GET("/v1/chat/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"version":   version.ShortInfo(),
		})
	})

	// Chat endpoints
	v1chat := router.Group("/v1/chat")
	{
		// Chat operations
		v1chat.POST("/send", chatHandler.SendMessage)
		v1chat.POST("/send-with-files", chatHandler.SendMessageWithFiles)
		v1chat.DELETE("/clear", chatHandler.ClearHistory)
		v1chat.GET("/stream", chatHandler.StreamChat)
		v1chat.GET("/stream/:streamId", chatHandler.StreamByID)

		// MCP operations
		v1chat.GET("/mcp/tools", chatHandler.GetMCPTools)

		// Session management
		v1chat.POST("/sessions", sessionHandler.CreateSession)
		v1chat.GET("/sessions", sessionHandler.ListSessions)
		v1chat.GET("/sessions/:id", sessionHandler.GetSession)
		v1chat.PUT("/sessions/:id", sessionHandler.UpdateSession)
		v1chat.DELETE("/sessions/:id", sessionHandler.DeleteSession)
		v1chat.GET("/sessions/:id/messages", sessionHandler.GetSessionMessages)
		v1chat.DELETE("/sessions/:id/messages", sessionHandler.ClearSessionMessages)
	}

	// Start server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: router,
	}

	go func() {
		l.WithField("port", cfg.Server.Port).Info("AI Chat Backend server started")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			l.WithError(err).Fatal("Failed to start server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	l.Info("Shutting down server")

	// Give outstanding requests 30 seconds to complete
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		l.WithError(err).Error("Server forced to shutdown")
	} else {
		l.Info("Server gracefully stopped")
	}

	// Close chat service (including stream manager)
	if err := chatService.Close(); err != nil {
		l.WithError(err).Error("Error closing chat service")
	}

	// Close MCP service
	if err := mcpService.Close(); err != nil {
		l.WithError(err).Error("Error closing MCP service")
	}

	l.Info("AI Chat Backend shutdown complete")
}

// maskPassword masks the password in a DSN for logging
func maskPassword(dsn string) string {
	if strings.Contains(dsn, "@") {
		// For postgres://user:pass@host format
		parts := strings.Split(dsn, "@")
		if len(parts) == 2 {
			userPart := parts[0]
			if strings.Contains(userPart, ":") {
				userPass := strings.Split(userPart, ":")
				if len(userPass) >= 2 {
					return userPass[0] + ":***@" + parts[1]
				}
			}
		}
	}
	return dsn // Return as-is if we can't parse it
}

// applyCliOverrides applies command line overrides to the configuration
func applyCliOverrides(cfg *config.Config, cli *CLI, l *logrus.Entry) {
	if cli.Port != 0 {
		l.WithFields(logrus.Fields{
			"old_port": cfg.Server.Port,
			"new_port": cli.Port,
		}).Info("Overriding server port from CLI")
		cfg.Server.Port = cli.Port
	}

	if cli.Provider != "" {
		l.WithFields(logrus.Fields{
			"old_provider": cfg.LLM.Provider,
			"new_provider": cli.Provider,
		}).Info("Overriding LLM provider from CLI")
		cfg.LLM.Provider = cli.Provider
	}

	if cli.Model != "" {
		l.WithFields(logrus.Fields{
			"old_model": cfg.LLM.Model,
			"new_model": cli.Model,
		}).Info("Overriding LLM model from CLI")
		cfg.LLM.Model = cli.Model
	}

	if cli.APIKey != "" {
		l.Info("Overriding API key from CLI")
		cfg.LLM.APIKey = cli.APIKey
	}

	if cli.DatabaseURL != "" {
		l.WithFields(logrus.Fields{
			"old_dsn": maskPassword(cfg.Database.DSN),
			"new_dsn": maskPassword(cli.DatabaseURL),
		}).Info("Overriding database URL from CLI")
		cfg.Database.DSN = cli.DatabaseURL
	}

	if cli.BaseURL != "" {
		l.Info("Overriding LLM base URL from CLI")
		cfg.LLM.BaseURL = cli.BaseURL
	}

	if cli.SystemPrompt != "" {
		l.Info("Overriding system prompt from CLI")
		cfg.LLM.SystemPrompt = cli.SystemPrompt
	}

	if cli.MCPServersFile != "" {
		l.Info("Overriding MCP servers file from CLI")
		cfg.MCP.ServersFile = cli.MCPServersFile
	}
}

// validateConfig validates the required configuration settings
func validateConfig(cfg *config.Config) error {
	// Check if API key is required based on provider
	switch cfg.LLM.Provider {
	case "openai":
		if cfg.LLM.APIKey == "" {
			return fmt.Errorf("OpenAI API key is required (set OPENAI_API_KEY or AICHAT_API_KEY environment variable)")
		}
	case "gemini", "google":
		if cfg.LLM.APIKey == "" {
			return fmt.Errorf("Google Gemini API key is required (set GEMINI_API_KEY, GOOGLE_API_KEY, or AICHAT_API_KEY environment variable)")
		}
	case "mock":
		// Mock provider doesn't require an API key
	case "claude", "anthropic":
		if cfg.LLM.APIKey == "" {
			return fmt.Errorf("Anthropic Claude API key is required (set ANTHROPIC_API_KEY or AICHAT_API_KEY environment variable)")
		}
	case "ollama":
		// Ollama typically doesn't require an API key (local deployment)
	default:
		// For unknown providers, require API key
		if cfg.LLM.APIKey == "" {
			return fmt.Errorf("API key is required for provider %s (set AICHAT_API_KEY environment variable)", cfg.LLM.Provider)
		}
	}

	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		return fmt.Errorf("invalid port number: %d", cfg.Server.Port)
	}

	return nil
}
