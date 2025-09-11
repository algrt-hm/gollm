package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestFilletHelp(t *testing.T) {
	// Here we essentially test that the README.md example of gollm -h updated and correct
	ret := filletHelp(getReadme())

	// check that ret is a single slice
	if len(ret) != 1 {
		t.Errorf("Expected 1 slice, got %d", len(ret))
	}

	// check that the slice is non-empty
	if len(ret[0]) == 0 {
		t.Errorf("Expected non-empty slice, got empty slice")
	}

	helpFromReadme := ret[0]
	helpFromCode := PrintUsage(true, getLogPath())

	// remove leading and trailing whitespace
	helpFromReadme = strings.TrimSpace(helpFromReadme)
	helpFromCode = strings.TrimSpace(helpFromCode)

	// run through both strings and print the first rune where they differ, with 10 runes of context either side
	runesReadme := []rune(helpFromReadme)
	runesCode := []rune(helpFromCode)

	minLen := len(runesReadme)
	if len(runesCode) < minLen {
		minLen = len(runesCode)
	}

	for i := 0; i < minLen; i++ {
		if runesReadme[i] != runesCode[i] {
			start := i - 10
			if start < 0 {
				start = 0
			}
			end := i + 10
			if end > minLen {
				end = minLen
			}
			fmt.Printf("First difference at rune position %d\n", i)
			// show runes as they are and with integer representation
			fmt.Printf("rune at issue readme: %c code: %c\n", runesReadme[i], runesCode[i])
			fmt.Printf("rune at issue readme: %d code: %d\n", runesReadme[i], runesCode[i])
			fmt.Println("Context:")
			fmt.Printf("Readme:\n%s\n", string(runesReadme[start:end]))
			fmt.Printf("Code:\n%s\n", string(runesCode[start:end]))
			break
		}
	}

	if helpFromReadme != helpFromCode {
		t.Errorf("Expected help from readme to match help from code, help from readme is:\n%s\nhelp from code is:\n%s", helpFromReadme, helpFromCode)
	}
}