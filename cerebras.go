package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

func CerebrasGenChatCompletionMock() *openai.ChatCompletion {
	return &openai.ChatCompletion{
		ID:      "cerebras-mock-123",
		Object:  "chat.completion",
		Created: 1677652288, // Example timestamp
		Model:   DefaultModels.Cerebras,
		Choices: []openai.ChatCompletionChoice{
			{
				Index: 0,
				Message: openai.ChatCompletionMessage{
					Role:    "assistant",
					Content: "This is a mocked Cerebras response.",
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

func CerebrasLowerWrapper(promptText string, mock bool) *openai.ChatCompletion {
	if mock {
		return CerebrasGenChatCompletionMock()
	}

	/*
		Text Completions
		The following fields are currently not supported and will result in a 400 error if they are supplied:

		- frequency_penalty
		- logit_bias
		- presence_penalty
		- parallel_tool_calls
		- service_tier

		https://inference-docs.cerebras.ai/resources/openai
	*/

	client := openai.NewClient(option.WithAPIKey(GetCerebrasAPIKeyOrBail()), option.WithBaseURL("https://api.cerebras.ai/v1"))
	chatCompletion, err := client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(promptText),
		},
		Model: DefaultModels.Cerebras,
	})

	if err != nil {
		Fatalf("Some error %s", err)
	}

	return chatCompletion
}

func CerebrasMiddleWrapper(promptText string, mock bool) string {
	fromTime := time.Now()

	c := CerebrasLowerWrapper(promptText, mock)

	duration := time.Since(fromTime)

	// Model: sonar-pro, 135 tokens used, finished due to: length, duration: 0.000 seconds
	fmtStr := "Model: %s, %d tokens used, finished due to: %s, duration: %.3f seconds"

	status := fmt.Sprintf(fmtStr, c.Model, c.Usage.TotalTokens, c.Choices[0].FinishReason, duration.Seconds())

	// TODO: need to loop through choices per the Gemini example in case we get more than one back
	return fmt.Sprintf("\n%s\n\n%s", status, c.Choices[0].Message.Content)
}

// cerebrasChat handles interactive chat with conversation history
func cerebrasChat(history []ChatMessage, userInput string, model string) (string, error) {
	client := openai.NewClient(option.WithAPIKey(GetCerebrasAPIKeyOrBail()), option.WithBaseURL("https://api.cerebras.ai/v1"))

	// Build messages array from history
	var messages []openai.ChatCompletionMessageParamUnion
	for _, msg := range history {
		if msg.Role == "user" {
			messages = append(messages, openai.UserMessage(msg.Content))
		} else {
			messages = append(messages, openai.AssistantMessage(msg.Content))
		}
	}
	// Add current user message
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

// CerebrasWrapper is the top-level function for Cerebras
func CerebrasWrapper(promptText string, mock bool, logToJsonl bool, quietMode bool) string {
	fromTime := time.Now()

	c := CerebrasLowerWrapper(promptText, mock)

	duration := time.Since(fromTime)

	// Log successful model call only if logging is enabled
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
			// Log error but don't fail the request
			fmt.Fprintf(os.Stderr, "Failed to write log entry: %v\n", err)
		}
	}

	if quietMode {
		return c.Choices[0].Message.Content
	}

	// Model: sonar-pro, 135 tokens used, finished due to: length, duration: 0.000 seconds
	fmtStr := "Model: %s, %d tokens used, finished due to: %s, duration: %.3f seconds"

	status := fmt.Sprintf(fmtStr, c.Model, c.Usage.TotalTokens, c.Choices[0].FinishReason, duration.Seconds())

	// TODO: need to loop through choices per the Gemini example in case we get more than one back
	return fmt.Sprintf("# Cerebras\n\n%s\n\n%s\n\n", status, c.Choices[0].Message.Content)
}
