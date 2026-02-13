package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

const llmProxyURLEnvVar = "LLM_PROXY_URL"

// providerProxyPrefix maps each provider to its proxy model prefix
var providerProxyPrefix = map[Provider]string{
	ProviderChatGPT:    "openai",
	ProviderClaude:     "anthropic",
	ProviderGemini:     "gemini",
	ProviderCerebras:   "cerebras",
	ProviderPerplexity: "perplexity",
}

// proxyModelName transforms a provider + model into a proxy-prefixed model name
func proxyModelName(provider Provider, model string) string {
	prefix, ok := providerProxyPrefix[provider]
	if !ok {
		return model
	}
	return prefix + ":" + model
}

// CheckLLMProxyHealth checks if LLM_PROXY_URL is set and the proxy is healthy.
// Returns true only if the env var is set and the proxy responds 200 on /health.
func CheckLLMProxyHealth() bool {
	url := os.Getenv(llmProxyURLEnvVar)
	if url == "" {
		return false
	}

	baseURL := strings.TrimSuffix(url, "/v1")
	client := http.Client{
		Timeout: 500 * time.Millisecond,
	}

	resp, err := client.Get(baseURL + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func getLLMProxyURL() string {
	url := os.Getenv(llmProxyURLEnvVar)
	if url == "" {
		return "http://localhost:8000/v1"
	}
	return url
}

func LLMProxyGenChatCompletionMock() *openai.ChatCompletion {
	return &openai.ChatCompletion{
		ID:      "llmproxy-mock-123",
		Object:  "chat.completion",
		Created: 1677652288,
		Model:   "openai:gpt-5.2",
		Choices: []openai.ChatCompletionChoice{
			{
				Index: 0,
				Message: openai.ChatCompletionMessage{
					Role:    "assistant",
					Content: "This is a mocked LLM Proxy response.",
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

func LLMProxyLowerWrapper(promptText string, mock bool, model string) *openai.ChatCompletion {
	if mock {
		return LLMProxyGenChatCompletionMock()
	}

	client := openai.NewClient(option.WithAPIKey("not-needed"), option.WithBaseURL(getLLMProxyURL()))
	chatCompletion, err := client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(promptText),
		},
		Model: model,
	})

	if err != nil {
		Fatalf("Some error %s", err)
	}

	return chatCompletion
}

// llmproxyChat handles interactive chat with conversation history
func llmproxyChat(history []ChatMessage, userInput string, model string) (string, error) {
	client := openai.NewClient(option.WithAPIKey("not-needed"), option.WithBaseURL(getLLMProxyURL()))

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

// LLMProxyWrapperForProvider wraps a call through the proxy for a specific provider
func LLMProxyWrapperForProvider(providerName string, promptText string, model string, mock bool, logToJsonl bool, quietMode bool) string {
	fromTime := time.Now()

	c := LLMProxyLowerWrapper(promptText, mock, model)

	duration := time.Since(fromTime)

	if len(c.Choices) == 0 {
		Fatalf("%s (via proxy) returned no choices", providerName)
	}

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

	fmtStr := "Model: %s, %d tokens used, finished due to: %s, duration: %.3f seconds"

	status := fmt.Sprintf(fmtStr, c.Model, c.Usage.TotalTokens, c.Choices[0].FinishReason, duration.Seconds())

	return fmt.Sprintf("# %s (via proxy)\n\n%s\n\n%s\n\n", providerName, status, c.Choices[0].Message.Content)
}
