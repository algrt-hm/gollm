package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestFilletHelp(t *testing.T) {
	// Here we essentially test that the README.md example of gollm -h updated and correct
	// Set all API keys so the dynamic "Setup:" section matches the README
	envVars := map[string]string{
		"PERPLEXITY_API_KEY": "test",
		"OPENAI_API_KEY":     "test",
		"GEMINI_API_KEY":     "test",
		"CEREBRAS_API_KEY":   "test",
		"ANTHROPIC_API_KEY":  "test",
		"LLM_PROXY_URL":     "http://localhost:8000/v1",
	}
	for k, v := range envVars {
		key := k // capture for closure
		prev, had := os.LookupEnv(key)
		os.Setenv(key, v)
		if had {
			prevVal := prev // capture for closure
			t.Cleanup(func() { os.Setenv(key, prevVal) })
		} else {
			t.Cleanup(func() { os.Unsetenv(key) })
		}
	}

	ret := filletHelp(getReadme())

	// check that ret is a single slice
	if len(ret) != 1 {
		t.Errorf("Expected 1 slice, got %d", len(ret))
	}

	// check that the slice is non-empty
	if len(ret[0]) == 0 {
		t.Errorf("Expected non-empty slice, got empty slice")
	}

	helpFromReadme := ret[0]
	helpFromCode := PrintUsage(true, getLogPath())

	// remove leading and trailing whitespace
	helpFromReadme = strings.TrimSpace(helpFromReadme)
	helpFromCode = strings.TrimSpace(helpFromCode)

	// run through both strings and print the first rune where they differ, with 10 runes of context either side
	runesReadme := []rune(helpFromReadme)
	runesCode := []rune(helpFromCode)

	minLen := len(runesReadme)
	if len(runesCode) < minLen {
		minLen = len(runesCode)
	}

	for i := 0; i < minLen; i++ {
		if runesReadme[i] != runesCode[i] {
			start := i - 10
			if start < 0 {
				start = 0
			}
			end := i + 10
			if end > minLen {
				end = minLen
			}
			fmt.Printf("First difference at rune position %d\n", i)
			// show runes as they are and with integer representation
			fmt.Printf("rune at issue readme: %c code: %c\n", runesReadme[i], runesCode[i])
			fmt.Printf("rune at issue readme: %d code: %d\n", runesReadme[i], runesCode[i])
			fmt.Println("Context:")
			fmt.Printf("Readme:\n%s\n", string(runesReadme[start:end]))
			fmt.Printf("Code:\n%s\n", string(runesCode[start:end]))
			break
		}
	}

	if helpFromReadme != helpFromCode {
		t.Errorf("Expected help from readme to match help from code, help from readme is:\n%s\nhelp from code is:\n%s", helpFromReadme, helpFromCode)
	}
}

func TestPrintUsageOfflineStatusLine(t *testing.T) {
	usage := PrintUsage(false, getLogPath())
	if !strings.Contains(usage, "- You are offline (or internet connectivity could not be verified)") {
		t.Errorf("Expected offline status line in usage output, got:\n%s", usage)
	}
}

