package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LogEntry represents a single log entry for model calls
type LogEntry struct {
	ModelName     string    `json:"model_name"`
	TotalTokens   int       `json:"total_tokens"`
	Duration      float64   `json:"duration_seconds"`
	StopReason    string    `json:"stop_reason"`
	PromptText    string    `json:"prompt_text"`
	ModelResponse string    `json:"model_response"`
	Timestamp     time.Time `json:"timestamp"`
}

// WriteLogEntry writes a single log entry to the JSONL file
func WriteLogEntry(entry LogEntry) error {
	// Get user's home directory
	// Note: %w is the special formatting for errors
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	// Create log file path in home directory
	logFilePath := filepath.Join(homeDir, "gollm_logs.jsonl")

	// Convert entry to JSON
	jsonData, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	// Open file in append mode, create if doesn't exist
	file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	// Write JSON line with newline
	if _, err := file.Write(append(jsonData, '\n')); err != nil {
		return fmt.Errorf("failed to write to log file: %w", err)
	}

	return nil
}
