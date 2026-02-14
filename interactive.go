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

// MultilineReader handles multiline input with history and line editing.
// Enter adds newlines (Ctrl+D submits), with support for cursor movement,
// input history (Up/Down), and standard terminal editing shortcuts.
type MultilineReader struct {
	fd      int
	history []string
	histIdx int
	saved   string // saved current input when browsing history
}

// NewMultilineReader creates a new multiline reader
func NewMultilineReader() *MultilineReader {
	return &MultilineReader{
		fd: int(os.Stdin.Fd()),
	}
}

// cursorPos returns the display line and column for a cursor position in the buffer
func cursorPos(buf []rune, cursor int) (line, col int) {
	for i := 0; i < cursor && i < len(buf); i++ {
		if buf[i] == '\n' {
			line++
			col = 0
		} else {
			col++
		}
	}
	return
}

// lineStart returns the buffer index of the start of the line containing the cursor
func lineStart(buf []rune, cursor int) int {
	for i := cursor - 1; i >= 0; i-- {
		if buf[i] == '\n' {
			return i + 1
		}
	}
	return 0
}

// lineEnd returns the buffer index of the end of the line containing the cursor
func lineEnd(buf []rune, cursor int) int {
	for i := cursor; i < len(buf); i++ {
		if buf[i] == '\n' {
			return i
		}
	}
	return len(buf)
}

// redraw clears the input area and redraws the buffer, positioning the terminal
// cursor to match the buffer cursor. prevCurLine is the display line the terminal
// cursor was on before this call. Returns the new cursor display line.
func (r *MultilineReader) redraw(buf []rune, cursor int, prevCurLine int) int {
	curLine, curCol := cursorPos(buf, cursor)
	lines := strings.Split(string(buf), "\n")

	var out strings.Builder

	// Move terminal cursor to start of input area
	if prevCurLine > 0 {
		fmt.Fprintf(&out, "\033[%dA", prevCurLine)
	}
	out.WriteString("\r\033[J") // column 0, clear to end of screen

	// Print all lines with prompt/continuation prefixes
	for i, line := range lines {
		if i > 0 {
			out.WriteString("\r\n")
		}
		if i == 0 {
			out.WriteString("> ")
		} else {
			out.WriteString("  ")
		}
		out.WriteString(line)
	}

	// Position terminal cursor at the buffer cursor location
	lastLine := len(lines) - 1
	if lastLine > curLine {
		fmt.Fprintf(&out, "\033[%dA", lastLine-curLine)
	}
	fmt.Fprintf(&out, "\r\033[%dC", curCol+2) // +2 for "> " or "  " prefix

	fmt.Print(out.String())
	return curLine
}

