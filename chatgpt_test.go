package main

import "testing"

func TestChatGPTWrapper(t *testing.T) {
	RenderWithGlamour(ChatGPTWrapper("Mock prompt", true, false))
}
