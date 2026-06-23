//go:build darwin

package notify

import "testing"

// The test binary is not an app bundle, so New must fall back to osascript and
// still report itself as available.
func TestNewUsesOsascriptBackend(t *testing.T) {
	n := New("tws_manager", "")
	if n.backend != "osascript" {
		t.Fatalf("backend=%q want osascript", n.backend)
	}
	if !n.Available() {
		t.Fatal("expected osascript backend to be available")
	}
}

// sendNative dispatches delivery onto the main queue and returns immediately; it
// must marshal C strings safely and never panic regardless of run-loop state.
func TestSendNativeDoesNotPanic(t *testing.T) {
	n := &Notifier{app: "Nothing Ear", icon: "audio-headphones"}
	if got := n.sendNative(0, UrgencyCritical, "Battery low", "Left earbud at 5%", ""); got != 0 {
		t.Fatalf("sendNative returned %d, want 0", got)
	}
	// Empty fields must not crash the UTF-8 conversion path.
	_ = n.sendNative(0, UrgencyNormal, "", "", "")
}