// ReadMultiline reads multiline input from the terminal with full line editing.
// Enter adds a newline, Ctrl+D submits. Supports cursor movement (Left/Right,
// Home/End, Ctrl+A/E), input history (Up/Down), and editing (Ctrl+W, Ctrl+U, Delete).
func (r *MultilineReader) ReadMultiline() (string, error) {
	if !term.IsTerminal(r.fd) {
		return r.readLineFallback()
	}

	oldState, err := term.MakeRaw(r.fd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[raw mode failed: %v]\n", err)
		return r.readLineFallback()
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	restore := func() { term.Restore(r.fd, oldState) }
	defer restore()

	go func() {
		<-sigCh
		restore()
		os.Exit(0)
	}()

	var buf []rune
	cursor := 0
	curLine := 0 // display line the terminal cursor is on

	r.histIdx = len(r.history)
	r.saved = ""

	readByte := func() byte {
		b := make([]byte, 1)
		os.Stdin.Read(b)
		return b[0]
	}

	for {
		b := make([]byte, 1)
		_, err := os.Stdin.Read(b)
		if err != nil {
			return string(buf), err
		}

		char := b[0]

		switch {
		case char == 0x1b: // ESC - start of escape sequence
			next := readByte()
			if next == '[' {
				// CSI sequence: ESC [ <params> <final>
				var params []byte
				for {
					p := readByte()
					if p >= 0x40 && p <= 0x7E {
						switch p {
						case 'A': // Up - history previous
							if len(r.history) > 0 && r.histIdx > 0 {
								if r.histIdx == len(r.history) {
									r.saved = string(buf)
								}
								r.histIdx--
								buf = []rune(r.history[r.histIdx])
								cursor = len(buf)
								curLine = r.redraw(buf, cursor, curLine)
							}
						case 'B': // Down - history next
							if r.histIdx < len(r.history) {
								r.histIdx++
								if r.histIdx == len(r.history) {
									buf = []rune(r.saved)
								} else {
									buf = []rune(r.history[r.histIdx])
								}
								cursor = len(buf)
								curLine = r.redraw(buf, cursor, curLine)
							}
						case 'C': // Right
							if cursor < len(buf) {
								cursor++
								curLine = r.redraw(buf, cursor, curLine)
							}
						case 'D': // Left
							if cursor > 0 {
								cursor--
								curLine = r.redraw(buf, cursor, curLine)
							}
						case 'H': // Home
							start := lineStart(buf, cursor)
							if cursor != start {
								cursor = start
								curLine = r.redraw(buf, cursor, curLine)
							}
						case 'F': // End
							end := lineEnd(buf, cursor)
							if cursor != end {
								cursor = end
								curLine = r.redraw(buf, cursor, curLine)
							}
						case '~': // Extended keys
							switch string(params) {
							case "3": // Delete
								if cursor < len(buf) {
									buf = append(buf[:cursor], buf[cursor+1:]...)
									curLine = r.redraw(buf, cursor, curLine)
								}
							case "1": // Home (alternate)
								start := lineStart(buf, cursor)
								if cursor != start {
									cursor = start
									curLine = r.redraw(buf, cursor, curLine)
								}
							case "4": // End (alternate)
								end := lineEnd(buf, cursor)
								if cursor != end {
									cursor = end
									curLine = r.redraw(buf, cursor, curLine)
								}
							}
						}
						break
					}
					params = append(params, p)
				}
			} else if next == 'O' {
				// SS3 sequence
				p := readByte()
				switch p {
				case 'H': // Home
					start := lineStart(buf, cursor)
					if cursor != start {
						cursor = start
						curLine = r.redraw(buf, cursor, curLine)
					}
				case 'F': // End
					end := lineEnd(buf, cursor)
					if cursor != end {
						cursor = end
						curLine = r.redraw(buf, cursor, curLine)
					}
				}
			}

		case char == 0x04: // Ctrl+D - submit
			fmt.Print("\r\n")
			result := string(buf)
			if strings.TrimSpace(result) != "" {
				r.history = append(r.history, result)
			}
			return result, nil

		case char == 0x03: // Ctrl+C - cancel
			fmt.Print("^C\r\n")
			return "", nil

		case char == 0x01: // Ctrl+A - start of line
			start := lineStart(buf, cursor)
			if cursor != start {
				cursor = start
				curLine = r.redraw(buf, cursor, curLine)
			}

		case char == 0x05: // Ctrl+E - end of line
			end := lineEnd(buf, cursor)
			if cursor != end {
				cursor = end
				curLine = r.redraw(buf, cursor, curLine)
			}

		case char == 0x15: // Ctrl+U - clear to start of line
			start := lineStart(buf, cursor)
			if cursor > start {
				buf = append(buf[:start], buf[cursor:]...)
				cursor = start
				curLine = r.redraw(buf, cursor, curLine)
			}

		case char == 0x17: // Ctrl+W - delete word backward
			if cursor > 0 {
				end := cursor
				// Skip spaces backward (stop at newline)
				for cursor > 0 && buf[cursor-1] == ' ' {
					cursor--
				}
				// Skip non-spaces backward (stop at space or newline)
				for cursor > 0 && buf[cursor-1] != ' ' && buf[cursor-1] != '\n' {
					cursor--
				}
				if cursor < end {
					buf = append(buf[:cursor], buf[end:]...)
					curLine = r.redraw(buf, cursor, curLine)
				}
			}

		case char == '\r' || char == '\n': // Enter
			trimmed := strings.TrimSpace(string(buf))
			if isImmediateCommand(trimmed) {
				fmt.Print("\r\n")
				result := string(buf)
				if strings.TrimSpace(result) != "" {
					r.history = append(r.history, result)
				}
				return result, nil
			}
			// Insert newline at cursor position
			buf = append(buf, 0)
			copy(buf[cursor+1:], buf[cursor:])
			buf[cursor] = '\n'
			cursor++
			curLine = r.redraw(buf, cursor, curLine)

		case char == 0x7f || char == 0x08: // Backspace
			if cursor > 0 {
				buf = append(buf[:cursor-1], buf[cursor:]...)
				cursor--
				curLine = r.redraw(buf, cursor, curLine)
			}

		default: // Regular character
			buf = append(buf, 0)
			copy(buf[cursor+1:], buf[cursor:])
			buf[cursor] = rune(char)
			cursor++
			curLine = r.redraw(buf, cursor, curLine)
		}
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
	ActionMemory                              // Toggle memory mode
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

	// Memory toggle
	if input == "/memory" {
		state.AmnesiaMode = !state.AmnesiaMode
		msg := "*Memory enabled* - chat history will be sent to the model"
		if state.AmnesiaMode {
			msg = "*Memory disabled* - chat history will not be sent to the model"
		}
		return CommandResult{Action: ActionMemory, Message: msg}
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
- ` + "`Up/Down`" + ` - cycle through input history
- ` + "`Left/Right`" + ` - move cursor within input
- ` + "`Home/End`" + ` or ` + "`Ctrl+A/E`" + ` - jump to start/end of line
- ` + "`Ctrl+W`" + ` - delete word backward
- ` + "`Ctrl+U`" + ` - clear line to left of cursor

**Commands:**
- ` + "`/help`" + ` - show this help message
- ` + "`exit`" + ` or ` + "`quit`" + ` - end the session
- ` + "`/memory`" + ` - toggle memory mode (sends chat history to model)
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
	amnesiaMode := true
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

		// Check for memory toggle
		if input == "/memory" {
			amnesiaMode = !amnesiaMode
			if !amnesiaMode {
				RenderWithGlamourPtr("*Memory enabled* - chat history will be sent to the model")
			} else {
				RenderWithGlamourPtr("*Memory disabled* - chat history will not be sent to the model")
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
