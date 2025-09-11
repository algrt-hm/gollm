package main

import (
	"os"
	"strings"
)

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