package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

func getLogPath() (string, error) {
	var err error = nil
	const logFn string = "gollm_logs.jsonl"

	// Get user's home directory
	// Note: %w is the special formatting for errors
	homeDir, err := os.UserHomeDir()

	if err != nil {
		err = fmt.Errorf("failed to get user home directory: %w", err)
	}

	// Create log file path in home directory
	logFilePath := filepath.Join(homeDir, logFn)

	return logFilePath, err
}

// printLogEntry
// is a helper function to print the log entry
func printLogEntry(i int, r LogEntry, incResponse bool) {
	if incResponse {
		Render(r.ModelResponse)
		return
	}

	var prompt string

	niceTimestamp := r.Timestamp.Format("2006-01-02 15:04:05")
	if len(r.PromptText) > 120 {
		prompt = r.PromptText[:120] + " ..."
	} else {
		prompt = r.PromptText
	}

	// let's indent the prompt by replacing each \n with \t\n
	prompt = strings.ReplaceAll(prompt, "\n", "\n\t")
	fmt.Printf("%d :: %s :: %s\n\t> %s\n\n", i, niceTimestamp, r.ModelName, prompt)
}

// ReadLogIdx
// pass in a negative logIdx to print all
func ReadLogIdx(logIdx int) error {
	var logEntries []LogEntry

	logFilePath, err := getLogPath()
	if err != nil {
		return err
	}

	// Open file
	file, err := os.OpenFile(logFilePath, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNo := 0

	for scanner.Scan() {
		lineNo++
		lineBytes := scanner.Bytes()

		// Nothing to do if empty line
		if len(lineBytes) == 0 {
			continue
		}

		var logEntry LogEntry
		err := json.Unmarshal(lineBytes, &logEntry)
		if err != nil {
			// If an error print and skip
			fmt.Printf("Error unmarshalling line %d (%s): %v", lineNo, string(lineBytes), err)
			continue
		}
		logEntries = append(logEntries, logEntry)
	}

	if err := scanner.Err(); err != nil {
		Fatalf("Error reading file %s: %v", logFilePath, err)
	}

	sort.Slice(logEntries, func(i int, j int) bool {
		return logEntries[i].Timestamp.After(logEntries[j].Timestamp)
	})

	if logIdx < 0 {
		for i, r := range logEntries {
			printLogEntry(i, r, false)
		}
	} else {
		nLogEntries := len(logEntries)
		if logIdx+1 > nLogEntries {
			Fatalf("Idx %d doesn't make sense when we have %d log entries", logIdx, nLogEntries)
		}
		printLogEntry(logIdx, logEntries[logIdx], true)
	}

	return nil
}

// WriteLogEntry writes a single log entry to the JSONL file
func WriteLogEntry(entry LogEntry) error {
	// Convert entry to JSON
	jsonData, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	logFilePath, err := getLogPath()
	if err != nil {
		return err
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
