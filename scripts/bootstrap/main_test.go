package main

import (
	"testing"
)

func setup() {
}

// tests Grep function
func TestGrep(t *testing.T) {
	setup()

	have, _ := grep("main_test.go", "wooty tooty")
	want := "	have, _ := grep(\"main_test.go\", \"wooty tooty\")"

	if have != want {
		t.Fatalf("Unexpected output. \nHave: %s\nWant: %s", have, want)
	}
}
