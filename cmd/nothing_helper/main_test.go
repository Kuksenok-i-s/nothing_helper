package main

import "testing"

func TestHelp(t *testing.T) {
	if got := runMain([]string{"--help"}); got != 0 {
		t.Fatalf("help exit = %d", got)
	}
}
func TestInvalidFlag(t *testing.T) {
	if got := runMain([]string{"--not-a-flag"}); got != 2 {
		t.Fatalf("invalid flag exit = %d", got)
	}
}
