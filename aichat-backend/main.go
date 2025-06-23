package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/percona/pmm/aichat-backend/internal/config"
	"github.com/percona/pmm/aichat-backend/internal/handlers"
	"github.com/percona/pmm/aichat-backend/internal/services"
	"github.com/percona/pmm/version"
)

//go:embed migrations/*.sql
var embeddedMigrations embed.FS

func main() {

	var (
		configPath  string
		envOnly     bool
		showVersion bool
	)

	flag.StringVar(&configPath, "config", "config.yaml", "Path to configuration file")
	flag.BoolVar(&envOnly, "env-only", false, "Load configuration only from environment variables")
	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.Parse()

	if showVersion {
		fmt.Printf("AI Chat Backend\n%s\n", version.FullInfo())
		os.Exit(0)
	}

	// Load configuration
	var cfg *config.Config
	var err error

	if envOnly {
		log.Printf("Loading configuration from environment variables only")
		cfg = config.GetConfigFromEnv()
	} else {
		log.Printf("Loading configuration from file: %s (with environment variable overrides)", configPath)
		cfg, err = config.Load(configPath)
		if err != nil {
			log.Fatalf("Failed to load configuration: %v", err)
		}
	}

	// Validate required configuration
	if err := validateConfig(cfg); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	log.Printf("Starting AI Chat Backend on port %d", cfg.Server.Port)
	log.Printf("LLM Provider: %s, Model: %s", cfg.LLM.Provider, cfg.LLM.Model)
	log.Printf("MCP Servers File: %s", cfg.MCP.ServersFile)

	// Initialize database service
	dbConfig := config.GetDatabaseConfig()
	db, err := dbConfig.OpenDatabase()
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer func() {
		if db != nil {
			db.Close()
		}
	}()

	// Create database service
	databaseService := services.NewDatabaseService(db)

	// Create migration service and run migrations
	log.Printf("Running database migrations...")
	migrationService := services.NewMigrationService(db, embeddedMigrations)

	if err := migrationService.RunMigrations(); err != nil {
		log.Fatalf("❌ Failed to run database migrations: %v", err)
	}

	// Log current migration version
	migrationVersion, dirty, err := migrationService.GetMigrationVersion()
	if err != nil {
		log.Printf("⚠️ Warning: Failed to get migration version: %v", err)
	} else {
		if dirty {
			log.Printf("⚠️ Warning: Database is in dirty state (version %d)", migrationVersion)
		} else {
			log.Printf("✅ Database migration version: %d", migrationVersion)
		}
	}

	// Initialize services
	llmService := services.NewLLMService(cfg.LLM)
	mcpService := services.NewMCPService(cfg)

	// Initialize MCP service (connect to servers)
	ctx := context.Background()
	if err := mcpService.Initialize(ctx); err != nil {
		log.Printf("Warning: Failed to initialize MCP service: %v", err)
	}

	chatService := services.NewChatService(llmService, mcpService, databaseService)

	// Set system prompt from configuration
	if cfg.LLM.SystemPrompt != "" {
		chatService.SetSystemPrompt(cfg.LLM.SystemPrompt)
		log.Printf("✅ System prompt configured (%d characters)", len(cfg.LLM.SystemPrompt))
	}

	// Initialize HTTP handlers
	chatHandler := handlers.NewChatHandler(chatService)
	sessionHandler := handlers.NewSessionHandler(databaseService)

	// Setup router
	router := gin.Default()

	// Configure CORS
	corsConfig := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		AllowCredentials: true,
	}

	// Set CORS origins from environment or use defaults
	if origins := os.Getenv("AICHAT_CORS_ORIGINS"); origins != "" {
		corsConfig.AllowOrigins = []string{origins}
	} else {
		corsConfig.AllowOrigins = []string{
			"http://localhost:3000",
			"http://localhost:5173",
			"http://localhost:8080",
			"http://localhost:8443",
		}
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

		// MCP operations
		v1chat.GET("/mcp/tools", chatHandler.GetMCPTools)

		// Session management
		v1chat.POST("/sessions", sessionHandler.CreateSession)
		v1chat.GET("/sessions", sessionHandler.ListSessions)
		v1chat.GET("/sessions/:id", sessionHandler.GetSession)
		v1chat.PUT("/sessions/:id", sessionHandler.UpdateSession)
		v1chat.DELETE("/sessions/:id", sessionHandler.DeleteSession)
		v1chat.GET("/sessions/:id/messages", sessionHandler.GetSessionMessages)
	}

	// Start server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: router,
	}

	go func() {
		log.Printf("API endpoints available at http://localhost:%d/v1/chat", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Close chat service (this will close both LLM and MCP services)
	if err := chatService.Close(); err != nil {
		log.Printf("Error closing chat service: %v", err)
	}

	// The context is used to inform the server it has 5 seconds to finish
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
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
