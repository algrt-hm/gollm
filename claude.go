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

// ClaudeResponse holds the accumulated response from streaming
type ClaudeResponse struct {
	Content      string
	Model        string
	StopReason   string
	InputTokens  int64
	OutputTokens int64
}

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

func ClaudeGenChatCompletionMock() *ClaudeResponse {
	return &ClaudeResponse{
		Content:      "This is a mocked Claude response.",
		Model:        DefaultModels.Claude,
		StopReason:   "end_turn",
		InputTokens:  10,
		OutputTokens: 5,
	}
}

func ClaudeLowerWrapper(promptText string, mock bool) *ClaudeResponse {
	if mock {
		return ClaudeGenChatCompletionMock()
	}

	client := anthropic.NewClient(option.WithAPIKey(GetClaudeAPIKeyOrBail()))

	// Use streaming API
	stream := client.Messages.NewStreaming(context.TODO(), anthropic.MessageNewParams{
		Model:     anthropic.Model(DefaultModels.Claude),
		MaxTokens: 64000,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(promptText)),
		},
	})
	defer stream.Close()

	response := &ClaudeResponse{
		Model: DefaultModels.Claude,
	}
	var contentBuilder strings.Builder

	// Process streaming events
	for stream.Next() {
		event := stream.Current()

		switch event.Type {
		case "message_start":
			// Initial message with model info
			msg := event.AsMessageStart()
			response.Model = string(msg.Message.Model)
			response.InputTokens = msg.Message.Usage.InputTokens

		case "content_block_delta":
			// Text content delta
			delta := event.AsContentBlockDelta()
			if delta.Delta.Type == "text_delta" {
				contentBuilder.WriteString(delta.Delta.Text)
			}

		case "message_delta":
			// Final message info with stop reason and output tokens
			msgDelta := event.AsMessageDelta()
			response.StopReason = string(msgDelta.Delta.StopReason)
			response.OutputTokens = msgDelta.Usage.OutputTokens
		}
	}

	if err := stream.Err(); err != nil {
		Fatalf("Streaming error: %s", err)
	}

	response.Content = contentBuilder.String()
	return response
}

func ClaudeMiddleWrapper(promptText string, mock bool) string {
	fromTime := time.Now()

	r := ClaudeLowerWrapper(promptText, mock)

	duration := time.Since(fromTime)

	totalTokens := r.InputTokens + r.OutputTokens
	fmtStr := "Model: %s, %d tokens used, finished due to: %s, duration: %.3f seconds"
	status := fmt.Sprintf(fmtStr, r.Model, totalTokens, r.StopReason, duration.Seconds())

	return fmt.Sprintf("\n%s\n\n%s", status, r.Content)
}

// claudeChat handles interactive chat with conversation history
func claudeChat(history []ChatMessage, userInput string) (string, error) {
	client := anthropic.NewClient(option.WithAPIKey(GetClaudeAPIKeyOrBail()))

	// Build messages array from history
	var messages []anthropic.MessageParam
	for _, msg := range history {
		if msg.Role == "user" {
			messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(msg.Content)))
		} else {
			messages = append(messages, anthropic.NewAssistantMessage(anthropic.NewTextBlock(msg.Content)))
		}
	}
	// Add current user message
	messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(userInput)))

	// Use streaming API
	stream := client.Messages.NewStreaming(context.TODO(), anthropic.MessageNewParams{
		Model:     anthropic.Model(DefaultModels.Claude),
		MaxTokens: 8192,
		Messages:  messages,
	})
	defer stream.Close()

	var contentBuilder strings.Builder

	// Process streaming events
	for stream.Next() {
		event := stream.Current()
		if event.Type == "content_block_delta" {
			delta := event.AsContentBlockDelta()
			if delta.Delta.Type == "text_delta" {
				contentBuilder.WriteString(delta.Delta.Text)
			}
		}
	}

	if err := stream.Err(); err != nil {
		return "", fmt.Errorf("streaming error: %w", err)
	}

	return contentBuilder.String(), nil
}

// ClaudeWrapper is the top-level function for Claude
func ClaudeWrapper(promptText string, mock bool, logToJsonl bool, quietMode bool) string {
	fromTime := time.Now()

	r := ClaudeLowerWrapper(promptText, mock)

	duration := time.Since(fromTime)

	totalTokens := r.InputTokens + r.OutputTokens

	// Log successful model call only if logging is enabled
	if logToJsonl {
		logEntry := LogEntry{
			ModelName:     r.Model,
			TotalTokens:   int(totalTokens),
			Duration:      duration.Seconds(),
			StopReason:    r.StopReason,
			PromptText:    promptText,
			ModelResponse: r.Content,
			Timestamp:     time.Now(),
		}
		if err := WriteLogEntry(logEntry); err != nil {
			// Log error but don't fail the request
			fmt.Fprintf(os.Stderr, "Failed to write log entry: %v\n", err)
		}
	}

	if quietMode {
		return r.Content
	}

	fmtStr := "Model: %s, %d tokens used, finished due to: %s, duration: %.3f seconds"
	status := fmt.Sprintf(fmtStr, r.Model, totalTokens, r.StopReason, duration.Seconds())

	return fmt.Sprintf("# Claude\n\n%s\n\n%s\n\n", status, r.Content)
}
