package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/openai/openai-go/v2"
)

func TestLLMProxyWrapperForProvider(t *testing.T) {
	// To turn off output if we don't want it
	if testingSuppressOutput {
		t.Cleanup(func() { RenderWithGlamourPtr = RenderWithGlamour })
		RenderWithGlamourPtr = func(s string) {}
	}

	t.Run("mock_response_contains_expected_content", func(t *testing.T) {
		result, err := LLMProxyWrapperForProvider("ChatGPT", "Mock prompt", "openai:gpt-5.4", true, false, true)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if result != "This is a mocked LLM Proxy response." {
			t.Errorf("Expected mocked response content, got %q", result)
		}
	})

	t.Run("mock_response_has_provider_header_in_normal_mode", func(t *testing.T) {
		result, err := LLMProxyWrapperForProvider("ChatGPT", "Mock prompt", "openai:gpt-5.4", true, false, false)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if !strings.Contains(result, "# ChatGPT (via proxy)") {
			t.Errorf("Expected '# ChatGPT (via proxy)' header, got %q", result)
		}
		if !strings.Contains(result, "mocked LLM Proxy response") {
			t.Errorf("Expected mocked response in output, got %q", result)
		}
	})
}

func TestExtractCitations(t *testing.T) {
	raw := `{"id":"x","object":"chat.completion","created":1,"model":"sonar-pro","citations":["https://go.dev/","https://example.com/"],"choices":[{"index":0,"finish_reason":"stop","message":{"role":"assistant","content":"Go was released in 2009[1]."}}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`
	var c openai.ChatCompletion
	if err := c.UnmarshalJSON([]byte(raw)); err != nil {
		t.Fatalf("Failed to unmarshal fixture: %v", err)
	}

	citations := extractCitations(&c)
	if len(citations) != 2 || citations[0] != "https://go.dev/" {
		t.Errorf("Expected 2 citations starting with go.dev, got %v", citations)
	}

	formatted := appendCitations(c.Choices[0].Message.Content, citations)
	if !strings.Contains(formatted, "2009[^1]") {
		t.Errorf("Expected [1] rewritten to footnote form, got %q", formatted)
	}
	if !strings.Contains(formatted, "Citations:") || !strings.Contains(formatted, "1. https://go.dev/") {
		t.Errorf("Expected citation list in output, got %q", formatted)
	}

	// No citations field -> nil, and content passes through unchanged
	var plain openai.ChatCompletion
	if err := plain.UnmarshalJSON([]byte(`{"id":"y","object":"chat.completion","created":1,"model":"gpt-5.5","choices":[]}`)); err != nil {
		t.Fatalf("Failed to unmarshal fixture: %v", err)
	}
	if got := extractCitations(&plain); got != nil {
		t.Errorf("Expected nil citations, got %v", got)
	}
	if got := appendCitations("hello", nil); got != "hello" {
		t.Errorf("Expected content unchanged with no citations, got %q", got)
	}
}

func TestGetLLMProxyURL(t *testing.T) {
	t.Run("returns_default_when_env_unset", func(t *testing.T) {
		original := os.Getenv(llmProxyURLEnvVar)
		os.Unsetenv(llmProxyURLEnvVar)
		t.Cleanup(func() {
			if original != "" {
				os.Setenv(llmProxyURLEnvVar, original)
			}
		})

		url := getLLMProxyURL()
		if url != "http://localhost:8000/v1" {
			t.Errorf("Expected default URL, got %q", url)
		}
	})

	t.Run("returns_env_value_when_set", func(t *testing.T) {
		original := os.Getenv(llmProxyURLEnvVar)
		os.Setenv(llmProxyURLEnvVar, "http://custom:9000/v1")
		t.Cleanup(func() {
			if original != "" {
				os.Setenv(llmProxyURLEnvVar, original)
			} else {
				os.Unsetenv(llmProxyURLEnvVar)
			}
		})

		url := getLLMProxyURL()
		if url != "http://custom:9000/v1" {
			t.Errorf("Expected custom URL, got %q", url)
		}
	})
}

