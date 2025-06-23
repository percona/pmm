package services

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/percona/pmm/aichat-backend/internal/config"
	"github.com/percona/pmm/aichat-backend/internal/models"
)

func TestMCPService_ThreadSafety(t *testing.T) {
	cfg := &config.Config{}
	service := NewMCPService(cfg)

	// Test concurrent access to tools and clients maps
	var wg sync.WaitGroup
	numGoroutines := 10

	// Test concurrent writes to tools map
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(serverID int) {
			defer wg.Done()

			// Simulate loading tools (this would normally be called from loadToolsFromServer)
			service.mu.Lock()
			service.tools[fmt.Sprintf("server%d/tool1", serverID)] = models.MCPTool{
				Name:        "tool1",
				Description: fmt.Sprintf("Tool from server %d", serverID),
				Server:      fmt.Sprintf("server%d", serverID),
			}
			service.mu.Unlock()
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Verify all tools were stored
	tools := service.GetTools()
	if len(tools) != numGoroutines {
		t.Errorf("Expected %d tools, got %d", numGoroutines, len(tools))
	}
}

func TestMCPService_ParallelInitialization(t *testing.T) {
	// Test that the Initialize method can handle parallel execution
	// This is a basic test to ensure no race conditions in the initialization logic

	// Create a temporary file for MCP servers
	tmpFile, err := os.CreateTemp("", "mcp-servers-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write empty MCP servers config
	mcpConfig := `{"mcpServers": {}}`
	if _, err := tmpFile.WriteString(mcpConfig); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	tmpFile.Close()

	cfg := &config.Config{
		MCP: config.MCPConfig{
			ServersFile: tmpFile.Name(),
		},
	}
	service := NewMCPService(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Since we have an empty servers config, this will complete quickly
	// but it tests the parallel execution structure
	start := time.Now()
	err = service.Initialize(ctx)
	elapsed := time.Since(start)

	// Should not return an error with empty servers config
	if err != nil {
		t.Errorf("Initialize returned error: %v", err)
	}

	// Should complete quickly since there are no servers to connect to
	t.Logf("Initialization took: %v", elapsed)

	if elapsed > 2*time.Second {
		t.Errorf("Initialization took too long: %v", elapsed)
	}
}

func TestMCPService_ParallelConnectionBenefits(t *testing.T) {
	// Create a temporary file with multiple MCP servers
	tmpFile, err := os.CreateTemp("", "mcp-servers-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write MCP servers config with multiple servers
	// Note: These will fail to connect, but that's OK for testing parallel execution
	mcpConfig := `{
		"mcpServers": {
			"server1": {
				"description": "Test Server 1",
				"command": "nonexistent-command-1",
				"args": ["arg1"],
				"timeout": 1,
				"enabled": true
			},
			"server2": {
				"description": "Test Server 2", 
				"command": "nonexistent-command-2",
				"args": ["arg2"],
				"timeout": 1,
				"enabled": true
			},
			"server3": {
				"description": "Test Server 3",
				"command": "nonexistent-command-3", 
				"args": ["arg3"],
				"timeout": 1,
				"enabled": true
			}
		}
	}`
	if _, err := tmpFile.WriteString(mcpConfig); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	tmpFile.Close()

	cfg := &config.Config{
		MCP: config.MCPConfig{
			ServersFile: tmpFile.Name(),
		},
	}
	service := NewMCPService(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test parallel initialization with multiple servers
	start := time.Now()
	err = service.Initialize(ctx)
	elapsed := time.Since(start)

	// Should not return an error even if connections fail
	if err != nil {
		t.Errorf("Initialize returned error: %v", err)
	}

	// With parallel execution, this should complete in roughly the timeout of one server (1s)
	// plus some overhead, rather than 3x the timeout (3s) for sequential execution
	t.Logf("Parallel initialization of 3 servers took: %v", elapsed)

	// Allow some overhead, but it should be significantly faster than sequential (3+ seconds)
	if elapsed > 3*time.Second {
		t.Errorf("Parallel initialization took too long: %v (expected < 3s)", elapsed)
	}

	// Verify servers were loaded even though connections failed
	service.mu.RLock()
	serverCount := len(service.servers)
	service.mu.RUnlock()

	if serverCount != 3 {
		t.Errorf("Expected 3 servers to be loaded, got %d", serverCount)
	}
}

func TestRefreshTools_EmptyServers(t *testing.T) {
	cfg := &config.Config{}
	service := NewMCPService(cfg)

	// Test RefreshTools with no configured servers
	err := service.RefreshTools()
	assert.NoError(t, err)

	// Should have no tools
	tools := service.GetTools()
	assert.Len(t, tools, 0)
}

func TestRefreshTools_Parallel_Structure(t *testing.T) {
	cfg := &config.Config{}
	service := NewMCPService(cfg)

	// Test that the parallel structure works even with no servers
	// This verifies the goroutine and WaitGroup logic
	start := time.Now()
	err := service.RefreshTools()
	duration := time.Since(start)

	assert.NoError(t, err)

	// Should complete very quickly with no servers
	assert.Less(t, duration, 100*time.Millisecond, "RefreshTools should complete quickly with no servers")

	// Verify tools map is properly initialized
	tools := service.GetTools()
	assert.NotNil(t, tools)
	assert.Len(t, tools, 0)
}

func TestRefreshTools_WithReconnection(t *testing.T) {
	// Create a temporary file with MCP servers config
	tmpFile, err := os.CreateTemp("", "mcp-servers-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write MCP servers config with servers that will fail to connect
	// This tests the reconnection logic without requiring actual MCP servers
	mcpConfig := `{
		"mcpServers": {
			"test-server1": {
				"description": "Test Server 1",
				"command": "nonexistent-command-1",
				"args": ["arg1"],
				"timeout": 1,
				"enabled": true
			},
			"test-server2": {
				"description": "Test Server 2",
				"command": "nonexistent-command-2", 
				"args": ["arg2"],
				"timeout": 1,
				"enabled": true
			}
		}
	}`
	if _, err := tmpFile.WriteString(mcpConfig); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	tmpFile.Close()

	cfg := &config.Config{
		MCP: config.MCPConfig{
			ServersFile: tmpFile.Name(),
		},
	}
	service := NewMCPService(cfg)

	// Initialize to load server configurations
	ctx := context.Background()
	service.Initialize(ctx)

	// Verify servers were loaded
	service.mu.RLock()
	serverCount := len(service.servers)
	service.mu.RUnlock()
	assert.Equal(t, 2, serverCount, "Should have loaded 2 server configurations")

	// Test RefreshTools with reconnection
	start := time.Now()
	err = service.RefreshTools()
	duration := time.Since(start)

	assert.NoError(t, err, "RefreshTools should not return error even if connections fail")

	// Should complete within reasonable time (servers will fail to connect quickly)
	assert.Less(t, duration, 10*time.Second, "RefreshTools should complete within timeout")

	// Verify tools map is cleared and initialized (even though connections failed)
	tools := service.GetTools()
	assert.NotNil(t, tools)
	// Should be empty since connections failed
	assert.Len(t, tools, 0)

	// Verify clients map is cleared (since connections failed)
	service.mu.RLock()
	clientCount := len(service.clients)
	service.mu.RUnlock()
	assert.Equal(t, 0, clientCount, "Should have no connected clients after failed reconnection attempts")
}
