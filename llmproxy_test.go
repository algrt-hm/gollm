package main

import (
	"os"
	"strings"
	"testing"
)

func TestLLMProxyWrapperForProvider(t *testing.T) {
	// To turn off output if we don't want it
	if testingSuppressOutput {
		t.Cleanup(func() { RenderWithGlamourPtr = RenderWithGlamour })
		RenderWithGlamourPtr = func(s string) {}
	}

	t.Run("mock_response_contains_expected_content", func(t *testing.T) {
		result := LLMProxyWrapperForProvider("ChatGPT", "Mock prompt", "openai:gpt-5.2", true, false, true)
		if result != "This is a mocked LLM Proxy response." {
			t.Errorf("Expected mocked response content, got %q", result)
		}
	})

	t.Run("mock_response_has_provider_header_in_normal_mode", func(t *testing.T) {
		result := LLMProxyWrapperForProvider("ChatGPT", "Mock prompt", "openai:gpt-5.2", true, false, false)
		if !strings.Contains(result, "# ChatGPT (via proxy)") {
			t.Errorf("Expected '# ChatGPT (via proxy)' header, got %q", result)
		}
		if !strings.Contains(result, "mocked LLM Proxy response") {
			t.Errorf("Expected mocked response in output, got %q", result)
		}
	})
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
	t.Run("x_flag_sets_useLLMProxy", func(t *testing.T) {
		argv := []string{"-x", "hello"}
		ret, err := handleOpts(argv, len(argv))
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !ret.useLLMProxy {
			t.Error("Expected useLLMProxy to be true")
		}
		if ret.promptText != "hello" {
			t.Errorf("Expected prompt 'hello', got %q", ret.promptText)
		}
	})

	t.Run("x_flag_not_in_default_all_models", func(t *testing.T) {
		// When no model flags are specified, all 5 default models are used but NOT LLM Proxy
		argv := []string{"hello"}
		ret, err := handleOpts(argv, len(argv))
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if ret.useLLMProxy {
			t.Error("Expected useLLMProxy to be false when no flags specified")
		}
		if !ret.useChatGPT || !ret.useGemini || !ret.usePerplexity || !ret.useCerebras || !ret.useClaude {
			t.Error("Expected all 5 default models to be true")
		}
	})

	t.Run("x_flag_interactive_mode_auto_selects_provider", func(t *testing.T) {
		// -i -x should auto-select a provider (proxy mode, not a provider itself)
		argv := []string{"-i", "-x"}
		ret, err := handleOpts(argv, len(argv))
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !ret.interactive {
			t.Error("Expected interactive to be true")
		}
		if !ret.useLLMProxy {
			t.Error("Expected useLLMProxy to be true")
		}
		// Should auto-select a provider since -x is not a model flag
		hasProvider := ret.useChatGPT || ret.useGemini || ret.usePerplexity || ret.useCerebras || ret.useClaude
		if !hasProvider {
			t.Error("Expected a provider to be auto-selected")
		}
	})

	t.Run("x_with_model_interactive_is_valid", func(t *testing.T) {
		// -i -x -c should work: ChatGPT via proxy
		argv := []string{"-i", "-x", "-c"}
		ret, err := handleOpts(argv, len(argv))
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !ret.interactive {
			t.Error("Expected interactive to be true")
		}
		if !ret.useLLMProxy {
			t.Error("Expected useLLMProxy to be true")
		}
		if !ret.useChatGPT {
			t.Error("Expected useChatGPT to be true")
		}
	})

	t.Run("x_flag_is_routing_not_model", func(t *testing.T) {
		// -x alone should use all 5 default models (routed through proxy)
		argv := []string{"-x", "hello"}
		ret, err := handleOpts(argv, len(argv))
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !ret.useLLMProxy {
			t.Error("Expected useLLMProxy to be true")
		}
		if !ret.useChatGPT || !ret.useGemini || !ret.usePerplexity || !ret.useCerebras || !ret.useClaude {
			t.Error("Expected all 5 default models to be true when -x is the only flag")
		}
	})
}

func TestProxyModelName(t *testing.T) {
	tests := []struct {
		provider Provider
		model    string
		expected string
	}{
		{ProviderChatGPT, "gpt-5.2", "openai:gpt-5.2"},
		{ProviderClaude, "claude-sonnet-4-5-20250929", "anthropic:claude-sonnet-4-5-20250929"},
		{ProviderGemini, "models/gemini-3-pro-preview", "gemini:models/gemini-3-pro-preview"},
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
