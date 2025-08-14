package main

import "testing"

func TestCerebrasWrapper(t *testing.T) {
	quietMode = true
	Render(CerebrasWrapper("Mock prompt", true, false, quietMode))
}
