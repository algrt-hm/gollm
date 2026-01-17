package main

import (
	"github.com/openai/openai-go/v2/shared"
)

// From Cerebras docs:
//
// See https://inference-docs.cerebras.ai/introduction
// Our free tier supports a context length of 8,192 tokens
// const cerebrasLlama = "llama-4-scout-17b-16e-instruct"
//
// https://inference-docs.cerebras.ai/models/overview
// 64k tokens context length on free tier so seems like a sensible default choice

const cerebrasModel = "zai-glm-4.7"

// Where we store default model info
var DefaultModels = struct {
	Perplexity string
	Gemini     string
	ChatGPT    shared.ChatModel
	Cerebras   string
	Claude     string
}{
	Perplexity: "sonar-pro",
	Gemini:     "models/gemini-3-pro-preview",
	ChatGPT:    "gpt-5.2",
	Cerebras:   cerebrasModel,
	Claude:     "claude-sonnet-4-5-20250929",
}
