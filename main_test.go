package main

import (
	"fmt"
	"testing"
)

func TestPrintAPIKeys(t *testing.T) {
	PrintAPIKeys()
}

func TestCheckInternetHTTP(t *testing.T) {
	ret, err := CheckInternetHTTP()
	if ret {
		fmt.Println("Connected to internet")
	} else {
		fmt.Printf("Connected to internet %v: %v", ret, err)
	}
}

func TestPrintUsage(t *testing.T) {
	PrintUsage(true)
}
