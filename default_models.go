package main

import (
	"github.com/openai/openai-go/v2"
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

const cerebrasGptOss = "gpt-oss-120b"

// Where we store default model info
var DefaultModels = struct {
	Perplexity string
	Gemini     string
	ChatGPT    shared.ChatModel
	Cerebras   string
}{
	Perplexity: "sonar-pro",
	Gemini:     "models/gemini-2.5-pro",
	ChatGPT:    openai.ChatModelGPT5,
	Cerebras:   cerebrasGptOss,
}
