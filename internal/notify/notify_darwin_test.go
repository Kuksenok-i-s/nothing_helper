//go:build darwin

package notify

import "testing"

func TestNewUsesOsascriptBackend(t *testing.T) {
	n := New("tws_manager", "")
	if n.backend != "osascript" {
		t.Fatalf("backend=%q want osascript", n.backend)
	}
	if !n.Available() {
		t.Fatal("expected osascript backend to be available")
	}
}
