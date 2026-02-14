package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFormatConversationMarkdown(t *testing.T) {
	history := []ChatMessage{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
		{Role: "user", Content: "How are you?"},
		{Role: "assistant", Content: "I'm doing well, thanks!"},
	}

	result := formatConversationMarkdown(history, ProviderClaude, "claude-sonnet-4-5-20250929")

	// Check header
	if !strings.Contains(result, "# Conversation Log") {
		t.Error("Expected conversation log header")
	}

	// Check provider
	if !strings.Contains(result, "**Provider:** Claude") {
		t.Error("Expected provider in output")
	}

	// Check model
	if !strings.Contains(result, "**Model:** claude-sonnet-4-5-20250929") {
		t.Error("Expected model in output")
	}

	// Check messages
	if !strings.Contains(result, "## User") {
		t.Error("Expected User header")
	}
	if !strings.Contains(result, "## Assistant") {
		t.Error("Expected Assistant header")
	}
	if !strings.Contains(result, "Hello") {
		t.Error("Expected user message content")
	}
	if !strings.Contains(result, "Hi there!") {
		t.Error("Expected assistant message content")
	}
}

func TestSaveConversation(t *testing.T) {
	// Create a temp directory for testing
	tmpDir := t.TempDir()

	history := []ChatMessage{
		{Role: "user", Content: "Test message"},
		{Role: "assistant", Content: "Test response"},
	}

	t.Run("saves_with_full_path", func(t *testing.T) {
		filename := filepath.Join(tmpDir, "test_conversation.md")
		err := saveConversation(filename, history, ProviderChatGPT, "gpt-4")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Check file exists
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			t.Error("Expected file to be created")
		}

		// Check content
		content, err := os.ReadFile(filename)
		if err != nil {
			t.Errorf("Failed to read file: %v", err)
		}
		if !strings.Contains(string(content), "Test message") {
			t.Error("Expected file to contain test message")
		}
	})

	t.Run("adds_md_extension_if_missing", func(t *testing.T) {
		filename := filepath.Join(tmpDir, "no_extension")
		err := saveConversation(filename, history, ProviderChatGPT, "gpt-4")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Check file with .md extension exists
		expectedPath := filename + ".md"
		if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
			t.Error("Expected file with .md extension to be created")
		}
	})

	t.Run("does_not_double_md_extension", func(t *testing.T) {
		filename := filepath.Join(tmpDir, "already_has.md")
		err := saveConversation(filename, history, ProviderChatGPT, "gpt-4")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Check file exists without double extension
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			t.Error("Expected file to be created")
		}
		// Check no double extension
		doubleExt := filename + ".md"
		if _, err := os.Stat(doubleExt); !os.IsNotExist(err) {
			t.Error("Should not create file with double .md extension")
		}
	})
}

