package main

import "testing"

func TestChatGPTWrapper(t *testing.T) {
	quietMode = true
	Render(ChatGPTWrapper("Mock prompt", true, false, quietMode))
}
