package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func isFlag(s string, arg string) bool {
	// note this may only work on ascii input

	// first strip whitespace from s
	s2 := strings.TrimSpace(s)

	// test it's the same
	return s2 == arg
}

func handleOpts(argv []string, argc int) (optionsStruct, error) {
	var opts optionsStruct

	opts.useGemini, opts.usePerplexity, opts.useChatGPT, opts.useCerebras, opts.useClaude, opts.useDeepseek = false, false, false, false, false, false
	opts.logToJsonl = false
	opts.readLog = false

	// utility function for clarity (-x is a bypass flag, not a model)
	anyModelsSpecified := func(opt optionsStruct) bool {
		return opt.useChatGPT || opt.useGemini || opt.usePerplexity || opt.useCerebras || opt.useClaude || opt.useDeepseek
	}

	// loop through each arg
	for idx, each := range argv {
		if isFlag(each, "-rl") {
			opts.readLog = true
			// negative means print all
			var logIdx = -1
			// look ahead and see if the next argument can be an integer
			// if it can be, that's our idx for ReadLogIdx

			// if there is a next arg
			if idx+1 < argc {
				// try to atoi the next arg
				intArg, err := strconv.Atoi(argv[idx+1])
				// if successful use it
				if err == nil {
					logIdx = intArg
					// otherwise we need to raise an error
				} else {
					Fatalf("Invalid log index: %s, error is %s", argv[idx+1], err.Error())
				}
			}
			opts.readLogIdx = logIdx
		}

		if isFlag(each, "-h") {
			opts.printUsage = true
		}

		if isFlag(each, "-t") {
			opts.printAPIKeys = true
		}

		if isFlag(each, "-lg") {
			opts.listGeminiModels = true
		}

		if isFlag(each, "-lc") {
			opts.listOpenAIModels = true
		}

		if isFlag(each, "-la") {
			opts.listAnthropicModels = true
		}

		if isFlag(each, "-q") {
			opts.quietMode = true

			if opts.logToJsonl {
				opts.logToJsonl = false
			}
		}

		if isFlag(each, "-l") {
			if opts.quietMode {
				fmt.Fprintf(os.Stderr, "Ignoring logging arg as quiet mode activated\n")
			} else if opts.interactive {
				fmt.Fprintf(os.Stderr, "Ignoring logging arg as interactive mode activated\n")
			} else {
				opts.logToJsonl = true
			}
		}

		if isFlag(each, "-c") {
			opts.useChatGPT = true
		}

		if isFlag(each, "-g") {
			opts.useGemini = true
		}

		if isFlag(each, "-p") {
			opts.usePerplexity = true
		}

		if isFlag(each, "-f") {
			opts.useCerebras = true
		}

		if isFlag(each, "-s") {
			opts.useClaude = true
		}

		if isFlag(each, "-d") {
			opts.useDeepseek = true
		}

		if isFlag(each, "-x") {
			opts.bypassLLMProxy = true
		}

		// Claude and ChatGPT combo
		if isFlag(each, "-b") {
			opts.useClaude = true
			opts.useChatGPT = true
		}

		if isFlag(each, "-i") {
			opts.interactive = true
			// Interactive mode is incompatible with logging
			if opts.logToJsonl {
				opts.logToJsonl = false
			}
		}
	}

	// If we specify a model and readlog is set we want an error
	if anyModelsSpecified(opts) && opts.readLog {
		return opts, fmt.Errorf("-rl provided with a model specifier flag, please provide just one or the other")
	}

	// Interactive mode requires exactly one model (-x is a bypass flag, not counted)
	if opts.interactive {
		modelCount := 0
		if opts.useChatGPT {
			modelCount++
		}
		if opts.useGemini {
			modelCount++
		}
		if opts.usePerplexity {
			modelCount++
		}
		if opts.useCerebras {
			modelCount++
		}
		if opts.useClaude {
			modelCount++
		}
		if opts.useDeepseek {
			modelCount++
		}
		if modelCount == 0 {
			// Default to first preferred provider with an API key
			preferred := GetPreferredProvider()
			switch preferred {
			case ProviderCerebras:
				opts.useCerebras = true
			case ProviderClaude:
				opts.useClaude = true
			case ProviderChatGPT:
				opts.useChatGPT = true
			case ProviderGemini:
				opts.useGemini = true
			case ProviderPerplexity:
				opts.usePerplexity = true
			case ProviderDeepseek:
				opts.useDeepseek = true
			}
		} else if modelCount > 1 {
			return opts, fmt.Errorf("interactive mode (-i) requires exactly one model, got %d", modelCount)
		}
	}

	// Functionality for including the prompt as an argument
	// (unless we have set readLog)
	if !opts.readLog {
		// take the last arg
		// and see if there is the flag character "-" in it
		// if there is not, we assume a prompt
		lastArg := argv[argc-1]
		lastArgContainsDash := strings.Contains(lastArg, "-")

		if !lastArgContainsDash {
			opts.promptText = lastArg
		} else {
			// impliedly there is a dash
			// the last arg may or may not be a prompt
			// let's see if the first rune is a dash
			lastArgAsRunes := []rune(lastArg)

			// if it's not, we can assume it's a prompt
			if len(lastArgAsRunes) > 0 && lastArgAsRunes[0] != '-' {
				opts.promptText = lastArg
			}
		}
	}

	// If none explicitly selected
	if !anyModelsSpecified(opts) {
		// and we have not set readLog or interactive mode
		if !opts.readLog && !opts.interactive {
			// then use all models; providers without API keys are skipped later in callModels
			opts.modelsImplied = true
			opts.useChatGPT, opts.useGemini, opts.usePerplexity, opts.useCerebras, opts.useClaude, opts.useDeepseek = true, true, true, true, true, true
		}
	}

	// success
	return opts, nil
}
