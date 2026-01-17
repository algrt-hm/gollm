package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ChatMessage represents a single message in the conversation
type ChatMessage struct {
	Role    string // "user" or "assistant"
	Content string
}

// Provider represents a chat provider
type Provider string

const (
	ProviderClaude     Provider = "Claude"
	ProviderChatGPT    Provider = "ChatGPT"
	ProviderGemini     Provider = "Gemini"
	ProviderCerebras   Provider = "Cerebras"
	ProviderPerplexity Provider = "Perplexity"
)

// AllProviders lists all available providers
var AllProviders = []Provider{
	ProviderClaude,
	ProviderChatGPT,
	ProviderGemini,
	ProviderCerebras,
	ProviderPerplexity,
}

// ProviderPreference defines the order of preference for auto-selecting a provider
// The first provider with a valid API key will be used
var ProviderPreference = []Provider{
	ProviderCerebras,
	ProviderClaude,
	ProviderChatGPT,
	ProviderGemini,
	ProviderPerplexity,
}

// hasAPIKey checks if an API key is set for the given provider
func hasAPIKey(provider Provider) bool {
	switch provider {
	case ProviderClaude:
		return getClaudeAPIKey() != ""
	case ProviderChatGPT:
		return getChatGPTAPIKey() != ""
	case ProviderGemini:
		return getGeminiAPIKey() != ""
	case ProviderCerebras:
		return getCerebrasAPIKey() != ""
	case ProviderPerplexity:
		return getPerplexityAPIKey() != ""
	default:
		return false
	}
}

// GetPreferredProvider returns the first provider from the preference list
// that has an API key set, or falls back to the first in the list
func GetPreferredProvider() Provider {
	for _, p := range ProviderPreference {
		if hasAPIKey(p) {
			return p
		}
	}
	// Fallback to first preference if none have keys (will error later)
	return ProviderPreference[0]
}

// getAvailableModels returns the available models for a provider
func getAvailableModels(provider Provider) []string {
	switch provider {
	case ProviderClaude:
		return AvailableModels.Claude
	case ProviderChatGPT:
		return AvailableModels.ChatGPT
	case ProviderGemini:
		return AvailableModels.Gemini
	case ProviderCerebras:
		return AvailableModels.Cerebras
	case ProviderPerplexity:
		return AvailableModels.Perplexity
	default:
		return nil
	}
}

// getDefaultModel returns the default model for a provider
func getDefaultModel(provider Provider) string {
	switch provider {
	case ProviderClaude:
		return DefaultModels.Claude
	case ProviderChatGPT:
		return string(DefaultModels.ChatGPT)
	case ProviderGemini:
		return DefaultModels.Gemini
	case ProviderCerebras:
		return DefaultModels.Cerebras
	case ProviderPerplexity:
		return DefaultModels.Perplexity
	default:
		return ""
	}
}

// callChat calls the appropriate chat function for the provider
func callChat(provider Provider, history []ChatMessage, userInput string, model string) (string, error) {
	switch provider {
	case ProviderClaude:
		return claudeChat(history, userInput, model)
	case ProviderChatGPT:
		return chatGPTChat(history, userInput, model)
	case ProviderGemini:
		return geminiChat(history, userInput, model)
	case ProviderCerebras:
		return cerebrasChat(history, userInput, model)
	case ProviderPerplexity:
		return perplexityChat(history, userInput, model)
	default:
		return "", fmt.Errorf("unknown provider: %s", provider)
	}
}

