package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
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

// isValidJSON checks if a string is valid JSON, returning true if it is
func isValidJSON(jsonStr string) bool {
	return json.Valid([]byte(jsonStr))
}

// ParsePerplexityResponse parses a Perplexity response and returns a ModelResponse
func ParsePerplexityResponse(result string) ModelResponse {
	var response PerplexityResponse
	err := json.Unmarshal([]byte(result), &response)
	if err != nil {
		Fatalf("Failed to unmarshal hardcoded mock JSON response: %v\nResponse was: %s", err, result)
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
	}
}

func FmtModelResponse(response ModelResponse, duration time.Duration) string {
	var out string

	out += fmt.Sprintf("Model: %s, %d tokens used, finished due to: %s, duration: %.3f seconds\n", response.Model, response.TotalTokens, response.FinishReason, duration.Seconds())

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

	return "# Perplexity\n" + out + "\n"
}

// CallPerplexityAPI calls the Perplexity API
func CallPerplexityAPI(promptText string, mock bool) (string, time.Duration) {
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
}`, time.Since(startTime)
	}

	perplexityApiKey := "PERPLEXITY_API_KEY"
	key := os.Getenv(perplexityApiKey)
	url := "https://api.perplexity.ai/chat/completions"

	// Optional fields not used:
	// "response_format": {},

	payloadStr := fmt.Sprintf(`{
  "model": "sonar-pro",
  "messages": [
    {
      "role": "system",
      "content": "Be precise and concise."
    },
    {
      "role": "user",
      "content": "%s"
    }
  ],
  "max_tokens": 4000,
  "temperature": 0.2,
  "top_p": 0.9,
  "search_domain_filter": [],
  "return_images": false,
  "return_related_questions": false,
  "search_recency_filter": "month",
  "top_k": 0,
  "stream": false,
  "presence_penalty": 0,
  "frequency_penalty": 1,
  "web_search_options": {
    "search_context_size": "high"
  }
}`, promptText)

	// fmt.Printf(`
	// url: %s
	// payload: %s
	// `, url, payloadStr)

	// Check we're valid
	valid := isValidJSON(payloadStr)

	if !valid {
		Fatalf("JSON not valid")
	}

	payload := strings.NewReader(payloadStr)

	req, _ := http.NewRequest("POST", url, payload)

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", key))
	req.Header.Add("Content-Type", "application/json")

	// Print the request
	// fmt.Printf("%+v", req)

	res, _ := http.DefaultClient.Do(req)

	// Print the response
	// fmt.Printf("%+v", res)

	if res.StatusCode != 200 {
		fmt.Printf("Some issue: %+v", res)
	}

	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	return string(body), time.Since(startTime)
}

func PerplexityWrapper(promptText string, mock bool, logToJsonl bool) string {
	var response PerplexityResponse

	result, duration := CallPerplexityAPI(promptText, mock)

	err := json.Unmarshal([]byte(result), &response)
	if err != nil {
		Fatalf("Failed to unmarshal hardcoded mock JSON response: %v\nResponse was: %s", err, result)
	}

	modelResponse := ParsePerplexityResponse(result)

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

	return FmtModelResponse(modelResponse, duration)
}
