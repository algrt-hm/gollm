package main

import "testing"

func TestCerebrasWrapper(t *testing.T) {
	// To turn off output if we don't want it
	if testingSuppressOutput {
		t.Cleanup(func() { RenderWithGlamourIndirect = RenderWithGlamour })
		RenderWithGlamourIndirect = func(s string) {}
	}

	quietMode = false
	Render(CerebrasWrapper("Mock prompt", true, false, quietMode))
}