func TestHandleCommand(t *testing.T) {
	// Helper to create a fresh state for each test
	newState := func() *SessionState {
		return &SessionState{
			Provider:     ProviderClaude,
			CurrentModel: DefaultModels.Claude,
			History:      []ChatMessage{},
			AmnesiaMode:  true,
		}
	}

	t.Run("exit_commands", func(t *testing.T) {
		for _, cmd := range []string{"exit", "quit"} {
			state := newState()
			result := handleCommand(cmd, state)
			if result.Action != ActionExit {
				t.Errorf("%q: expected ActionExit, got %v", cmd, result.Action)
			}
		}
	})

	t.Run("help_command", func(t *testing.T) {
		state := newState()
		result := handleCommand("/help", state)
		if result.Action != ActionHelp {
			t.Errorf("expected ActionHelp, got %v", result.Action)
		}
	})

	t.Run("memory_toggle", func(t *testing.T) {
		state := newState()

		// First toggle - enable memory (amnesia was on by default)
		result := handleCommand("/memory", state)
		if result.Action != ActionMemory {
			t.Errorf("expected ActionMemory, got %v", result.Action)
		}
		if state.AmnesiaMode {
			t.Error("expected AmnesiaMode to be false (memory enabled)")
		}
		if !strings.Contains(result.Message, "Memory enabled") {
			t.Errorf("expected 'Memory enabled' in message, got %q", result.Message)
		}

		// Second toggle - disable memory (amnesia back on)
		result = handleCommand("/memory", state)
		if !state.AmnesiaMode {
			t.Error("expected AmnesiaMode to be true (memory disabled)")
		}
		if !strings.Contains(result.Message, "Memory disabled") {
			t.Errorf("expected 'Memory disabled' in message, got %q", result.Message)
		}
	})

	t.Run("model_list", func(t *testing.T) {
		state := newState()
		result := handleCommand("/model", state)
		if result.Action != ActionListModels {
			t.Errorf("expected ActionListModels, got %v", result.Action)
		}
	})

	t.Run("model_switch_valid", func(t *testing.T) {
		state := newState()
		result := handleCommand("/model 1", state)
		if result.Action != ActionSwitchModel {
			t.Errorf("expected ActionSwitchModel, got %v", result.Action)
		}
		if result.ModelIndex != 0 {
			t.Errorf("expected ModelIndex 0, got %d", result.ModelIndex)
		}
	})

	t.Run("model_switch_invalid", func(t *testing.T) {
		state := newState()
		result := handleCommand("/model 999", state)
		if result.Action != ActionError {
			t.Errorf("expected ActionError, got %v", result.Action)
		}
		if !strings.Contains(result.Message, "Invalid model number") {
			t.Errorf("expected error message about invalid model number")
		}
	})

	t.Run("model_switch_non_numeric", func(t *testing.T) {
		state := newState()
		result := handleCommand("/model abc", state)
		if result.Action != ActionError {
			t.Errorf("expected ActionError, got %v", result.Action)
		}
	})

	t.Run("provider_list", func(t *testing.T) {
		state := newState()
		result := handleCommand("/provider", state)
		if result.Action != ActionListProviders {
			t.Errorf("expected ActionListProviders, got %v", result.Action)
		}
	})

	t.Run("provider_switch_invalid", func(t *testing.T) {
		state := newState()
		result := handleCommand("/provider 999", state)
		if result.Action != ActionError {
			t.Errorf("expected ActionError, got %v", result.Action)
		}
	})

	t.Run("provider_switch_proxy_mode_skips_key_check", func(t *testing.T) {
		state := newState()
		state.ProxyMode = true
		// Pick a provider index that likely has no local API key set in test env;
		// with ProxyMode the switch should succeed regardless.
		result := handleCommand("/provider 1", state)
		if result.Action != ActionSwitchProvider {
			t.Errorf("expected ActionSwitchProvider in proxy mode, got %v (message: %s)", result.Action, result.Message)
		}
	})

	t.Run("save_with_no_history", func(t *testing.T) {
		state := newState()
		result := handleCommand("/save", state)
		if result.Action != ActionError {
			t.Errorf("expected ActionError (no history), got %v", result.Action)
		}
		if !strings.Contains(result.Message, "No conversation") {
			t.Errorf("expected message about no conversation")
		}
	})

	t.Run("save_with_history", func(t *testing.T) {
		state := newState()
		state.History = []ChatMessage{
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "hi"},
		}
		result := handleCommand("/save", state)
		if result.Action != ActionSave {
			t.Errorf("expected ActionSave, got %v", result.Action)
		}
	})

	t.Run("save_case_insensitive", func(t *testing.T) {
		state := newState()
		state.History = []ChatMessage{{Role: "user", Content: "test"}}

		for _, cmd := range []string{"/save", "/Save", "/SAVE", "/sAvE"} {
			result := handleCommand(cmd, state)
			if result.Action != ActionSave {
				t.Errorf("%q: expected ActionSave, got %v", cmd, result.Action)
			}
		}
	})

	t.Run("empty_input", func(t *testing.T) {
		state := newState()
		result := handleCommand("", state)
		if result.Action != ActionEmpty {
			t.Errorf("expected ActionEmpty, got %v", result.Action)
		}

		result = handleCommand("   ", state)
		if result.Action != ActionEmpty {
			t.Errorf("expected ActionEmpty for whitespace, got %v", result.Action)
		}
	})

	t.Run("regular_message", func(t *testing.T) {
		state := newState()
		result := handleCommand("hello world", state)
		if result.Action != ActionNone {
			t.Errorf("expected ActionNone (send to LLM), got %v", result.Action)
		}
	})

	t.Run("unknown_command", func(t *testing.T) {
		state := newState()
		result := handleCommand("/unknown", state)
		if result.Action != ActionNone {
			t.Errorf("expected ActionNone for unknown command, got %v", result.Action)
		}
	})

	// Mid-conversation tests - simulate a conversation then run commands
	t.Run("mid_conversation_help", func(t *testing.T) {
		state := newState()
		// Simulate some conversation history
		state.History = []ChatMessage{
			{Role: "user", Content: "What is Go?"},
			{Role: "assistant", Content: "Go is a programming language."},
			{Role: "user", Content: "Tell me more"},
			{Role: "assistant", Content: "It was created at Google."},
		}

		result := handleCommand("/help", state)
		if result.Action != ActionHelp {
			t.Errorf("expected ActionHelp mid-conversation, got %v", result.Action)
		}
		// History should be preserved
		if len(state.History) != 4 {
			t.Errorf("expected history to be preserved, got %d messages", len(state.History))
		}
	})

	t.Run("mid_conversation_memory_toggle", func(t *testing.T) {
		state := newState()
		state.History = []ChatMessage{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi!"},
		}

		// Enable memory (amnesia is on by default)
		result := handleCommand("/memory", state)
		if result.Action != ActionMemory {
			t.Errorf("expected ActionMemory, got %v", result.Action)
		}
		if state.AmnesiaMode {
			t.Error("expected memory to be enabled (amnesia off)")
		}
		// History should still be preserved (just not sent to model)
		if len(state.History) != 2 {
			t.Errorf("expected history preserved, got %d", len(state.History))
		}

		// Disable memory (amnesia back on)
		result = handleCommand("/memory", state)
		if !state.AmnesiaMode {
			t.Error("expected memory to be disabled (amnesia on)")
		}
		// History still preserved
		if len(state.History) != 2 {
			t.Errorf("expected history preserved after toggle, got %d", len(state.History))
		}
	})

	t.Run("mid_conversation_model_switch", func(t *testing.T) {
		state := newState()
		state.History = []ChatMessage{
			{Role: "user", Content: "Question 1"},
			{Role: "assistant", Content: "Answer 1"},
		}
		originalModel := state.CurrentModel

		result := handleCommand("/model 2", state)
		if result.Action != ActionSwitchModel {
			t.Errorf("expected ActionSwitchModel, got %v", result.Action)
		}
		// Model should change
		if state.CurrentModel == originalModel {
			t.Error("expected model to change")
		}
		// History should be preserved
		if len(state.History) != 2 {
			t.Errorf("expected history preserved, got %d", len(state.History))
		}
	})

	t.Run("mid_conversation_save", func(t *testing.T) {
		state := newState()
		state.History = []ChatMessage{
			{Role: "user", Content: "First message"},
			{Role: "assistant", Content: "First response"},
			{Role: "user", Content: "Second message"},
			{Role: "assistant", Content: "Second response"},
		}

		result := handleCommand("/save", state)
		if result.Action != ActionSave {
			t.Errorf("expected ActionSave mid-conversation, got %v", result.Action)
		}
	})

	t.Run("mid_conversation_regular_message", func(t *testing.T) {
		state := newState()
		state.History = []ChatMessage{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi!"},
		}

		result := handleCommand("Another question", state)
		if result.Action != ActionNone {
			t.Errorf("expected ActionNone (send to LLM), got %v", result.Action)
		}
		// History unchanged by handleCommand (actual addition happens in InteractiveSession)
		if len(state.History) != 2 {
			t.Errorf("expected history unchanged by handleCommand, got %d", len(state.History))
		}
	})

	t.Run("multiple_commands_sequence", func(t *testing.T) {
		state := newState()

		// Simulate a full session with multiple commands
		commands := []struct {
			input          string
			expectedAction CommandAction
		}{
			{"hello", ActionNone},           // Message to LLM
			{"/help", ActionHelp},           // Help
			{"/memory", ActionMemory},       // Enable memory
			{"another message", ActionNone}, // Message to LLM
			{"/model", ActionListModels},    // List models
			{"/memory", ActionMemory},       // Disable memory
			{"/provider", ActionListProviders},
		}

		// Add some history to simulate conversation
		state.History = []ChatMessage{
			{Role: "user", Content: "test"},
			{Role: "assistant", Content: "response"},
		}

		for _, cmd := range commands {
			result := handleCommand(cmd.input, state)
			if result.Action != cmd.expectedAction {
				t.Errorf("command %q: expected %v, got %v", cmd.input, cmd.expectedAction, result.Action)
			}
		}

		// After all commands, history should still be intact
		if len(state.History) != 2 {
			t.Errorf("expected history preserved through commands, got %d", len(state.History))
		}
	})
}

func TestSpinner(t *testing.T) {
	t.Run("can_start_and_stop", func(t *testing.T) {
		spinner := NewSpinner()

		// Start should not block
		spinner.Start()

		// Let it run briefly
		time.Sleep(100 * time.Millisecond)

		// Stop should not block or panic
		spinner.Stop()
	})

	t.Run("multiple_stops_are_safe", func(t *testing.T) {
		spinner := NewSpinner()
		spinner.Start()
		time.Sleep(50 * time.Millisecond)

		// Multiple stops should be safe
		spinner.Stop()
		spinner.Stop() // Should not panic
	})

	t.Run("stop_without_start_is_safe", func(t *testing.T) {
		spinner := NewSpinner()
		// Stop without start should not panic
		spinner.Stop()
	})
}
