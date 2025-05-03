package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

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

	// --- List Models ---
	builder.WriteString("--- Available Models ---\n")
	iter := client.ListModels(ctx)

	// Loop through the models returned by the iterator
	for {
		info, err := iter.Next()

		if errors.Is(err, iterator.Done) {
			// The iterator is exhausted, break the loop
			break
		}

		if err != nil {
			// Handle any other error during iteration
			Fatalf("Failed to iterate models: %v", err)
		}

		// Note that we're only interested in models with generateContent in SupportedGenerationMethods
		if strSliceContains(info.SupportedGenerationMethods, "generateContent") {
			// Print information about the model
			fmt.Fprintf(&builder, "%s Display name: %s Supports: %v\n", info.Name, info.DisplayName, info.SupportedGenerationMethods)
			if info.Description != "" {
				fmt.Fprintf(&builder, "Description: %s\n", info.Description)
			} else {
				builder.WriteString("Description: (none)\n")
			}
			builder.WriteString("----------------------\n")
		}
	}
	builder.WriteString("--- End of List ---\n")

	return builder.String()
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
	// --- 3. Select the model ---
	model := client.GenerativeModel(modelName)

	resp, err := model.GenerateContent(ctx, genai.Text(promptText))

	if err != nil {
		Fatalf("Failed to generate content: %v", err)
	}

	return resp, err
}

// GeminiLowerWrapper calls the Gemini API
func GeminiLowerWrapper(promptText string, ctx context.Context, client *genai.Client, mock bool) string {
	// Start the timer
	startTime := time.Now()
	modelName := "models/gemini-2.0-pro-exp-02-05"

	resp, err := GeminiCallAPI(modelName, promptText, ctx, client, mock)

	if err != nil {
		Fatalf("Some issue: %s", err)
	}

	buffer, finishReason, safetyRating := StringifyGeminiResponse(resp, modelName)
	totalTokenCount := resp.UsageMetadata.TotalTokenCount
	duration := time.Since(startTime)

	if safetyRating != "" {
		return fmt.Sprintf("\nModel: %s, %d tokens used, finished due to: %s, safety rating: %s, duration: %.3f seconds\n\n%s\n", modelName, totalTokenCount, finishReason, safetyRating, duration.Seconds(), buffer)
	} else {
		return fmt.Sprintf("\nModel: %s, %d tokens used, finished due to: %s, duration: %.3f seconds\n\n%s\n", modelName, totalTokenCount, finishReason, duration.Seconds(), buffer)
	}
}

func GeminiMiddleWrapper(promptText string, mock bool) string {
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

	output := GeminiLowerWrapper(promptText, ctx, client, mock)

	return output
}

func GeminiWrapper(promptText string, mock bool) string {
	return fmt.Sprintf("# Gemini\n%s\n\n", GeminiMiddleWrapper(promptText, mock))
}
