package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// DeepSeek API uses an OpenAI-compatible format.
// See https://api-docs.deepseek.com/
const deepseekBaseURL = "https://api.deepseek.com"

func DeepseekGenChatCompletionMock() *openai.ChatCompletion {
	return &openai.ChatCompletion{
		ID:      "deepseek-mock-123",
		Object:  "chat.completion",
		Created: 1677652288,
		Model:   DefaultModels.Deepseek,
		Choices: []openai.ChatCompletionChoice{
			{
				Index: 0,
				Message: openai.ChatCompletionMessage{
					Role:    "assistant",
					Content: "This is a mocked DeepSeek response.",
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

func DeepseekLowerWrapper(promptText string, mock bool) *openai.ChatCompletion {
	if mock {
		return DeepseekGenChatCompletionMock()
	}

	client := openai.NewClient(option.WithAPIKey(GetDeepseekAPIKeyOrBail()), option.WithBaseURL(deepseekBaseURL))
	chatCompletion, err := client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(promptText),
		},
		Model: DefaultModels.Deepseek,
	})

	if err != nil {
		Fatalf("Some error %s", err)
	}

	return chatCompletion
}

// deepseekChat handles interactive chat with conversation history
func deepseekChat(history []ChatMessage, userInput string, model string) (string, error) {
	client := openai.NewClient(option.WithAPIKey(GetDeepseekAPIKeyOrBail()), option.WithBaseURL(deepseekBaseURL))

	var messages []openai.ChatCompletionMessageParamUnion
	for _, msg := range history {
		if msg.Role == "user" {
			messages = append(messages, openai.UserMessage(msg.Content))
		} else {
			messages = append(messages, openai.AssistantMessage(msg.Content))
		}
	}
	messages = append(messages, openai.UserMessage(userInput))

	chatCompletion, err := client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Messages: messages,
		Model:    model,
	})

	if err != nil {
		return "", fmt.Errorf("API error: %w", err)
	}

	if len(chatCompletion.Choices) == 0 {
		return "", fmt.Errorf("no response from model")
	}

	return chatCompletion.Choices[0].Message.Content, nil
}

// DeepseekWrapper is the top-level function for Deepseek
func DeepseekWrapper(promptText string, mock bool, logToJsonl bool, quietMode bool) string {
	fromTime := time.Now()

	c := DeepseekLowerWrapper(promptText, mock)

	duration := time.Since(fromTime)

	if len(c.Choices) == 0 {
		Fatalf("DeepSeek response contained no choices")
	}

	if logToJsonl {
		logEntry := LogEntry{
			ModelName:     c.Model,
			TotalTokens:   int(c.Usage.TotalTokens),
			Duration:      duration.Seconds(),
			StopReason:    c.Choices[0].FinishReason,
			PromptText:    promptText,
			ModelResponse: c.Choices[0].Message.Content,
			Timestamp:     time.Now(),
		}
		if err := WriteLogEntry(logEntry); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write log entry: %v\n", err)
		}
	}

	if quietMode {
		return c.Choices[0].Message.Content
	}

	fmtStr := "Model: %s, %d tokens used, finished due to: %s, duration: %.3f seconds"

	status := fmt.Sprintf(fmtStr, c.Model, c.Usage.TotalTokens, c.Choices[0].FinishReason, duration.Seconds())

	return fmt.Sprintf("# DeepSeek\n\n%s\n\n%s\n\n", status, c.Choices[0].Message.Content)
}
