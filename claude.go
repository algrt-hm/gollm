package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// ListAnthropicModels will list Anthropic models which are available
func ListAnthropicModels() string {
	var builder strings.Builder

	// --- Get API Key ---
	apiKey := GetClaudeAPIKeyOrBail()

	// --- Set up the Anthropic client ---
	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	// List models using the API
	ctx := context.Background()
	resp, err := client.Models.List(ctx, anthropic.ModelListParams{})

	if err != nil {
		Fatalf("Error listing Anthropic models: %v", err)
	}

	builder.WriteString("Available Anthropic Models:\n")
	for _, model := range resp.Data {
		// Format output with model ID and creation date if available
		builder.WriteString(fmt.Sprintf("- %s", model.ID))
		if model.DisplayName != "" {
			builder.WriteString(fmt.Sprintf(" (%s)", model.DisplayName))
		}
		if model.CreatedAt.IsZero() == false {
			builder.WriteString(fmt.Sprintf(", Created: %s", model.CreatedAt.Format("2006-01-02")))
		}
		builder.WriteString("\n")
	}

	return builder.String()
}

func ClaudeGenChatCompletionMock() *anthropic.Message {
	// Create a mock message manually
	mock := &anthropic.Message{
		ID:         "msg-mock-123",
		Model:      anthropic.Model(DefaultModels.Claude),
		StopReason: anthropic.StopReasonEndTurn,
		Usage: anthropic.Usage{
			InputTokens:  10,
			OutputTokens: 5,
		},
	}
	// Note: We can't easily set Content here due to union types
	// This is sufficient for basic mocking
	return mock
}

func ClaudeLowerWrapper(promptText string, mock bool) *anthropic.Message {
	if mock {
		return ClaudeGenChatCompletionMock()
	}

	client := anthropic.NewClient(option.WithAPIKey(GetClaudeAPIKeyOrBail()))
	message, err := client.Messages.New(context.TODO(), anthropic.MessageNewParams{
		Model: anthropic.Model(DefaultModels.Claude),
		// Different models have different maximum values for this parameter. See
		// [models](https://docs.anthropic.com/en/docs/models-overview) for details.
		//
		// as at 25 October 2025 it's 64k for Sonnet 4.5
		MaxTokens: 64000,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(promptText)),
		},
	})

	if err != nil {
		Fatalf("Some error %s", err)
	}

	return message
}

func ClaudeMiddleWrapper(promptText string, mock bool) string {
	fromTime := time.Now()

	m := ClaudeLowerWrapper(promptText, mock)

	duration := time.Since(fromTime)

	// Extract text content from the message
	var content string
	for _, block := range m.Content {
		if block.Type == "text" {
			content = block.Text
		}
	}

	totalTokens := m.Usage.InputTokens + m.Usage.OutputTokens
	fmtStr := "Model: %s, %d tokens used, finished due to: %s, duration: %.3f seconds"
	status := fmt.Sprintf(fmtStr, m.Model, totalTokens, m.StopReason, duration.Seconds())

	return fmt.Sprintf("\n%s\n\n%s", status, content)
}

// ClaudeWrapper is the top-level function for Claude
func ClaudeWrapper(promptText string, mock bool, logToJsonl bool, quietMode bool) string {
	fromTime := time.Now()

	m := ClaudeLowerWrapper(promptText, mock)

	duration := time.Since(fromTime)

	// Extract text content from the message
	var content string
	for _, block := range m.Content {
		if block.Type == "text" {
			content = block.Text
		}
	}

	totalTokens := m.Usage.InputTokens + m.Usage.OutputTokens

	// Log successful model call only if logging is enabled
	if logToJsonl {
		logEntry := LogEntry{
			ModelName:     string(m.Model),
			TotalTokens:   int(totalTokens),
			Duration:      duration.Seconds(),
			StopReason:    string(m.StopReason),
			PromptText:    promptText,
			ModelResponse: content,
			Timestamp:     time.Now(),
		}
		if err := WriteLogEntry(logEntry); err != nil {
			// Log error but don't fail the request
			fmt.Fprintf(os.Stderr, "Failed to write log entry: %v\n", err)
		}
	}

	if quietMode {
		return content
	}

	fmtStr := "Model: %s, %d tokens used, finished due to: %s, duration: %.3f seconds"
	status := fmt.Sprintf(fmtStr, m.Model, totalTokens, m.StopReason, duration.Seconds())

	return fmt.Sprintf("# Claude\n\n%s\n\n%s\n\n", status, content)
}