func TestLLMProxyFlagParsing(t *testing.T) {
	t.Run("x_flag_sets_bypassLLMProxy", func(t *testing.T) {
		argv := []string{"-x", "hello"}
		ret, err := handleOpts(argv, len(argv))
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !ret.bypassLLMProxy {
			t.Error("Expected bypassLLMProxy to be true")
		}
		if ret.promptText != "hello" {
			t.Errorf("Expected prompt 'hello', got %q", ret.promptText)
		}
	})

	t.Run("x_flag_not_in_default_all_models", func(t *testing.T) {
		// When no model flags are specified, all 5 default models are used and bypassLLMProxy is false
		argv := []string{"hello"}
		ret, err := handleOpts(argv, len(argv))
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if ret.bypassLLMProxy {
			t.Error("Expected bypassLLMProxy to be false when no flags specified")
		}
		if !ret.useChatGPT || !ret.useGemini || !ret.usePerplexity || !ret.useCerebras || !ret.useClaude {
			t.Error("Expected all 5 default models to be true")
		}
	})

	t.Run("x_flag_interactive_mode_auto_selects_provider", func(t *testing.T) {
		// -i -x should auto-select a provider (-x is a bypass flag, not a provider itself)
		argv := []string{"-i", "-x"}
		ret, err := handleOpts(argv, len(argv))
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !ret.interactive {
			t.Error("Expected interactive to be true")
		}
		if !ret.bypassLLMProxy {
			t.Error("Expected bypassLLMProxy to be true")
		}
		// Should auto-select a provider since -x is not a model flag
		hasProvider := ret.useChatGPT || ret.useGemini || ret.usePerplexity || ret.useCerebras || ret.useClaude
		if !hasProvider {
			t.Error("Expected a provider to be auto-selected")
		}
	})

	t.Run("x_with_model_interactive_is_valid", func(t *testing.T) {
		// -i -x -c should work: ChatGPT with proxy bypassed
		argv := []string{"-i", "-x", "-c"}
		ret, err := handleOpts(argv, len(argv))
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !ret.interactive {
			t.Error("Expected interactive to be true")
		}
		if !ret.bypassLLMProxy {
			t.Error("Expected bypassLLMProxy to be true")
		}
		if !ret.useChatGPT {
			t.Error("Expected useChatGPT to be true")
		}
	})

	t.Run("x_flag_is_bypass_not_model", func(t *testing.T) {
		// -x alone should use all 5 default models (with proxy bypassed)
		argv := []string{"-x", "hello"}
		ret, err := handleOpts(argv, len(argv))
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !ret.bypassLLMProxy {
			t.Error("Expected bypassLLMProxy to be true")
		}
		if !ret.useChatGPT || !ret.useGemini || !ret.usePerplexity || !ret.useCerebras || !ret.useClaude {
			t.Error("Expected all 5 default models to be true when -x is the only flag")
		}
	})
}

func TestCheckLLMProxyHealth(t *testing.T) {
	t.Run("returns_false_when_env_unset", func(t *testing.T) {
		original := os.Getenv(llmProxyURLEnvVar)
		os.Unsetenv(llmProxyURLEnvVar)
		t.Cleanup(func() {
			if original != "" {
				os.Setenv(llmProxyURLEnvVar, original)
			}
		})

		if CheckLLMProxyHealth() {
			t.Error("Expected false when env var is unset")
		}
	})

	t.Run("returns_true_when_proxy_healthy", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		original := os.Getenv(llmProxyURLEnvVar)
		os.Setenv(llmProxyURLEnvVar, server.URL+"/v1")
		t.Cleanup(func() {
			if original != "" {
				os.Setenv(llmProxyURLEnvVar, original)
			} else {
				os.Unsetenv(llmProxyURLEnvVar)
			}
		})

		if !CheckLLMProxyHealth() {
			t.Error("Expected true when proxy returns 200 on /health")
		}
	})

	t.Run("returns_false_when_proxy_unreachable", func(t *testing.T) {
		original := os.Getenv(llmProxyURLEnvVar)
		os.Setenv(llmProxyURLEnvVar, "http://127.0.0.1:1/v1")
		t.Cleanup(func() {
			if original != "" {
				os.Setenv(llmProxyURLEnvVar, original)
			} else {
				os.Unsetenv(llmProxyURLEnvVar)
			}
		})

		if CheckLLMProxyHealth() {
			t.Error("Expected false when proxy is unreachable")
		}
	})
}

func TestProxyModelName(t *testing.T) {
	tests := []struct {
		provider Provider
		model    string
		expected string
	}{
		{ProviderChatGPT, "gpt-5.4", "openai:gpt-5.4"},
		{ProviderClaude, "claude-sonnet-4-6", "anthropic:claude-sonnet-4-6"},
		{ProviderGemini, "models/gemini-3.1-pro-preview", "gemini:models/gemini-3.1-pro-preview"},
		{ProviderCerebras, "gpt-oss-120b", "cerebras:gpt-oss-120b"},
		{ProviderPerplexity, "sonar-pro", "perplexity:sonar-pro"},
	}

	for _, tc := range tests {
		t.Run(string(tc.provider), func(t *testing.T) {
			result := proxyModelName(tc.provider, tc.model)
			if result != tc.expected {
				t.Errorf("proxyModelName(%s, %q) = %q, want %q", tc.provider, tc.model, result, tc.expected)
			}
		})
	}
}
