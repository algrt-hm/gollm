package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/term"
)

// ChatMessage represents a single message in the conversation
type ChatMessage struct {
	Role    string // "user" or "assistant"
	Content string
}

// Spinner displays an ASCII spinner while waiting for operations
type Spinner struct {
	frames  []string
	delay   time.Duration
	stop    chan struct{}
	stopped chan struct{}
	mu      sync.Mutex
	running bool
}

// NewSpinner creates a new spinner with default frames
func NewSpinner() *Spinner {
	return &Spinner{
		frames:  []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		delay:   80 * time.Millisecond,
		stop:    make(chan struct{}),
		stopped: make(chan struct{}),
	}
}

// Start begins the spinner animation
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.stop = make(chan struct{})
	s.stopped = make(chan struct{})
	s.mu.Unlock()

	go func() {
		defer close(s.stopped)
		i := 0
		for {
			select {
			case <-s.stop:
				// Clear the spinner line
				fmt.Print("\r\033[K")
				return
			default:
				fmt.Printf("\r%s ", s.frames[i%len(s.frames)])
				i++
				time.Sleep(s.delay)
			}
		}
	}()
}

// Stop halts the spinner and clears the line
func (s *Spinner) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()

	close(s.stop)
	<-s.stopped
}

// MultilineReader handles multiline input with Enter for newlines and Ctrl+D to submit
type MultilineReader struct {
	fd int
}

// NewMultilineReader creates a new multiline reader
func NewMultilineReader() *MultilineReader {
	return &MultilineReader{
		fd: int(os.Stdin.Fd()),
	}
}

// ReadMultiline reads multiline input from the terminal
// Enter adds a newline, Ctrl+D submits
// Returns the complete input string and any error
func (r *MultilineReader) ReadMultiline() (string, error) {
	// Check if stdin is a terminal
	if !term.IsTerminal(r.fd) {
		return r.readLineFallback()
	}

	// Put terminal in raw mode to capture individual keystrokes
	oldState, err := term.MakeRaw(r.fd)
	if err != nil {
		// Fallback to simple line reading if raw mode fails
		fmt.Fprintf(os.Stderr, "[raw mode failed: %v]\n", err)
		return r.readLineFallback()
	}

	// Set up signal handler to restore terminal on interrupt
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	// Ensure terminal is restored on exit
	restore := func() {
		term.Restore(r.fd, oldState)
	}
	defer restore()

	// Handle signals in background
	go func() {
		<-sigCh
		restore()
		os.Exit(0)
	}()

	var buf strings.Builder
	b := make([]byte, 1)

	for {
		_, err := os.Stdin.Read(b)
		if err != nil {
			return buf.String(), err
		}

		char := b[0]

		// Handle escape sequences - just skip them (arrow keys, etc.)
		if char == 0x1b { // ESC
			// Read and discard escape sequence
			// Most sequences are 2-3 bytes: ESC [ X
			seq := make([]byte, 8)
			os.Stdin.Read(seq) // This may block briefly but escape sequences come fast
			continue
		}

		// Ctrl+D (EOT) - submit input
		if char == 0x04 {
			fmt.Print("\r\n")
			return buf.String(), nil
		}

		// Ctrl+C - abort current input
		if char == 0x03 {
			fmt.Print("^C\r\n")
			return "", nil
		}

		// Enter (CR or LF) - check for commands or add newline
		if char == '\r' || char == '\n' {
			// Check if buffer is a command that should submit immediately
			trimmed := strings.TrimSpace(buf.String())
			if isImmediateCommand(trimmed) {
				fmt.Print("\r\n")
				return buf.String(), nil
			}
			// Otherwise add newline and continue
			buf.WriteByte('\n')
			fmt.Print("\r\n  ") // New line with continuation indent
			continue
		}

		// Backspace handling
		if char == 0x7f || char == 0x08 {
			str := buf.String()
			if len(str) > 0 {
				if str[len(str)-1] == '\n' {
					// Deleting a newline - move cursor up
					buf.Reset()
					buf.WriteString(str[:len(str)-1])
					fmt.Print("\r\033[K") // Clear current line
					lines := strings.Split(buf.String(), "\n")
					if len(lines) > 0 {
						lastLine := lines[len(lines)-1]
						fmt.Print("\033[A\r") // Move up
						if len(lines) == 1 {
							fmt.Print("> " + lastLine)
						} else {
							fmt.Print("  " + lastLine)
						}
					}
				} else {
					buf.Reset()
					buf.WriteString(str[:len(str)-1])
					fmt.Print("\b \b") // Erase character
				}
			}
			continue
		}

		// Regular character - add to buffer and echo
		buf.WriteByte(char)
		fmt.Print(string(char))
	}
}

