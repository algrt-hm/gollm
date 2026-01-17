package main

import (
	"github.com/openai/openai-go/v2/shared"
)

// From Cerebras docs:
//
// See https://inference-docs.cerebras.ai/introduction
// https://inference-docs.cerebras.ai/models/overview
// https://inference-docs.cerebras.ai/api-reference/models/list-models
//
// Available models (from API): gpt-oss-120b, llama-3.3-70b, llama3.1-8b,
// qwen-3-32b, qwen-3-235b-a22b-instruct-2507, zai-glm-4.6, zai-glm-4.7

const cerebrasModel = "gpt-oss-120b"

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

// AvailableModels lists models available for each provider in interactive mode
var AvailableModels = struct {
	Claude    []string
	ChatGPT   []string
	Gemini    []string
	Cerebras  []string
	Perplexity []string
}{
	Claude: []string{
		"claude-sonnet-4-5-20250929",
		"claude-opus-4-20250514",
		"claude-3-5-haiku-20241022",
	},
	ChatGPT: []string{
		"gpt-5.2",
		"gpt-4.1",
		"gpt-4.1-mini",
		"o3",
		"o4-mini",
	},
	Gemini: []string{
		"models/gemini-3-pro-preview",
		"models/gemini-2.5-pro-preview-06-05",
		"models/gemini-2.5-flash-preview-05-20",
	},
	Cerebras: []string{
		"gpt-oss-120b",
		"llama-3.3-70b",
		"llama3.1-8b",
		"qwen-3-32b",
		"qwen-3-235b-a22b-instruct-2507",
		"zai-glm-4.6",
		"zai-glm-4.7",
	},
	Perplexity: []string{
		"sonar-pro",
		"sonar",
		"sonar-reasoning-pro",
		"sonar-reasoning",
	},
}
