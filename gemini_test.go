package main

import "testing"

func TestGeminiWrapper(t *testing.T) {
	quietMode = true
	Render(GeminiWrapper("Mock prompt", true, false, quietMode))
}
