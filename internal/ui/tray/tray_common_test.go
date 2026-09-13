package tray

import (
	"strings"
	"testing"

	"nothing_helper/internal/session"
	"nothing_helper/internal/spp"
)

func TestFormatBatteries(t *testing.T) {
	got := formatBatteries(map[string]spp.Battery{
		"left":  {Percent: 90},
		"right": {Percent: 80, Charging: true},
		"case":  {Percent: 50},
	})
	want := "L90% R80%+ C50%"
	if got != want {
		t.Fatalf("formatBatteries = %q, want %q", got, want)
	}
}

func TestStatusTitle(t *testing.T) {
	snap := session.Snapshot{
		Connected: true,
		Model:     spp.ModelInfo{Product: "Ear (3)"},
	}
	if got := statusTitle(snap); got != "Connected: Ear (3)" {
		t.Fatalf("statusTitle = %q", got)
	}
}

func TestTooltipForSnapshot(t *testing.T) {
	snap := session.Snapshot{
		Connected: true,
		Batteries: map[string]spp.Battery{"left": {Percent: 80}, "right": {Percent: 70}},
		Model:     spp.ModelInfo{Product: "Ear"},
	}
	got := tooltipForSnapshot(snap)
	if got == "" || !strings.Contains(got, "80") {
		t.Fatalf("tooltip=%q", got)
	}
}
