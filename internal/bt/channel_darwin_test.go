//go:build darwin

package bt

import "testing"

func TestChannelCandidatesDarwin(t *testing.T) {
	got := channelCandidates(7)
	if len(got) != 1 || got[0] != 7 {
		t.Fatalf("candidates = %v, want [7]", got)
	}
	got = channelCandidates(DefaultRFCOMMChannel)
	if len(got) != 1 || got[0] != DefaultRFCOMMChannel {
		t.Fatalf("default-only = %v", got)
	}
}