func TestHandleOpts(t *testing.T) {
	// utility to print argv for development
	printArgv := func(argv []string) {
		fmt.Printf("argv: %v\n", argv)
	}

	if testingSuppressOutput {
		printArgv = func(argv []string) {
			// don't print
		}
	}

	t.Run("prompt_only_argument", func(t *testing.T) {
		prompt := "Please tell me about yourself"

		// Test that when no -rl the last argument can be considered a prompt
		expected := optionsStruct{
			useGemini:     true,
			usePerplexity: true,
			useChatGPT:    true,
			useCerebras:   true,
			useClaude:     true,
			logToJsonl:    false,
			quietMode:     false,

			readLog:    false,
			readLogIdx: 0,

			printUsage:       false,
			printAPIKeys:     false,
			listGeminiModels: false,
			listOpenAIModels: false,

			promptText: prompt,
		}

		// Prompt as only arg
		// all models as none selected through args
		argv := []string{prompt}
		printArgv(argv)

		ret, err := handleOpts(argv, len(argv))
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if ret != expected {
			t.Errorf("\nExpected %+v\nGot %+v", expected, ret)
		}
	})

	t.Run("read_log_with_index", func(t *testing.T) {
		// Test that when -rl the last argument can not be considered a prompt
		// we expect
		// readLog to be true
		// readLogIdx == 10
		// promptText == ""

		expected := optionsStruct{
			useGemini:     false,
			usePerplexity: false,
			useChatGPT:    false,
			useCerebras:   false,
			logToJsonl:    false,
			quietMode:     false,

			readLog:    true,
			readLogIdx: 10,

			printUsage:       false,
			printAPIKeys:     false,
			listGeminiModels: false,
			listOpenAIModels: false,

			promptText: "",
		}

		// -rl with idx 10
		argv := []string{"-rl", "10"}
		printArgv(argv)

		ret, err := handleOpts(argv, len(argv))
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if ret != expected {
			t.Errorf("\nExpected %+v\nGot %+v", expected, ret)
		}
	})

	t.Run("multiple_model_flags_with_prompt", func(t *testing.T) {
		// If we want ChatGPT and Gemini ...
		expectedPrompt := "Expected prompt"

		expected := optionsStruct{
			useGemini:     true,
			usePerplexity: false,
			useChatGPT:    true,
			useCerebras:   false,
			logToJsonl:    false,
			quietMode:     false,

			readLog:    false,
			readLogIdx: 0,

			printUsage:       false,
			printAPIKeys:     false,
			listGeminiModels: false,
			listOpenAIModels: false,

			promptText: expectedPrompt,
		}

		// -c and -g with prompt
		argv := []string{"-c", "-g", expectedPrompt}
		printArgv(argv)

		ret, err := handleOpts(argv, len(argv))
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if ret != expected {
			t.Errorf("\nExpected %+v\nGot %+v", expected, ret)
		}
	})

	t.Run("read_log_without_index_is_fine", func(t *testing.T) {
		// -rl on its own should is fine
		argv := []string{"-rl"}
		printArgv(argv)

		expected := optionsStruct{
			useGemini:     false,
			usePerplexity: false,
			useChatGPT:    false,
			useCerebras:   false,
			logToJsonl:    false,
			quietMode:     false,

			readLog:    true,
			readLogIdx: -1,

			printUsage:       false,
			printAPIKeys:     false,
			listGeminiModels: false,
			listOpenAIModels: false,

			promptText: "",
		}

		ret, err := handleOpts(argv, len(argv))
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if ret != expected {
			t.Errorf("\nExpected %+v\nGot %+v", expected, ret)
		}
	})

	t.Run("read_log_with_model_flag_should_error", func(t *testing.T) {
		// -rl with an index with a model is somewhat confusing so should result in an error
		argv := []string{"-rl", "1", "-c"}
		printArgv(argv)

		_, err := handleOpts(argv, len(argv))
		expectedErr := fmt.Errorf("-rl provided with a model specifier flag, please provide just one or the other")
		if err == nil || err.Error() != expectedErr.Error() {
			t.Errorf("\nExpected error %+v\nGot %+v", expectedErr, err)
		}
	})

	t.Run("single_string_arg_with_a_dash_is_a_prompt", func(t *testing.T) {
		// this should be handled ok and not regard the dash as a flag
		prompt := "How should I think about objectively researching the best technology start-ups in the UK?"
		argv := []string{prompt}
		printArgv(argv)

		expected := optionsStruct{
			useGemini:     true,
			usePerplexity: true,
			useChatGPT:    true,
			useCerebras:   true,
			useClaude:     true,
			logToJsonl:    false,
			quietMode:     false,

			readLog:    false,
			readLogIdx: 0,

			printUsage:       false,
			printAPIKeys:     false,
			listGeminiModels: false,
			listOpenAIModels: false,

			promptText: prompt,
		}

		ret, err := handleOpts(argv, len(argv))
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if ret != expected {
			t.Errorf("\nExpected %+v\nGot %+v", expected, ret)
		}

	})

	t.Run("potential_flag_in_prompt_as_last_arg", func(t *testing.T) {
		// gollm -f 'How to get not-a-time value for Pandas in python'
		// previously yielded the API keys so this is the test for the fix

		prompt := "How to get not-a-time value for Pandas in python"

		argv := []string{"-f", prompt}
		printArgv(argv)

		expected := optionsStruct{
			useGemini:     false,
			usePerplexity: false,
			useChatGPT:    false,
			useCerebras:   true,
			logToJsonl:    false,
			quietMode:     false,

			readLog:    false,
			readLogIdx: 0,

			printUsage:       false,
			printAPIKeys:     false,
			listGeminiModels: false,
			listOpenAIModels: false,

			promptText: prompt,
		}

		ret, err := handleOpts(argv, len(argv))
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if ret != expected {
			t.Errorf("\nExpected %+v\nGot %+v", expected, ret)
		}
	})

	t.Run("use_gemini", func(t *testing.T) {
		// gollm -g 'Does MCP work with load-balancers etc?'

		prompt := "Does MCP work with load-balancers etc?"

		argv := []string{"-g", prompt}
		printArgv(argv)

		expected := optionsStruct{
			useGemini:     true,
			usePerplexity: false,
			useChatGPT:    false,
			useCerebras:   false,
			logToJsonl:    false,
			quietMode:     false,

			readLog:    false,
			readLogIdx: 0,

			printUsage:       false,
			printAPIKeys:     false,
			listGeminiModels: false,
			listOpenAIModels: false,

			promptText: prompt,
		}

		ret, err := handleOpts(argv, len(argv))
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if ret != expected {
			t.Errorf("\nExpected %+v\nGot %+v", expected, ret)
		}
	})
}

