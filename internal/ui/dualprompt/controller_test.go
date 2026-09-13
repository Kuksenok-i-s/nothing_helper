package dualprompt

import (
	"errors"
	"fmt"
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

func TestEnsureHostMACLoadsAdapter(t *testing.T) {
	dualpolicy.SetHostAdapterMACHook(func() (string, error) {
		return "AA:BB:CC:DD:EE:FF", nil
	})
	t.Cleanup(func() { dualpolicy.SetHostAdapterMACHook(nil) })

	c := &Controller{}
	c.EnsureHostMAC()
	if c.HostMAC != "AA:BB:CC:DD:EE:FF" || c.HostErr != "" {
		t.Fatalf("HostMAC=%q HostErr=%q", c.HostMAC, c.HostErr)
	}
	c.EnsureHostMAC()
}

func TestEnsureHostMACRecordsError(t *testing.T) {
	dualpolicy.SetHostAdapterMACHook(func() (string, error) {
		return "", errors.New("adapter unavailable")
	})
	t.Cleanup(func() { dualpolicy.SetHostAdapterMACHook(nil) })

	c := &Controller{}
	c.EnsureHostMAC()
	if c.HostErr == "" {
		t.Fatal("expected host error")
	}
}

// Status polling and further UI interaction must not reopen an answered prompt.
func TestAnsweredPromptStaysDismissedUntilReconnect(t *testing.T) {
	for _, accept := range []bool{false, true} {
		t.Run(fmt.Sprint(accept), func(t *testing.T) {
			c := &Controller{Mode: dualpolicy.ModeAsk, HostMAC: "AA:BB:CC:DD:EE:FF"}
			snap := session.Snapshot{Connected: true, Model: spp.ModelInfo{Codename: "EarTwos", Features: []string{"dual"}}, DualList: []spp.DualDevice{{MAC: "11:22:33:44:55:66", Connected: true, Owner: true}}}
			c.OnSnapshot(snap)
			c.OnInteraction()
			if !c.Visible {
				t.Fatal("expected initial prompt")
			}
			if accept {
				if _, err := c.AcceptFields(); err != nil {
					t.Fatal(err)
				}
			} else {
				c.Decline()
			}
			for range 3 {
				c.OnSnapshot(snap)
				c.OnInteraction()
				if c.PromptLine() != "" {
					t.Fatal("answered prompt reopened")
				}
			}
			// Transient empty owner lists must not undo the user's answer either.
			c.OnSnapshot(session.Snapshot{Connected: true})
			c.OnSnapshot(snap)
			if c.Visible {
				t.Fatal("prompt reopened after empty owner list")
			}
			c.OnDisconnected()
			c.OnSnapshot(snap)
			c.OnInteraction()
			if !c.Visible {
				t.Fatal("new connection should allow another prompt")
			}
		})
	}
}
