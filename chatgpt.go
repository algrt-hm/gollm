package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

func ListOpenAIModels() string {
	client := openai.NewClient(option.WithAPIKey(GetChatGPTAPIKeyOrBail()))

	// context.TODO() is appropriate for simple short-lived API calls
	// where e.g. no timeout is needed
	resp, err := client.Models.List(context.TODO())

	if err != nil {
		Fatalf("Error listing models: %v", err)
	}

	// Sort models by creation timestamp (descending)
	sort.SliceStable(resp.Data, func(i, j int) bool {
		return resp.Data[i].Created > resp.Data[j].Created
	})

	// strings.Builder is an efficient way to build strings incrementally
	// It minimizes memory copying and is more efficient than string concatenation
	// The zero value is ready to use; no initialization needed
	var builder strings.Builder
	builder.WriteString("Available OpenAI Models:\n")
	for _, model := range resp.Data {
		// Convert Unix timestamp to a readable time format
		createdTime := time.Unix(int64(model.Created), 0).Format(time.RFC1123)
		// Only print Owned by: if not "system"
		if model.OwnedBy != "system" {
			builder.WriteString(fmt.Sprintf("- %s: Owned by: %s, Created: %s\n", model.ID, model.OwnedBy, createdTime))
		} else {
			builder.WriteString(fmt.Sprintf("- %s: Created: %s\n", model.ID, createdTime))
		}
	}
	return builder.String()
}

func ChatGPTGenChatCompletionMock() *openai.ChatCompletion {
	return &openai.ChatCompletion{
		ID:      "chatcmpl-mock-123",
		Object:  "chat.completion",
		Created: 1677652288, // Example timestamp
		Model:   openai.ChatModelGPT4o,
		Choices: []openai.ChatCompletionChoice{
			{
				Index: 0,
				Message: openai.ChatCompletionMessage{
					Role:    "assistant",
					Content: "This is a mocked ChatGPT response.",
				},
				FinishReason: "stop",
			},
			{
				Index: 1,
				Message: openai.ChatCompletionMessage{
					Role:    "assistant",
					Content: "This is another mocked ChatGPT response.",
				},
				FinishReason: "stop",
			},
		},
		Usage: openai.CompletionUsage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}
}

func ChatGPTLowerWrapper(promptText string, mock bool) *openai.ChatCompletion {
	if mock {
		return ChatGPTGenChatCompletionMock()
	}

	client := openai.NewClient(option.WithAPIKey(GetChatGPTAPIKeyOrBail()))
	chatCompletion, err := client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(promptText),
		},
		Model: openai.ChatModelGPT4o,
	})

	if err != nil {
		Fatalf("Some error %s", err)
	}

	return chatCompletion
}

func ChatGPTMiddleWrapper(promptText string, mock bool) string {
	fromTime := time.Now()

	c := ChatGPTLowerWrapper(promptText, mock)

	duration := time.Since(fromTime)

	// Use the finish reason from the first choice as representative
	finishReason := "N/A"
	if len(c.Choices) > 0 {
		finishReason = string(c.Choices[0].FinishReason) // Convert FinishReason type to string
	}

	var contentBuilder strings.Builder
	firstFinishReason := finishReason // Store the first reason for comparison
	for i, choice := range c.Choices {
		if i > 0 {
			contentBuilder.WriteString("\n---\n") // Add separator for multiple choices
		}
		contentBuilder.WriteString(choice.Message.Content)

		// Check if finish reason is different from the first one
		currentFinishReason := string(choice.FinishReason)
		if currentFinishReason != firstFinishReason {
			// Append if different and not already added (to avoid duplicates if many differ)
			if !strings.Contains(finishReason, currentFinishReason) {
				finishReason += ", " + currentFinishReason
			}
		}
	}

	// Update status string *after* the loop in case finishReason was modified
	fmtStr := "Model: %s, %d tokens used, finished due to: %s, duration: %.3f seconds"
	status := fmt.Sprintf(fmtStr, c.Model, c.Usage.TotalTokens, finishReason, duration.Seconds())

	return fmt.Sprintf("\n%s\n\n%s", status, contentBuilder.String())
}

func ChatGPTWrapper(promptText string, mock bool, logToJsonl bool, quietMode bool) string {
	fromTime := time.Now()

	c := ChatGPTLowerWrapper(promptText, mock)

	duration := time.Since(fromTime)

	// Use the finish reason from the first choice as representative
	finishReason := "N/A"
	if len(c.Choices) > 0 {
		finishReason = string(c.Choices[0].FinishReason) // Convert FinishReason type to string
	}

	var contentBuilder strings.Builder
	firstFinishReason := finishReason // Store the first reason for comparison
	for i, choice := range c.Choices {
		if i > 0 {
			contentBuilder.WriteString("\n---\n") // Add separator for multiple choices
		}
		contentBuilder.WriteString(choice.Message.Content)

		// Check if finish reason is different from the first one
		currentFinishReason := string(choice.FinishReason)
		if currentFinishReason != firstFinishReason {
			// Append if different and not already added (to avoid duplicates if many differ)
			if !strings.Contains(finishReason, currentFinishReason) {
				finishReason += ", " + currentFinishReason
			}
		}
	}

	// Log successful model call only if logging is enabled
	if logToJsonl {
		logEntry := LogEntry{
			ModelName:     c.Model,
			TotalTokens:   int(c.Usage.TotalTokens),
			Duration:      duration.Seconds(),
			StopReason:    finishReason,
			PromptText:    promptText,
			ModelResponse: contentBuilder.String(),
			Timestamp:     time.Now(),
		}
		if err := WriteLogEntry(logEntry); err != nil {
			// Log error but don't fail the request
			fmt.Fprintf(os.Stderr, "Failed to write log entry: %v\n", err)
		}
	}

	if quietMode {
		return contentBuilder.String()
	}

	// Update status string *after* the loop in case finishReason was modified
	fmtStr := "Model: %s, %d tokens used, finished due to: %s, duration: %.3f seconds"
	status := fmt.Sprintf(fmtStr, c.Model, c.Usage.TotalTokens, finishReason, duration.Seconds())

	return fmt.Sprintf("# ChatGPT\n\n%s\n\n%s\n\n", status, contentBuilder.String())
}
