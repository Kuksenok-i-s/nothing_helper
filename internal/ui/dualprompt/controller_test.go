package dualprompt

import (
	"testing"

	"tws_manager/internal/dualpolicy"
	"tws_manager/internal/session"
	"tws_manager/internal/spp"
)

func TestOnSnapshot_ShowsPromptAfterInteraction(t *testing.T) {
	c := &Controller{Mode: dualpolicy.ModeAsk, HostMAC: "AA:BB:CC:DD:EE:FF"}
	snap := session.Snapshot{
		Connected: true,
		Model:     spp.ModelInfo{Codename: "EarTwos", Features: []string{"dual"}},
		DualList: []spp.DualDevice{
			{MAC: "11:22:33:44:55:66", Connected: true, Owner: true},
		},
	}
	if status := c.OnSnapshot(snap); status != "" {
		t.Fatalf("status = %q, want empty before interaction", status)
	}
	if c.Visible {
		t.Fatal("prompt should stay hidden until interaction")
	}
	c.OnInteraction()
	c.OnSnapshot(snap)
	if !c.Visible {
		t.Fatal("prompt should become visible after interaction")
	}
	if c.PromptLine() == "" {
		t.Fatal("expected prompt line")
	}
}

func TestAcceptFields_ReturnsHostMAC(t *testing.T) {
	c := &Controller{HostMAC: "AA:BB:CC:DD:EE:FF"}
	fields, err := c.AcceptFields()
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 3 || fields[2] != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf("fields = %v", fields)
	}
}
