package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// ModelInfo represents the information we need about a model
type ModelInfo struct {
	Name                       string
	DisplayName                string
	Description                string
	SupportedGenerationMethods []string
}

// ListGeminiModels will list Gemini models which are available
func ListGeminiModels() string {
	var builder strings.Builder

	// --- Get API Key ---
	apiKey := GetGeminiAPIKeyOrBail()

	// --- Set up the Gemini client ---
	ctx := context.Background()

	// Use option.WithAPIKey to authenticate with an API key
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		Fatalf("Failed to create client: %v", err)
	}

	// Ensure the client is closed when main function finishes
	defer client.Close()

	// --- Collect models in a single pass ---
	supportedModels, err := collectSupportedModels(ctx, client)
	if err != nil {
		Fatalf("Failed to collect models: %v", err)
	}

	// --- Format detailed output ---
	builder.WriteString("--- Available Models ---\n")
	for _, model := range supportedModels {
		fmt.Fprintf(&builder, "%s Display name: %s Supports: %v\n", model.Name, model.DisplayName, model.SupportedGenerationMethods)
		if model.Description != "" {
			fmt.Fprintf(&builder, "Description: %s\n", model.Description)
		} else {
			builder.WriteString("Description: (none)\n")
		}
		builder.WriteString("----------------------\n")
	}
	builder.WriteString("--- End of List ---\n")

	// --- Format bulleted list ---
	builder.WriteString("\n--- Models supporting generateContent ---\n")
	for _, model := range supportedModels {
		fmt.Fprintf(&builder, " - %s (%s)\n", model.Name, model.DisplayName)
	}
	builder.WriteString("--- End of bulleted list ---\n")

	return builder.String()
}

// collectSupportedModels iterates through all models once and returns those supporting generateContent
func collectSupportedModels(ctx context.Context, client *genai.Client) ([]ModelInfo, error) {
	var supportedModels []ModelInfo

	iter := client.ListModels(ctx)
	for {
		info, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate models: %w", err)
		}

		// Only collect models that support generateContent
		if strSliceContains(info.SupportedGenerationMethods, "generateContent") {
			model := ModelInfo{
				Name:                       info.Name,
				DisplayName:                info.DisplayName,
				Description:                info.Description,
				SupportedGenerationMethods: info.SupportedGenerationMethods,
			}
			supportedModels = append(supportedModels, model)
		}
	}

	return supportedModels, nil
}

// StringifyGeminiResponse is a helper function to print the response content
// it returns response, finishReason, safetyRating
func StringifyGeminiResponse(resp *genai.GenerateContentResponse, model string) (string, string, string) {
	var response strings.Builder
	var finishReason string = ""
	var safetyRating strings.Builder

	if resp == nil || len(resp.Candidates) == 0 {
		return "Received an empty response.", "", ""
	}
	// impliedly the response is not nil or of length 0

	// Iterate through candidates (usually just one for basic generation)
	for _, cand := range resp.Candidates {
		// Iterate through the parts of the content
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				if textPart, ok := part.(genai.Text); ok {
					response.WriteString(string(textPart))
				} else {
					// It's not genai.Text (could be ImageData, FunctionCall, etc.)
					Fatalf("Part is not genai.Text, it's type %T\n", part)
				}
			}
		} else {
			return "Candidate content is nil.", "", ""
		}

		// If there's a safety rating then stringify it
		if len(cand.SafetyRatings) > 0 {
			for i, each := range cand.SafetyRatings {
				if i > 0 {
					safetyRating.WriteString(", ") // Add separator
				}
				fmt.Fprintf(&safetyRating, "%+v", each)
			}
		}

		// Capture last finish reason
		if cand.FinishReason != genai.FinishReasonUnspecified {
			finishReason = fmt.Sprintf("%+v", cand.FinishReason)
		}
	}

	if finishReason == "" {
		finishReason = "None"
	}

	return response.String(), finishReason, safetyRating.String()
}

func MockGenerateContentResponse() *genai.GenerateContentResponse {
	// Create a mock text part
	mockTextPart := genai.Text("This is a mocked Gemini response.")

	// Create mock content containing the text part
	mockContent := &genai.Content{
		Parts: []genai.Part{mockTextPart},
		Role:  "model", // Typically the role is "model" for the response
	}

	// Create a mock candidate containing the content
	mockCandidate := &genai.Candidate{
		Content:      mockContent,
		FinishReason: genai.FinishReasonStop, // Example finish reason
		SafetyRatings: []*genai.SafetyRating{ // Example safety rating
			{
				Category:    genai.HarmCategoryHarassment,
				Probability: genai.HarmProbabilityNegligible,
			},
		},
	}

	// Create the mock response containing the candidate
	mockResponse := &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{mockCandidate},
		UsageMetadata: &genai.UsageMetadata{ // Example usage metadata
			PromptTokenCount:     10,
			CandidatesTokenCount: 20,
			TotalTokenCount:      30,
		},
	}

	return mockResponse
}

