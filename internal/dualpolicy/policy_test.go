package dualpolicy

import (
	"strings"
	"testing"

	"tws_manager/internal/spp"
)

func TestParseMode(t *testing.T) {
	got, err := ParseMode("ask")
	if err != nil || got != ModeAsk {
		t.Fatalf("ask: got %q err=%v", got, err)
	}
	got, err = ParseMode("")
	if err != nil || got != ModeAsk {
		t.Fatalf("empty: got %q err=%v", got, err)
	}
	if _, err := ParseMode("bad"); err == nil {
		t.Fatal("expected error for bad mode")
	}
}

func TestHostOwnsDual(t *testing.T) {
	host := "AA:BB:CC:DD:EE:FF"
	devs := []spp.DualDevice{
		{MAC: host, Name: "PC", Connected: true, Owner: true},
		{MAC: "11:22:33:44:55:66", Name: "Phone", Connected: true, Owner: false},
	}
	if !HostOwnsDual(devs, host) {
		t.Fatal("expected host to own dual")
	}
	if _, ok := ShouldPrompt(ModeAsk, spp.ModelInfo{Codename: "EarThree", Features: []string{"dual"}}, devs, host); ok {
		t.Fatal("should not prompt when host already owns dual even if phone is connected")
	}
}

func TestPhoneOwner(t *testing.T) {
	host := "AA:BB:CC:DD:EE:FF"
	devs := []spp.DualDevice{
		{MAC: host, Name: "PC", Connected: true, Owner: true},
	}
	if _, ok := PhoneOwner(devs, host); ok {
		t.Fatal("expected no phone when only PC is owner")
	}
	devs = []spp.DualDevice{
		{MAC: host, Name: "PC", Connected: false, Owner: false},
		{MAC: "11:22:33:44:55:66", Name: "Phone", Connected: true, Owner: true},
	}
	phone, ok := PhoneOwner(devs, host)
	if !ok || phone.MAC != "11:22:33:44:55:66" {
		t.Fatalf("phone=%+v ok=%v", phone, ok)
	}
}

func TestShouldPrompt(t *testing.T) {
	model := spp.ModelInfo{Codename: "EarThree", Features: []string{"dual"}}
	phone, ok := ShouldPrompt(ModeAsk, model, []spp.DualDevice{
		{MAC: "11:22:33:44:55:66", Connected: true, Owner: true},
	}, "AA:BB:CC:DD:EE:FF")
	if !ok || phone.MAC != "11:22:33:44:55:66" {
		t.Fatalf("phone=%+v ok=%v", phone, ok)
	}
	if _, ok := ShouldPrompt(ModeOff, model, []spp.DualDevice{
		{MAC: "11:22:33:44:55:66", Connected: true, Owner: true},
	}, "AA:BB:CC:DD:EE:FF"); ok {
		t.Fatal("off mode should not prompt")
	}
}

func TestPromptText(t *testing.T) {
	got := PromptText(spp.DualDevice{Name: "Pixel", MAC: "11:22:33:44:55:66"})
	if !strings.Contains(got, "Pixel") {
		t.Fatalf("text=%q", got)
	}
}
