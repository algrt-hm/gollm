package main

import (
	"fmt"
	"testing"
)

func TestPrintAPIKeys(t *testing.T) {
	PrintAPIKeys()
}

func TestCheckInternetHTTP(t *testing.T) {
	ret, err := CheckInternetHTTP()
	if ret {
		fmt.Println("Connected to internet")
	} else {
		fmt.Printf("Connected to internet %v: %v", ret, err)
	}
}

func TestPrintUsage(t *testing.T) {
	PrintUsage(true, getLogPath())
}

func TestHandleOpts(t *testing.T) {

	prompt := "Please tell me about yourself"

	// Test that when no -rl the last argument can be considered a prompt
	expected := optionsStruct{
		useGemini:     true,
		usePerplexity: true,
		useChatGPT:    true,
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

	// Prompt as only arg
	// all models as none selected through args
	argv := []string{prompt}
	fmt.Printf("argv: %v\n", argv)

	ret := handleOpts(argv, len(argv))
	if ret != expected {
		t.Errorf("\nExpected %+v\nGot %+v", expected, ret)
	}

	// Test that when -rl the last argument can not be considered a prompt
	// we expect 
	// readLog to be true
	// readLogIdx == 10
	// promptText == ""

	expected = optionsStruct{
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
	argv = []string{"-rl", "10"}
	fmt.Printf("argv: %v\n", argv)

	ret = handleOpts(argv, len(argv))
	if ret != expected {
		t.Errorf("\nExpected %+v\nGot %+v", expected, ret)
	}

	// If we want ChatGPT and Gemini ...
	expectedPrompt := "Expected prompt"

	expected = optionsStruct{
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
	argv = []string{"-c", "-g", expectedPrompt}
	fmt.Printf("argv: %v\n", argv)

	ret = handleOpts(argv, len(argv))
	if ret != expected {
		t.Errorf("\nExpected %+v\nGot %+v", expected, ret)
	}

}