func GeminiCallAPI(modelName string, promptText string, ctx context.Context, client *genai.Client, mock bool) (*genai.GenerateContentResponse, error) {
	if mock {
		return MockGenerateContentResponse(), nil
	}
	// Select the model
	model := client.GenerativeModel(modelName)

	resp, err := model.GenerateContent(ctx, genai.Text(promptText))

	if err != nil {
		Fatalf("Failed to generate content: %v", err)
	}

	return resp, err
}

// GeminiLowerWrapper calls the Gemini API
func GeminiLowerWrapper(promptText string, ctx context.Context, client *genai.Client, mock bool, logToJsonl bool, quietMode bool) string {
	// Start the timer
	startTime := time.Now()
	modelName := DefaultModels.Gemini

	resp, err := GeminiCallAPI(modelName, promptText, ctx, client, mock)

	if err != nil {
		Fatalf("Some issue: %s", err)
	}

	buffer, finishReason, safetyRating := StringifyGeminiResponse(resp, modelName)
	totalTokenCount := resp.UsageMetadata.TotalTokenCount
	duration := time.Since(startTime)

	// Log successful model call only if logging is enabled
	if logToJsonl {
		logEntry := LogEntry{
			ModelName:     modelName,
			TotalTokens:   int(totalTokenCount),
			Duration:      duration.Seconds(),
			StopReason:    finishReason,
			PromptText:    promptText,
			ModelResponse: buffer,
			Timestamp:     time.Now(),
		}
		if err := WriteLogEntry(logEntry); err != nil {
			// Log error but don't fail the request
			fmt.Fprintf(os.Stderr, "Failed to write log entry: %v\n", err)
		}
	}

	if quietMode {
		return buffer
	}

	if safetyRating != "" {
		return fmt.Sprintf("\nModel: %s, %d tokens used, finished due to: %s, safety rating: %s, duration: %.3f seconds\n\n%s\n", modelName, totalTokenCount, finishReason, safetyRating, duration.Seconds(), buffer)
	} else {
		return fmt.Sprintf("\nModel: %s, %d tokens used, finished due to: %s, duration: %.3f seconds\n\n%s\n", modelName, totalTokenCount, finishReason, duration.Seconds(), buffer)
	}
}

func GeminiMiddleWrapper(promptText string, mock bool, logToJsonl bool, quietMode bool) string {
	// --- Get API Key ---
	apiKey := GetGeminiAPIKeyOrBail()

	// --- Set up the Gemini client ---
	ctx := context.Background()

	// Use option.WithAPIKey to authenticate with an API key
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		Fatalf("Failed to create client: %v", err)
	}

	// Ensure the client is closed when main function finishes
	defer client.Close()

	output := GeminiLowerWrapper(promptText, ctx, client, mock, logToJsonl, quietMode)

	return output
}

// geminiChat handles interactive chat with conversation history
func geminiChat(history []ChatMessage, userInput string) (string, error) {
	apiKey := GetGeminiAPIKeyOrBail()
	ctx := context.Background()

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return "", fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	model := client.GenerativeModel(DefaultModels.Gemini)

	// Start a chat session with history
	chat := model.StartChat()

	// Add history to chat
	for _, msg := range history {
		role := "user"
		if msg.Role == "assistant" {
			role = "model"
		}
		chat.History = append(chat.History, &genai.Content{
			Parts: []genai.Part{genai.Text(msg.Content)},
			Role:  role,
		})
	}

	// Send the new message
	resp, err := chat.SendMessage(ctx, genai.Text(userInput))
	if err != nil {
		return "", fmt.Errorf("failed to send message: %w", err)
	}

	response, _, _ := StringifyGeminiResponse(resp, DefaultModels.Gemini)
	return response, nil
}

func GeminiWrapper(promptText string, mock bool, logToJsonl bool, quietMode bool) string {
	s := GeminiMiddleWrapper(promptText, mock, logToJsonl, quietMode) + "\n"
	if quietMode {
		return s
	}
	return fmt.Sprintf("# Gemini\n%s\n", s)
}