// readLineFallback uses simple line reading when raw mode is unavailable
func (r *MultilineReader) readLineFallback() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	return strings.TrimSuffix(input, "\n"), err
}

// isImmediateCommand checks if the input is a command that should submit on Enter
func isImmediateCommand(input string) bool {
	input = strings.TrimSpace(input)
	if input == "" {
		return false
	}
	// Exit commands
	if input == "exit" || input == "quit" {
		return true
	}
	// Slash commands
	if strings.HasPrefix(input, "/") {
		return true
	}
	return false
}

// CommandAction represents the result of processing a command
type CommandAction int

const (
	ActionNone           CommandAction = iota // Not a command, send to LLM
	ActionExit                                // Exit the session
	ActionHelp                                // Show help
	ActionAmnesia                             // Toggle amnesia mode
	ActionListModels                          // List available models
	ActionSwitchModel                         // Switch to a different model
	ActionListProviders                       // List available providers
	ActionSwitchProvider                      // Switch to a different provider
	ActionSave                                // Save conversation
	ActionEmpty                               // Empty input, ignore
	ActionError                               // Error occurred
)

// CommandResult contains the result of processing a command
type CommandResult struct {
	Action       CommandAction
	Message      string // Message to display to user
	Error        error  // Error if Action is ActionError
	ModelIndex   int    // For ActionSwitchModel
	ProviderIndex int   // For ActionSwitchProvider
}

// SessionState holds the current state of an interactive session
type SessionState struct {
	Provider     Provider
	CurrentModel string
	History      []ChatMessage
	AmnesiaMode  bool
	ProxyMode    bool
}