// InteractiveSession runs an interactive chat session with the selected model
func InteractiveSession(o optionsStruct) {
	var provider Provider
	var currentModel string

	// Determine which provider to use
	switch {
	case o.useClaude:
		provider = ProviderClaude
		currentModel = DefaultModels.Claude
	case o.useChatGPT:
		provider = ProviderChatGPT
		currentModel = string(DefaultModels.ChatGPT)
	case o.useGemini:
		provider = ProviderGemini
		currentModel = DefaultModels.Gemini
	case o.useCerebras:
		provider = ProviderCerebras
		currentModel = DefaultModels.Cerebras
	case o.usePerplexity:
		provider = ProviderPerplexity
		currentModel = DefaultModels.Perplexity
	default:
		Fatalf("No model selected for interactive mode")
	}

	// Render welcome header with markdown
	welcomeHeader := fmt.Sprintf("# Interactive Chat\n\n**Provider:** %s\n**Model:** `%s`\n\nCommands:\n- `exit` or `quit` - end the session\n- `/amnesia` - toggle amnesia mode (disables chat history)\n- `/model` - list available models\n- `/model <number>` - switch to a different model\n- `/provider` - list available providers\n- `/provider <number>` - switch to a different provider\n\n---", provider, currentModel)
	RenderWithGlamourPtr(welcomeHeader)

	var history []ChatMessage
	reader := bufio.NewReader(os.Stdin)
	amnesiaMode := false
	// Track if we just rendered something with glamour (which adds its own trailing newline)
	justRendered := true

	for {
		if justRendered {
			fmt.Print("> ")
		} else {
			fmt.Print("\n> ")
		}
		justRendered = false

		input, err := reader.ReadString('\n')
		if err != nil {
			RenderWithGlamourPtr("\n*Goodbye!*")
			return
		}

		input = strings.TrimSpace(input)

		// Check for exit commands
		if input == "exit" || input == "quit" {
			RenderWithGlamourPtr("*Goodbye!*")
			return
		}

		// Check for amnesia toggle
		if input == "/amnesia" {
			amnesiaMode = !amnesiaMode
			if amnesiaMode {
				RenderWithGlamourPtr("*Amnesia mode enabled* - chat history will not be sent to the model")
			} else {
				RenderWithGlamourPtr("*Amnesia mode disabled* - chat history will be sent to the model")
			}
			justRendered = true
			continue
		}

		// Check for model command
		if input == "/model" || strings.HasPrefix(input, "/model ") {
			availableModels := getAvailableModels(provider)

			if input == "/model" {
				// List available models
				var sb strings.Builder
				sb.WriteString(fmt.Sprintf("**Available %s models:**\n\n", provider))
				for i, m := range availableModels {
					marker := "  "
					if m == currentModel {
						marker = "→ "
					}
					sb.WriteString(fmt.Sprintf("%s%d. `%s`\n", marker, i+1, m))
				}
				sb.WriteString(fmt.Sprintf("\nUse `/model <number>` to switch (e.g., `/model 2`)"))
				RenderWithGlamourPtr(sb.String())
				justRendered = true
				continue
			}

			// Parse model number
			parts := strings.SplitN(input, " ", 2)
			if len(parts) == 2 {
				numStr := strings.TrimSpace(parts[1])
				num, err := strconv.Atoi(numStr)
				if err != nil || num < 1 || num > len(availableModels) {
					RenderWithGlamourPtr(fmt.Sprintf("*Invalid model number.* Use 1-%d", len(availableModels)))
					justRendered = true
					continue
				}
				currentModel = availableModels[num-1]
				RenderWithGlamourPtr(fmt.Sprintf("*Switched to model:* `%s`", currentModel))
				justRendered = true
				continue
			}
		}

		// Check for provider command
		if input == "/provider" || strings.HasPrefix(input, "/provider ") {
			if input == "/provider" {
				// List available providers
				var sb strings.Builder
				sb.WriteString("**Available providers:**\n\n")
				for i, p := range AllProviders {
					marker := "  "
					if p == provider {
						marker = "→ "
					}
					sb.WriteString(fmt.Sprintf("%s%d. %s\n", marker, i+1, p))
				}
				sb.WriteString(fmt.Sprintf("\nUse `/provider <number>` to switch (e.g., `/provider 2`)"))
				RenderWithGlamourPtr(sb.String())
				justRendered = true
				continue
			}

			// Parse provider number
			parts := strings.SplitN(input, " ", 2)
			if len(parts) == 2 {
				numStr := strings.TrimSpace(parts[1])
				num, err := strconv.Atoi(numStr)
				if err != nil || num < 1 || num > len(AllProviders) {
					RenderWithGlamourPtr(fmt.Sprintf("*Invalid provider number.* Use 1-%d", len(AllProviders)))
					justRendered = true
					continue
				}
				provider = AllProviders[num-1]
				currentModel = getDefaultModel(provider)
				RenderWithGlamourPtr(fmt.Sprintf("*Switched to provider:* **%s**\n*Model:* `%s`", provider, currentModel))
				justRendered = true
				continue
			}
		}

		// Skip empty input
		if input == "" {
			continue
		}

		// Get response from model (use empty history if amnesia mode is on)
		var historyToSend []ChatMessage
		if !amnesiaMode {
			historyToSend = history
		}
		response, err := callChat(provider, historyToSend, input, currentModel)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}

		// Add to history (still track it even in amnesia mode, in case user toggles back)
		history = append(history, ChatMessage{Role: "user", Content: input})
		history = append(history, ChatMessage{Role: "assistant", Content: response})

		// Display response with markdown rendering
		RenderWithGlamourPtr(response)
		justRendered = true
	}
}
