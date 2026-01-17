package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ChatMessage represents a single message in the conversation
type ChatMessage struct {
	Role    string // "user" or "assistant"
	Content string
}

// InteractiveSession runs an interactive chat session with the selected model
func InteractiveSession(o optionsStruct) {
	var modelName string
	var chatFunc func(history []ChatMessage, userInput string) (string, error)

	// Determine which model to use
	switch {
	case o.useClaude:
		modelName = "Claude (" + DefaultModels.Claude + ")"
		chatFunc = claudeChat
	case o.useChatGPT:
		modelName = "ChatGPT (" + string(DefaultModels.ChatGPT) + ")"
		chatFunc = chatGPTChat
	case o.useGemini:
		modelName = "Gemini (" + DefaultModels.Gemini + ")"
		chatFunc = geminiChat
	case o.useCerebras:
		modelName = "Cerebras (" + DefaultModels.Cerebras + ")"
		chatFunc = cerebrasChat
	case o.usePerplexity:
		modelName = "Perplexity (" + DefaultModels.Perplexity + ")"
		chatFunc = perplexityChat
	default:
		Fatalf("No model selected for interactive mode")
	}

	fmt.Printf("Interactive chat with %s\n", modelName)
	fmt.Println("Type 'exit' or 'quit' to end the session, or press Ctrl+C")
	fmt.Println("---")

	var history []ChatMessage
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("\nYou: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("\nGoodbye!")
			return
		}

		input = strings.TrimSpace(input)

		// Check for exit commands
		if input == "exit" || input == "quit" {
			fmt.Println("Goodbye!")
			return
		}

		// Skip empty input
		if input == "" {
			continue
		}

		// Get response from model
		response, err := chatFunc(history, input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}

		// Add to history
		history = append(history, ChatMessage{Role: "user", Content: input})
		history = append(history, ChatMessage{Role: "assistant", Content: response})

		// Display response
		fmt.Printf("\nAssistant: %s\n", response)
	}
}
