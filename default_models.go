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
// Available models (from API, 2026-07-21): gpt-oss-120b, zai-glm-4.7, gemma-4-31b

const cerebrasModel = "gpt-oss-120b"

// Where we store default model info
var DefaultModels = struct {
	Perplexity string
	Gemini     string
	ChatGPT    shared.ChatModel
	Cerebras   string
	Claude     string
	Deepseek   string
}{
	Perplexity: "sonar-pro",
	Gemini:     "models/gemini-3.1-pro-preview",
	ChatGPT:    "gpt-5.5",
	Cerebras:   cerebrasModel,
	Claude:     "claude-sonnet-5",
	Deepseek:   "deepseek-v4-pro",
}

// AvailableModels lists models available for each provider in interactive mode
// (verified against each provider's models API / docs on 2026-07-21)
var AvailableModels = struct {
	Claude     []string
	ChatGPT    []string
	Gemini     []string
	Cerebras   []string
	Perplexity []string
	Deepseek   []string
}{
	Claude: []string{
		"claude-sonnet-5",
		"claude-sonnet-4-6",
		"claude-opus-4-8",
		"claude-opus-4-6",
		"claude-haiku-4-5",
	},
	ChatGPT: []string{
		"gpt-5.5",
		"gpt-5.4",
		"gpt-5.4-mini",
		"gpt-5.2",
		"gpt-4.1",
		"o3",
		"o4-mini",
	},
	Gemini: []string{
		"models/gemini-3.1-pro-preview",
		"models/gemini-3.5-flash",
		"models/gemini-3.1-flash-lite",
		"models/gemini-2.5-pro",
		"models/gemini-2.5-flash",
	},
	Cerebras: []string{
		"gpt-oss-120b",
		"zai-glm-4.7",
		"gemma-4-31b",
	},
	Perplexity: []string{
		"sonar-pro",
		"sonar",
		"sonar-reasoning-pro",
		"sonar-deep-research",
	},
	Deepseek: []string{
		"deepseek-v4-pro",
		"deepseek-v4-flash",
	},
}
