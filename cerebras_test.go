package main

import "testing"

func TestCerebrasWrapper(t *testing.T) {
	RenderWithGlamour(CerebrasWrapper("Mock prompt", true))
}