// handleCommand processes user input and returns what action to take
// This function is separated out to make it testable
func handleCommand(input string, state *SessionState) CommandResult {
	input = strings.TrimSpace(input)

	// Empty input
	if input == "" {
		return CommandResult{Action: ActionEmpty}
	}

	// Exit commands
	if input == "exit" || input == "quit" {
		return CommandResult{Action: ActionExit, Message: "*Goodbye!*"}
	}

	// Help command
	if input == "/help" {
		return CommandResult{Action: ActionHelp}
	}

	// Amnesia toggle
	if input == "/amnesia" {
		state.AmnesiaMode = !state.AmnesiaMode
		msg := "*Amnesia mode enabled* - chat history will not be sent to the model"
		if !state.AmnesiaMode {
			msg = "*Amnesia mode disabled* - chat history will be sent to the model"
		}
		return CommandResult{Action: ActionAmnesia, Message: msg}
	}

	// Model commands
	if input == "/model" {
		return CommandResult{Action: ActionListModels}
	}
	if strings.HasPrefix(input, "/model ") {
		parts := strings.SplitN(input, " ", 2)
		if len(parts) == 2 {
			numStr := strings.TrimSpace(parts[1])
			num, err := strconv.Atoi(numStr)
			if err != nil {
				availableModels := getAvailableModels(state.Provider)
				return CommandResult{
					Action:  ActionError,
					Message: fmt.Sprintf("*Invalid model number.* Use 1-%d", len(availableModels)),
				}
			}
			availableModels := getAvailableModels(state.Provider)
			if num < 1 || num > len(availableModels) {
				return CommandResult{
					Action:  ActionError,
					Message: fmt.Sprintf("*Invalid model number.* Use 1-%d", len(availableModels)),
				}
			}
			state.CurrentModel = availableModels[num-1]
			return CommandResult{
				Action:     ActionSwitchModel,
				Message:    fmt.Sprintf("*Switched to model:* `%s`", state.CurrentModel),
				ModelIndex: num - 1,
			}
		}
	}

	// Provider commands
	if input == "/provider" {
		return CommandResult{Action: ActionListProviders}
	}
	if strings.HasPrefix(input, "/provider ") {
		parts := strings.SplitN(input, " ", 2)
		if len(parts) == 2 {
			numStr := strings.TrimSpace(parts[1])
			num, err := strconv.Atoi(numStr)
			if err != nil || num < 1 || num > len(AllProviders) {
				return CommandResult{
					Action:  ActionError,
					Message: fmt.Sprintf("*Invalid provider number.* Use 1-%d", len(AllProviders)),
				}
			}
			newProvider := AllProviders[num-1]
			if !state.ProxyMode && !hasAPIKey(newProvider) {
				return CommandResult{
					Action:  ActionError,
					Message: fmt.Sprintf("*Cannot switch to %s:* API key not set.\n\nSet the environment variable for this provider and restart.", newProvider),
				}
			}
			state.Provider = newProvider
			state.CurrentModel = getDefaultModel(newProvider)
			return CommandResult{
				Action:        ActionSwitchProvider,
				Message:       fmt.Sprintf("*Switched to provider:* **%s**\n*Model:* `%s`", state.Provider, state.CurrentModel),
				ProviderIndex: num - 1,
			}
		}
	}

	// Save command (case-insensitive)
	if strings.EqualFold(input, "/save") {
		if len(state.History) == 0 {
			return CommandResult{
				Action:  ActionError,
				Message: "*No conversation to save yet.*",
			}
		}
		return CommandResult{Action: ActionSave}
	}

	// Not a command - will be sent to LLM
	return CommandResult{Action: ActionNone}
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
// When proxyMode is true, all calls route through the LLM Proxy
func callChat(provider Provider, history []ChatMessage, userInput string, model string, proxyMode bool) (string, error) {
	if proxyMode {
		return llmproxyChat(history, userInput, proxyModelName(provider, model))
	}
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

// formatConversationMarkdown formats the chat history as markdown
func formatConversationMarkdown(history []ChatMessage, provider Provider, model string) string {
	var sb strings.Builder

	sb.WriteString("# Conversation Log\n\n")
	sb.WriteString(fmt.Sprintf("**Provider:** %s\n", provider))
	sb.WriteString(fmt.Sprintf("**Model:** %s\n", model))
	sb.WriteString(fmt.Sprintf("**Date:** %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString("---\n\n")

	for _, msg := range history {
		if msg.Role == "user" {
			sb.WriteString("## User\n\n")
		} else {
			sb.WriteString("## Assistant\n\n")
		}
		sb.WriteString(msg.Content)
		sb.WriteString("\n\n---\n\n")
	}

	return sb.String()
}

// saveConversation saves the conversation history to a markdown file
func saveConversation(filename string, history []ChatMessage, provider Provider, model string) error {
	// If no path separator, prepend $HOME
	if !strings.Contains(filename, string(filepath.Separator)) {
		home := getHomeDir()
		filename = filepath.Join(home, filename)
	}

	// Ensure .md extension
	if !strings.HasSuffix(strings.ToLower(filename), ".md") {
		filename = filename + ".md"
	}

	content := formatConversationMarkdown(history, provider, model)
	err := os.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		return err
	}

	return nil
}

// InteractiveSession runs an interactive chat session with the selected model
func InteractiveSession(o optionsStruct) {
	var provider Provider
	var currentModel string

	proxyMode := o.useLLMProxy

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

	// Help text for commands (reused in welcome and /help)
	helpText := `**Input:**
- ` + "`Enter`" + ` - add a new line (multiline input)
- ` + "`Ctrl+D`" + ` - send message to model
- ` + "`Ctrl+C`" + ` - cancel current input

**Commands:**
- ` + "`/help`" + ` - show this help message
- ` + "`exit`" + ` or ` + "`quit`" + ` - end the session
- ` + "`/amnesia`" + ` - toggle amnesia mode (disables chat history)
- ` + "`/model`" + ` - list available models
- ` + "`/model <number>`" + ` - switch to a different model
- ` + "`/provider`" + ` - list available providers
- ` + "`/provider <number>`" + ` - switch to a different provider
- ` + "`/save`" + ` - save conversation to markdown file`

	// Render welcome header with markdown
	providerLabel := string(provider)
	if proxyMode {
		providerLabel += " (via proxy)"
	}
	welcomeHeader := fmt.Sprintf("# Interactive Chat\n\n**Provider:** %s\n**Model:** `%s`\n\n%s\n\n---", providerLabel, currentModel, helpText)
	RenderWithGlamourPtr(welcomeHeader)

	var history []ChatMessage
	mlReader := NewMultilineReader()
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

		input, err := mlReader.ReadMultiline()
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

		// Check for help command
		if input == "/help" {
			RenderWithGlamourPtr(helpText)
			justRendered = true
			continue
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
					sb.WriteString(fmt.Sprintf("%s%d\\. `%s`\n", marker, i+1, m))
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
					keyStatus := ""
					if !proxyMode && !hasAPIKey(p) {
						keyStatus = " *(no API key)*"
					}
					sb.WriteString(fmt.Sprintf("%s%d\\. %s%s\n", marker, i+1, p, keyStatus))
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
				newProvider := AllProviders[num-1]
				// Check if the user has an API key for this provider (skip in proxy mode)
				if !proxyMode && !hasAPIKey(newProvider) {
					RenderWithGlamourPtr(fmt.Sprintf("*Cannot switch to %s:* API key not set.\n\nSet the environment variable for this provider and restart.", newProvider))
					justRendered = true
					continue
				}
				provider = newProvider
				currentModel = getDefaultModel(provider)
				RenderWithGlamourPtr(fmt.Sprintf("*Switched to provider:* **%s**\n*Model:* `%s`", provider, currentModel))
				justRendered = true
				continue
			}
		}

		// Check for save command (case-insensitive)
		if strings.EqualFold(input, "/save") {
			if len(history) == 0 {
				RenderWithGlamourPtr("*No conversation to save yet.*")
				justRendered = true
				continue
			}

			fmt.Print("Enter filename (will be saved to $HOME if no path given): ")
			// Use simple line reader for filename (single line input)
			lineReader := bufio.NewReader(os.Stdin)
			filename, err := lineReader.ReadString('\n')
			if err != nil {
				RenderWithGlamourPtr("*Error reading filename.*")
				justRendered = true
				continue
			}
			filename = strings.TrimSpace(filename)

			if filename == "" {
				RenderWithGlamourPtr("*No filename provided, save cancelled.*")
				justRendered = true
				continue
			}

			err = saveConversation(filename, history, provider, currentModel)
			if err != nil {
				RenderWithGlamourPtr(fmt.Sprintf("*Error saving conversation:* %v", err))
				justRendered = true
				continue
			}

			// Show the full path that was saved
			savedPath := filename
			if !strings.Contains(filename, string(filepath.Separator)) {
				savedPath = filepath.Join(getHomeDir(), filename)
			}
			if !strings.HasSuffix(strings.ToLower(savedPath), ".md") {
				savedPath = savedPath + ".md"
			}
			RenderWithGlamourPtr(fmt.Sprintf("*Conversation saved to:* `%s`", savedPath))
			justRendered = true
			continue
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

		// Start spinner while waiting for response
		spinner := NewSpinner()
		spinner.Start()
		response, err := callChat(provider, historyToSend, input, currentModel, proxyMode)
		spinner.Stop()

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