func TestNoArgsShowsUsage(t *testing.T) {
	// gollm with no arguments should show usage
	// This is handled in main() before handleOpts is called, so we test the
	// built binary directly
	out, err := exec.Command("./gollm").CombinedOutput()
	if err != nil {
		t.Fatalf("Expected gollm with no args to exit cleanly, got error: %v", err)
	}
	output := string(out)
	if !strings.Contains(output, "show (this) help") {
		t.Errorf("Expected usage output, got:\n%s", output)
	}
}

func TestCheckCarriageReturns(t *testing.T) {
	// check testSillyWin returns false
	// This test assumes there are no .go files with carriage returns in the current directory.
	got := checkCarriageReturns()
	if got {
		t.Errorf("Expected checkCarriageReturns to return false, got true")
	}
}

func TestInteractiveModeFlags(t *testing.T) {
	t.Run("logging_then_interactive_ignores_logging", func(t *testing.T) {
		// gollm -l -i should enter interactive mode with logging disabled
		argv := []string{"-l", "-i"}

		ret, err := handleOpts(argv, len(argv))
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if !ret.interactive {
			t.Errorf("Expected interactive to be true")
		}
		if ret.logToJsonl {
			t.Errorf("Expected logToJsonl to be false in interactive mode")
		}
	})

	t.Run("interactive_then_logging_ignores_logging", func(t *testing.T) {
		// gollm -i -l should also enter interactive mode with logging disabled
		argv := []string{"-i", "-l"}

		ret, err := handleOpts(argv, len(argv))
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if !ret.interactive {
			t.Errorf("Expected interactive to be true")
		}
		if ret.logToJsonl {
			t.Errorf("Expected logToJsonl to be false in interactive mode")
		}
	})

	t.Run("interactive_with_single_model_flag", func(t *testing.T) {
		// gollm -i -s should use Claude in interactive mode
		argv := []string{"-i", "-s"}

		ret, err := handleOpts(argv, len(argv))
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if !ret.interactive {
			t.Errorf("Expected interactive to be true")
		}
		if !ret.useClaude {
			t.Errorf("Expected useClaude to be true")
		}
	})

	t.Run("interactive_with_multiple_models_should_error", func(t *testing.T) {
		// gollm -i -s -c should error (multiple models)
		argv := []string{"-i", "-s", "-c"}

		_, err := handleOpts(argv, len(argv))
		if err == nil {
			t.Errorf("Expected error for multiple models in interactive mode")
		}
	})

	t.Run("interactive_with_combo_flag_should_error", func(t *testing.T) {
		// gollm -i -b should error (-b sets both Claude and ChatGPT)
		argv := []string{"-i", "-b"}

		_, err := handleOpts(argv, len(argv))
		if err == nil {
			t.Errorf("Expected error for -b combo flag in interactive mode")
		}
	})
}
