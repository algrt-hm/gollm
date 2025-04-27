package main

import (
	"context"
	"fmt"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

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
		},
		Usage: openai.CompletionUsage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}
}

func ChatGPTLowerWrapper(promptText string, listModelsToggle bool, mock bool) *openai.ChatCompletion {
	if mock {
		return ChatGPTGenChatCompletionMock()
	}

	client := openai.NewClient(option.WithAPIKey(GetChatGPTAPIKey()))
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

func ChatGPTMiddleWrapper(promptText string, listModelsToggle bool, mock bool) string {
	fromTime := time.Now()

	c := ChatGPTLowerWrapper(promptText, listModelsToggle, mock)

	duration := time.Since(fromTime)

	fmtStr := "Model: %s, %d tokens used, finished due to: %s, duration: %.3f seconds"

	status := fmt.Sprintf(fmtStr, c.Model, c.Usage.TotalTokens, c.Choices[0].FinishReason, duration.Seconds())

	// TODO: need to loop through choices per the Gemini example in case we get more than one back
	return fmt.Sprintf("\n%s\n\n%s", status, c.Choices[0].Message.Content)
}

func ChatGPTWrapper(promptText string, listModelsToggle bool, mock bool) string {
	// Note that list models toggle is not used and probably should be
	return fmt.Sprintf("# ChatGPT\n%s\n\n", ChatGPTMiddleWrapper(promptText, listModelsToggle, mock))
}
