package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func checkCarriageReturns() bool {
	// read all the .go files in this directory
	// check for ^M
	// if found print the filename
	// if any are found return true
	// if not are found return false

	files, err := os.ReadDir(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading directory: %v\n", err)
		return false
	}

	foundCarriageReturns := false

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filename := file.Name()
		if !strings.HasSuffix(filename, ".go") {
			continue
		}

		data, err := os.ReadFile(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", filename, err)
			continue
		}

		if strings.Contains(string(data), "\r") {
			fmt.Printf("%s: has carriage returns\n", filename)
			foundCarriageReturns = true
		}
	}

	return foundCarriageReturns
}

func getReadme() []string {
	data, err := os.ReadFile("README.md")
	if err != nil {
		return []string{}
	}
	return strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
}

func filletHelp(lines []string) []string {
	var result []string
	inCodeBlock := false
	var currentBlock []string

	for _, line := range lines {
		// Check if line starts with ``` to handle code blocks with or without language specifiers
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			if inCodeBlock {
				// End of code block, add to result if not empty
				if len(currentBlock) > 0 {
					result = append(result, strings.Join(currentBlock, "\n"))
					currentBlock = []string{}
				}
			} else {
				// Start of code block
				currentBlock = []string{}
			}
			inCodeBlock = !inCodeBlock
		} else if inCodeBlock {
			// Inside a code block, add line to current block
			currentBlock = append(currentBlock, line)
		}
	}

	// Add the last block if it wasn't closed
	if inCodeBlock && len(currentBlock) > 0 {
		result = append(result, strings.Join(currentBlock, "\n"))
	}

	// Filter to only include multi-line code blocks
	var filteredResult []string
	for _, block := range result {
		if strings.Contains(block, "\n") {
			filteredResult = append(filteredResult, block)
		}
	}

	return filteredResult
}

func handleOpts(argv []string, argc int) (optionsStruct, error) {
	var opts optionsStruct

	opts.useGemini, opts.usePerplexity, opts.useChatGPT, opts.useCerebras = false, false, false, false
	opts.logToJsonl = false
	opts.readLog = false

	// utility function for clarity
	anyModelsSpecified := func(opt optionsStruct) bool {
		return opts.useChatGPT || opts.useGemini || opts.usePerplexity || opts.useCerebras
	}

	// loop through each arg
	for idx, each := range argv {
		if strings.Contains(each, "-rl") {
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

		if strings.Contains(each, "-h") {
			opts.printUsage = true
		}

		if strings.Contains(each, "-t") {
			opts.printAPIKeys = true
		}

		if strings.Contains(each, "-lg") {
			opts.listGeminiModels = true
		}

		if strings.Contains(each, "-lc") {
			opts.listOpenAIModels = true
		}

		if strings.Contains(each, "-q") {
			opts.quietMode = true

			if opts.logToJsonl {
				opts.logToJsonl = false
			}
		}

		if strings.Contains(each, "-l") {
			if opts.quietMode {
				fmt.Fprintf(os.Stderr, "Ignoring logging arg as quiet mode activated\n")
			} else {
				opts.logToJsonl = true
			}
		}

		if strings.Contains(each, "-c") {
			opts.useChatGPT = true
		}

		if strings.Contains(each, "-g") {
			opts.useGemini = true
		}

		if strings.Contains(each, "-p") {
			opts.usePerplexity = true
		}

		if strings.Contains(each, "-f") {
			opts.useCerebras = true
		}

		// Cerebras and ChatGPT combo
		if strings.Contains(each, "-b") {
			opts.useCerebras = true
			opts.useChatGPT = true
		}
	}

	// If we specify a model and readlog is set we want an error
	if anyModelsSpecified(opts) && opts.readLog {
		return opts, fmt.Errorf("-rl provided with a model specifier flag, please provide just one or the other")
	}

	// Functionality for including the prompt as an argument
	// If we specify a model OR no models are specified
	if anyModelsSpecified(opts) || !anyModelsSpecified(opts) {
		// and we have not set readLog
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
	}

	// If none explicitly selected
	if !anyModelsSpecified(opts) {
		// and we have not set readLog
		if !opts.readLog {
			// then use all models
			opts.useChatGPT, opts.useGemini, opts.usePerplexity, opts.useCerebras = true, true, true, true
		}
	}

	// success
	return opts, nil
}
