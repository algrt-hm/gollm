package main

import "testing"

func TestGeminiWrapper(t *testing.T) {
	RenderWithGlamour(GeminiWrapper("Mock prompt", true, false))
}
