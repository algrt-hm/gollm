package main

import (
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/shared"
)

// Where we store default model info
var DefaultModels = struct {
	Perplexity string
	Gemini     string
	ChatGPT    shared.ChatModel
	Cerebras   string
}{
	Perplexity: "sonar-pro",
	Gemini:     "models/gemini-2.5-pro",
	ChatGPT:    openai.ChatModelGPT4o,
	// From Cerebras docs:
	// See https://inference-docs.cerebras.ai/introduction
	// Our free tier supports a context length of 8,192 tokens
	// For all supported models, we also offer context lengths up to 128K upon request
	Cerebras: "llama-4-scout-17b-16e-instruct",
}
