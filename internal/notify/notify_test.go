package notify

import (
	"testing"

	"tws_manager/internal/spp"
)

func TestGdbusIDRegex(t *testing.T) {
	out := []byte("(uint32 42,)\n")
	m := gdbusID.FindSubmatch(out)
	if m == nil || string(m[1]) != "42" {
		t.Fatalf("failed to parse gdbus id from %q: %v", out, m)
	}
}

func TestFormatBatteries(t *testing.T) {
	got := formatBatteries(map[string]spp.Battery{
		"left":  {Percent: 90},
		"right": {Percent: 80, Charging: true},
		"case":  {Percent: 50},
	})
	want := "Left 90%   Right 80% ⚡   Case 50%"
	if got != want {
		t.Fatalf("formatBatteries = %q, want %q", got, want)
	}
	if formatBatteries(nil) != "" {
		t.Fatal("empty battery map should format to empty string")
	}
}

func TestCheckLowBattery(t *testing.T) {
	earbud := earbudLowLevels
	caseLv := caseLowLevels
	fired := map[string]int{}
	var lastUrgency Urgency
	calls := 0
	rec := &Notifier{send: func(_ uint32, u Urgency, _, _, _ string) uint32 { calls++; lastUrgency = u; return 0 }}

	// 25% -> no alert
	checkLowBattery(rec, map[string]spp.Battery{"left": {Percent: 25}}, earbud, caseLv, fired)
	if calls != 0 {
		t.Fatalf("expected no alert at 25%%, got %d", calls)
	}
	// 18% -> crosses 20 (normal), one alert
	checkLowBattery(rec, map[string]spp.Battery{"left": {Percent: 18}}, earbud, caseLv, fired)
	if calls != 1 || lastUrgency != UrgencyNormal {
		t.Fatalf("expected 1 normal alert at 18%%, got %d urgency=%d", calls, lastUrgency)
	}
	// still 15% -> same threshold, no repeat
	checkLowBattery(rec, map[string]spp.Battery{"left": {Percent: 15}}, earbud, caseLv, fired)
	if calls != 1 {
		t.Fatalf("expected no repeat alert at 15%%, got %d", calls)
	}
	// 8% -> crosses 10 (critical), new alert
	checkLowBattery(rec, map[string]spp.Battery{"left": {Percent: 8}}, earbud, caseLv, fired)
	if calls != 2 || lastUrgency != UrgencyCritical {
		t.Fatalf("expected 2nd critical alert at 8%%, got %d urgency=%d", calls, lastUrgency)
	}
	// 4% -> crosses 5 (critical), new alert
	checkLowBattery(rec, map[string]spp.Battery{"left": {Percent: 4}}, earbud, caseLv, fired)
	if calls != 3 {
		t.Fatalf("expected 3rd alert at 4%%, got %d", calls)
	}
	// charging resets the component
	checkLowBattery(rec, map[string]spp.Battery{"left": {Percent: 4, Charging: true}}, earbud, caseLv, fired)
	if _, ok := fired["left"]; ok {
		t.Fatal("charging should clear fired state")
	}

	// case battery now alerts on its own thresholds (20 then 10)
	caseFired := map[string]int{}
	calls = 0
	checkLowBattery(rec, map[string]spp.Battery{"case": {Percent: 18}}, earbud, caseLv, caseFired)
	if calls != 1 || lastUrgency != UrgencyNormal {
		t.Fatalf("expected case normal alert at 18%%, got %d urgency=%d", calls, lastUrgency)
	}
	checkLowBattery(rec, map[string]spp.Battery{"case": {Percent: 8}}, earbud, caseLv, caseFired)
	if calls != 2 || lastUrgency != UrgencyCritical {
		t.Fatalf("expected case critical alert at 8%%, got %d urgency=%d", calls, lastUrgency)
	}
	// case has no 5% level, so 3% does not add a new alert
	checkLowBattery(rec, map[string]spp.Battery{"case": {Percent: 3}}, earbud, caseLv, caseFired)
	if calls != 2 {
		t.Fatalf("expected no extra case alert at 3%%, got %d", calls)
	}
}
