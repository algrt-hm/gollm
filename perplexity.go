package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"time"
)

type UsageStats struct {
	PromptTokens      int    `json:"prompt_tokens"`
	CompletionTokens  int    `json:"completion_tokens"`
	TotalTokens       int    `json:"total_tokens"`
	SearchContextSize string `json:"search_context_size"` // Added based on JSON
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Delta struct { // Added based on JSON
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Choice struct {
	Index        int     `json:"index"`
	FinishReason string  `json:"finish_reason"`
	Message      Message `json:"message"`
	Delta        Delta   `json:"delta"` // Added based on JSON
}

type PerplexityResponse struct {
	ID        string     `json:"id"`
	Model     string     `json:"model"`
	Created   int64      `json:"created"` // Use int64 for timestamps
	Usage     UsageStats `json:"usage"`
	Citations []string   `json:"citations"` // Added based on JSON
	Object    string     `json:"object"`
	Choices   []Choice   `json:"choices"`
}

type WebSearchOptions struct {
	SearchContextSize string `json:"search_context_size"`
}

type PerplexityRequest struct {
	Model                  string           `json:"model"`
	Messages               []Message        `json:"messages"`
	MaxTokens              int              `json:"max_tokens"`
	Temperature            float64          `json:"temperature"`
	TopP                   float64          `json:"top_p"`
	SearchDomainFilter     []string         `json:"search_domain_filter"`
	ReturnImages           bool             `json:"return_images"`
	ReturnRelatedQuestions bool             `json:"return_related_questions"`
	SearchRecencyFilter    string           `json:"search_recency_filter"`
	TopK                   int              `json:"top_k"`
	Stream                 bool             `json:"stream"`
	PresencePenalty        float64          `json:"presence_penalty"`
	FrequencyPenalty       float64          `json:"frequency_penalty"`
	WebSearchOptions       WebSearchOptions `json:"web_search_options"`
}

// ParsePerplexityResponse parses a Perplexity response and returns a ModelResponse
func ParsePerplexityResponse(result string) (ModelResponse, error) {
	var response PerplexityResponse
	err := json.Unmarshal([]byte(result), &response)
	if err != nil {
		return ModelResponse{}, fmt.Errorf("failed to unmarshal JSON response: %w\nResponse was: %s", err, result)
	}

	if len(response.Choices) == 0 {
		return ModelResponse{}, fmt.Errorf("response contained no choices.\nResponse was: %s", result)
	}

	model := response.Model
	totalTokens := response.Usage.TotalTokens
	citations := response.Citations
	content := response.Choices[0].Message.Content
	finishReason := response.Choices[0].FinishReason

	return ModelResponse{
		Model:        model,
		TotalTokens:  totalTokens,
		Citations:    citations,
		Content:      content,
		FinishReason: finishReason,
	}, nil
}

func FmtModelResponse(response ModelResponse, duration time.Duration, quietMode bool) string {
	var out string

	if !quietMode {
		out += fmt.Sprintf("Model: %s, %d tokens used, finished due to: %s, duration: %.3f seconds\n", response.Model, response.TotalTokens, response.FinishReason, duration.Seconds())
	}

	// Replace e.g. [1] with [^1] in response.Content using a regex
	re := regexp.MustCompile(`\[(\d+)\]`)
	formattedContent := re.ReplaceAllString(response.Content, "[^$1]")

	out += fmt.Sprintf("\n%s\n\n", formattedContent)

	// Markdown citations
	for idx, citation := range response.Citations {
		out += fmt.Sprintf("[^%d]: %s\n", idx+1, citation)
	}

	out += "\n\nCitations:\n\n"

	// Non-markdown citations
	for idx, citation := range response.Citations {
		out += fmt.Sprintf("%d. %s\n", idx+1, citation)
	}

	if quietMode {
		return out + "\n"
	} // implied else

	return "# Perplexity\n" + out + "\n"
}

// CallPerplexityAPI calls the Perplexity API
func CallPerplexityAPI(promptText string, mock bool) (string, time.Duration, error) {
	// Start the timer
	startTime := time.Now()

	if mock {
		// This is our response to
		// promptText = "Please tell me about Perplexity"
		return `{
  "id": "a83283d7-4307-4c36-850f-56b648ae90a1",
  "model": "sonar-pro",
  "created": 1745486154,
  "usage": {
    "prompt_tokens": 12,
    "completion_tokens": 123,
    "total_tokens": 135,
    "search_context_size": "high"
  },
  "citations": [
    "https://www.youtube.com/watch?v=CxMVYwGO7Ec",
    "https://www.perplexity.ai/discover",
    "https://www.youtube.com/watch?v=O1UTAiigrx4",
    "https://www.perplexity.ai/hub/blog/choice-is-the-remedy",
    "https://www.fahimai.com/perplexity-ai",
    "https://www.appypieautomate.ai/blog/perplexity-ai-vs-chatgpt",
    "https://www.adexchanger.com/commerce/perplexity-takes-its-ai-search-engine-out-on-a-shopping-trip/"
  ],
  "object": "chat.completion",
  "choices": [
    {
      "index": 0,
      "finish_reason": "length",
      "message": {
        "role": "assistant",
        "content": "## What is Perplexity?\n\nPerplexity is an AI-powered answer engine designed to provide users with accurate, trusted, and real-time answers to any question. Unlike traditional search engines that return a list of links, Perplexity synthesizes information from the web and delivers direct answers with clear citations, making it easier for users to verify sources and get reliable information quickly[2][5][6].\n\n## Key Features\n\nDirect Answers with Citations\n- Perplexity uses advanced natural language processing to understand queries in plain language and responds with concise answers directly sourced from reputable web content. Each answer includes citations"
      },
      "delta": {
        "role": "assistant",
        "content": ""
      }
    }
  ]
}`, time.Since(startTime), nil
	}

	key := getPerplexityAPIKey()
	url := "https://api.perplexity.ai/chat/completions"

	requestPayload := PerplexityRequest{
		Model: DefaultModels.Perplexity,
		Messages: []Message{
			{Role: "system", Content: "Be precise and concise."},
			{Role: "user", Content: promptText},
		},
		MaxTokens:              4000,
		Temperature:            0.2,
		TopP:                   0.9,
		SearchDomainFilter:     []string{},
		ReturnImages:           false,
		ReturnRelatedQuestions: false,
		SearchRecencyFilter:    "month",
		TopK:                   0,
		Stream:                 false,
		PresencePenalty:        0,
		FrequencyPenalty:       1,
		WebSearchOptions: WebSearchOptions{
			SearchContextSize: "high",
		},
	}

	payloadBytes, err := json.Marshal(requestPayload)
	if err != nil {
		return "", time.Since(startTime), fmt.Errorf("failed to marshal request payload: %w", err)
	}

	payload := bytes.NewReader(payloadBytes)

	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		return "", time.Since(startTime), fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", key))
	req.Header.Add("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", time.Since(startTime), fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", time.Since(startTime), fmt.Errorf("failed to read response body: %w", err)
	}

	if res.StatusCode != 200 {
		return "", time.Since(startTime), fmt.Errorf("API error (status %d): %s", res.StatusCode, string(body))
	}

	return string(body), time.Since(startTime), nil
}

// perplexityChat handles interactive chat with conversation history
func perplexityChat(history []ChatMessage, userInput string, model string, showCitations bool) (string, error) {
	key := os.Getenv("PERPLEXITY_API_KEY")
	if key == "" {
		return "", fmt.Errorf("PERPLEXITY_API_KEY not set")
	}
	url := "https://api.perplexity.ai/chat/completions"

	// Build messages array from history
	var messages []Message
	messages = append(messages, Message{Role: "system", Content: "Be precise and concise."})
	for _, msg := range history {
		messages = append(messages, Message{Role: msg.Role, Content: msg.Content})
	}
	messages = append(messages, Message{Role: "user", Content: userInput})

	requestPayload := PerplexityRequest{
		Model:                  model,
		Messages:               messages,
		MaxTokens:              4000,
		Temperature:            0.2,
		TopP:                   0.9,
		SearchDomainFilter:     []string{},
		ReturnImages:           false,
		ReturnRelatedQuestions: false,
		SearchRecencyFilter:    "month",
		TopK:                   0,
		Stream:                 false,
		PresencePenalty:        0,
		FrequencyPenalty:       1,
		WebSearchOptions: WebSearchOptions{
			SearchContextSize: "high",
		},
	}

	payloadBytes, err := json.Marshal(requestPayload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", key))
	req.Header.Add("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		body, _ := io.ReadAll(res.Body)
		return "", fmt.Errorf("API error (status %d): %s", res.StatusCode, string(body))
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var response PerplexityResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response from model")
	}

	content := response.Choices[0].Message.Content

	// Format citations like non-interactive mode (FmtModelResponse)
	if showCitations {
		content = appendCitations(content, response.Citations)
	}

	return content, nil
}

// appendCitations rewrites [n] markers as markdown footnotes and appends the
// citation list, matching the non-interactive output format (FmtModelResponse)
func appendCitations(content string, citations []string) string {
	if len(citations) == 0 {
		return content
	}

	re := regexp.MustCompile(`\[(\d+)\]`)
	content = re.ReplaceAllString(content, "[^$1]")

	content += "\n\n"
	for idx, citation := range citations {
		content += fmt.Sprintf("[^%d]: %s\n", idx+1, citation)
	}

	content += "\n\nCitations:\n\n"
	for idx, citation := range citations {
		content += fmt.Sprintf("%d. %s\n", idx+1, citation)
	}

	return content
}

func PerplexityWrapper(promptText string, mock bool, logToJsonl bool, quietMode bool) (string, error) {
	result, duration, err := CallPerplexityAPI(promptText, mock)
	if err != nil {
		return "", err
	}

	modelResponse, err := ParsePerplexityResponse(result)
	if err != nil {
		return "", err
	}

	// Log successful model call only if logging is enabled
	if logToJsonl {
		logEntry := LogEntry{
			ModelName:     modelResponse.Model,
			TotalTokens:   modelResponse.TotalTokens,
			Duration:      duration.Seconds(),
			StopReason:    modelResponse.FinishReason,
			PromptText:    promptText,
			ModelResponse: modelResponse.Content,
			Timestamp:     time.Now(),
		}
		if err := WriteLogEntry(logEntry); err != nil {
			// Log error but don't fail the request
			fmt.Fprintf(os.Stderr, "Failed to write log entry: %v\n", err)
		}
	}

	return FmtModelResponse(modelResponse, duration, quietMode), nil
}
