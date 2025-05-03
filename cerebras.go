package main

import (
	"context"
	"fmt"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// See https://inference-docs.cerebras.ai/introduction
// Our free tier supports a context length of 8,192 tokens
// For all supported models, we also offer context lengths up to 128K upon request
const cerebrasDefaultModel = "llama-4-scout-17b-16e-instruct"

func CerebrasGenChatCompletionMock() *openai.ChatCompletion {
	return &openai.ChatCompletion{
		ID:      "cerebras-mock-123",
		Object:  "chat.completion",
		Created: 1677652288, // Example timestamp
		Model:   cerebrasDefaultModel,
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
		Model: cerebrasDefaultModel,
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

// CerebrasWrapper is the top-level function for Cerebras
func CerebrasWrapper(promptText string, mock bool) string {
	// Note that list models toggle is not used and probably should be
	return fmt.Sprintf("# Cerebras\n%s\n\n", CerebrasMiddleWrapper(promptText, mock))
}
