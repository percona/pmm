package services

import (
	"strings"
	"testing"
	"time"

	"github.com/percona/pmm/aichat-backend/internal/models"
)

func TestPrepareMessagesWithSystemPrompt_TimeContext(t *testing.T) {
	// Create a mock chat service
	chatService := &ChatService{
		systemPrompt: "You are an AI assistant for PMM.",
	}

	// Create test messages
	messages := []*models.Message{
		{
			ID:      "user_1",
			Role:    "user",
			Content: "What is the current time?",
		},
	}

	// Call the method
	preparedMessages := chatService.prepareMessagesWithSystemPrompt(messages)

	// Verify system message was added
	if len(preparedMessages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(preparedMessages))
	}

	systemMessage := preparedMessages[0]
	if systemMessage.Role != "system" {
		t.Fatalf("Expected first message to be system role, got %s", systemMessage.Role)
	}

	// Verify time context is included
	content := systemMessage.Content

	// Debug: Print the enhanced system prompt (uncomment to see the full content)
	// t.Logf("Enhanced System Prompt:\n%s", content)

	if !strings.Contains(content, "CURRENT CONTEXT:") {
		t.Error("System prompt should contain CURRENT CONTEXT section")
	}
	if !strings.Contains(content, "Current time:") {
		t.Error("System prompt should contain current time")
	}
	if !strings.Contains(content, "12-hour period by default") {
		t.Error("System prompt should mention 12-hour default period")
	}
	if !strings.Contains(content, "RFC3339 format") {
		t.Error("System prompt should mention RFC3339 format")
	}

	// Verify original system prompt is preserved
	if !strings.Contains(content, "You are an AI assistant for PMM.") {
		t.Error("Original system prompt should be preserved")
	}

	// Verify time format is correct (should be parseable)
	now := time.Now()
	expectedTimePrefix := now.UTC().Format("2006-01-02")
	if !strings.Contains(content, expectedTimePrefix) {
		t.Errorf("System prompt should contain today's date %s", expectedTimePrefix)
	}
}

func TestPrepareMessagesWithSystemPrompt_NoSystemPrompt(t *testing.T) {
	// Create a mock chat service with no system prompt
	chatService := &ChatService{
		systemPrompt: "",
	}

	// Create test messages
	messages := []*models.Message{
		{
			ID:      "user_1",
			Role:    "user",
			Content: "Test message",
		},
	}

	// Call the method
	preparedMessages := chatService.prepareMessagesWithSystemPrompt(messages)

	// Verify no system message was added
	if len(preparedMessages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(preparedMessages))
	}

	if preparedMessages[0].Role != "user" {
		t.Fatalf("Expected message to be user role, got %s", preparedMessages[0].Role)
	}
}

func TestPrepareMessagesWithSystemPrompt_ExistingSystemMessage(t *testing.T) {
	// Create a mock chat service
	chatService := &ChatService{
		systemPrompt: "You are an AI assistant for PMM.",
	}

	// Create test messages with existing system message
	messages := []*models.Message{
		{
			ID:      "system_1",
			Role:    "system",
			Content: "Existing system message",
		},
		{
			ID:      "user_1",
			Role:    "user",
			Content: "Test message",
		},
	}

	// Call the method
	preparedMessages := chatService.prepareMessagesWithSystemPrompt(messages)

	// Verify no additional system message was added
	if len(preparedMessages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(preparedMessages))
	}

	if preparedMessages[0].Content != "Existing system message" {
		t.Error("Existing system message should be preserved")
	}
}

func TestSetSystemPrompt(t *testing.T) {
	// Create a mock chat service
	chatService := &ChatService{
		systemPrompt: "",
	}

	// Set system prompt
	testPrompt := "Test system prompt"
	chatService.SetSystemPrompt(testPrompt)

	// Verify system prompt was set
	if chatService.systemPrompt != testPrompt {
		t.Errorf("Expected system prompt to be %q, got %q", testPrompt, chatService.systemPrompt)
	}
}
