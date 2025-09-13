package main

import "testing"

func TestChatGPTWrapper(t *testing.T) {
	// To turn off output if we don't want it
	if testingSuppressOutput {
		t.Cleanup(func() { RenderWithGlamourPtr = RenderWithGlamour })
		RenderWithGlamourPtr = func(s string) {}
	}
	quietMode = false
	Render(ChatGPTWrapper("Mock prompt", true, false, quietMode))
}
