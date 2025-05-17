package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestPerplexityWrapper(t *testing.T) {
	promptText := "Please tell me about Perplexity"

	RenderWithGlamour(PerplexityWrapper(promptText, true, false))
}

func TestCallPerplexityAPIGeminiVersion(t *testing.T) {
	// Set a dummy API key for testing
	os.Setenv("PERPLEXITY_API_KEY", "test-api-key")
	defer os.Unsetenv("PERPLEXITY_API_KEY") // Clean up after test

	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request method and path
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("Expected path /chat/completions, got %s", r.URL.Path)
		}

		// Check Authorization header (optional but good practice)
		authHeader := r.Header.Get("Authorization")
		expectedAuth := "Bearer test-api-key"
		if authHeader != expectedAuth {
			t.Errorf("Expected Authorization header %s, got %s", expectedAuth, authHeader)
		}

		// Send a mock response
		w.WriteHeader(http.StatusOK)
		// Use the more detailed JSON structure
		mockResponse := `{
			"id": "mock-id-123",
			"model": "sonar-pro-test",
			"created": 1700000000,
			"usage": {
				"prompt_tokens": 10,
				"completion_tokens": 20,
				"total_tokens": 30,
				"search_context_size": "low"
			},
			"citations": ["http://example.com"],
			"object": "chat.completion.test",
			"choices": [
				{
					"index": 0,
					"finish_reason": "stop",
					"message": {
						"role": "assistant",
						"content": "This is the mocked test response content."
					},
					"delta": {"role":"assistant", "content":""}
				}
			]
		}`
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	// Test the mock=true path
	t.Run("Mock=true", func(t *testing.T) {
		prompt := "Test prompt for mock"
		// The mock response in CallPerplexityAPI is hardcoded and different
		// from the one served by our httptest server. This test checks the
		// hardcoded mock response.
		result, _ := CallPerplexityAPI(prompt, true)

		if result == "" {
			t.Fatal("Expected a non-empty mock response, got empty string")
		}

		// Deserialize the hardcoded mock response
		var response PerplexityResponse
		err := json.Unmarshal([]byte(result), &response)
		if err != nil {
			t.Fatalf("Failed to unmarshal hardcoded mock JSON response: %v\nResponse was: %s", err, result)
		}

		// --- Assertions on the deserialized hardcoded mock response ---
		expectedID := "a83283d7-4307-4c36-850f-56b648ae90a1"
		if response.ID != expectedID {
			t.Errorf("Expected mock response ID '%s', got '%s'", expectedID, response.ID)
		}

		if len(response.Choices) == 0 {
			t.Fatal("Expected at least one choice in mock response, got none")
		}

		expectedContentStart := "## What is Perplexity?"
		if !strings.HasPrefix(response.Choices[0].Message.Content, expectedContentStart) {
			t.Errorf("Expected mock response content to start with '%s', got '%s'", expectedContentStart, response.Choices[0].Message.Content)
		}

		if response.Usage.PromptTokens != 12 {
			t.Errorf("Expected mock usage prompt_tokens %d, got %d", 12, response.Usage.PromptTokens)
		}
	})
}
