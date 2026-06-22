//go:build linux

package notify

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"tws_manager/internal/session"
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

func TestOrderedComponentsIncludesTwsAndExtras(t *testing.T) {
	got := orderedComponents(map[string]spp.Battery{
		"tws":   {Percent: 50},
		"id_7":  {Percent: 40},
		"left":  {Percent: 60},
	})
	want := []string{"left", "tws", "id_7"}
	if len(got) != len(want) {
		t.Fatalf("orderedComponents = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("orderedComponents[%d] = %q, want %q (full: %v)", i, got[i], want[i], got)
		}
	}
}

func TestCheckLowBattery(t *testing.T) {
	earbud := earbudLowLevels
	caseLv := caseLowLevels
	fired := map[string]int{}
	var lastUrgency Urgency
	calls := 0
	rec := &Notifier{send: func(_ uint32, u Urgency, _, _, _ string) uint32 { calls++; lastUrgency = u; return 0 }}

	checkLowBattery(rec, map[string]spp.Battery{"left": {Percent: 25}}, earbud, caseLv, fired)
	if calls != 0 {
		t.Fatalf("expected no alert at 25%%, got %d", calls)
	}
	checkLowBattery(rec, map[string]spp.Battery{"left": {Percent: 18}}, earbud, caseLv, fired)
	if calls != 1 || lastUrgency != UrgencyNormal {
		t.Fatalf("expected 1 normal alert at 18%%, got %d urgency=%d", calls, lastUrgency)
	}
	checkLowBattery(rec, map[string]spp.Battery{"left": {Percent: 15}}, earbud, caseLv, fired)
	if calls != 1 {
		t.Fatalf("expected no repeat alert at 15%%, got %d", calls)
	}
	checkLowBattery(rec, map[string]spp.Battery{"left": {Percent: 8}}, earbud, caseLv, fired)
	if calls != 2 || lastUrgency != UrgencyCritical {
		t.Fatalf("expected 2nd critical alert at 8%%, got %d urgency=%d", calls, lastUrgency)
	}
	checkLowBattery(rec, map[string]spp.Battery{"left": {Percent: 4}}, earbud, caseLv, fired)
	if calls != 3 {
		t.Fatalf("expected 3rd alert at 4%%, got %d", calls)
	}
	checkLowBattery(rec, map[string]spp.Battery{"left": {Percent: 4, Charging: true}}, earbud, caseLv, fired)
	if _, ok := fired["left"]; ok {
		t.Fatal("charging should clear fired state")
	}

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
	checkLowBattery(rec, map[string]spp.Battery{"case": {Percent: 3}}, earbud, caseLv, caseFired)
	if calls != 2 {
		t.Fatalf("expected no extra case alert at 3%%, got %d", calls)
	}

	calls = 0
	twsFired := map[string]int{}
	checkLowBattery(rec, map[string]spp.Battery{"tws": {Percent: 18}}, earbud, caseLv, twsFired)
	if calls != 1 {
		t.Fatalf("expected tws alert at 18%%, got %d", calls)
	}
}

func TestRunUnavailableWarns(t *testing.T) {
	origLookPath := execLookPath
	origWarnf := Warnf
	defer func() {
		execLookPath = origLookPath
		Warnf = origWarnf
	}()
	execLookPath = func(string) (string, error) { return "", fmt.Errorf("not found") }
	var warns []string
	Warnf = func(format string, args ...any) {
		warns = append(warns, fmt.Sprintf(format, args...))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	Run(ctx, session.New(nil, false, false), Options{AppName: "test"})

	if len(warns) != 1 || !strings.Contains(warns[0], "gdbus") {
		t.Fatalf("warns = %v, want unavailable PATH warning", warns)
	}
}

type fakeSessionView struct {
	snap session.Snapshot
}

func (f fakeSessionView) Snapshot() session.Snapshot { return f.snap }

func TestProcessEventConnectAndLowBattery(t *testing.T) {
	sv := fakeSessionView{snap: session.Snapshot{
		Batteries: map[string]spp.Battery{"left": {Percent: 18}},
	}}

	var alerts []string
	n := &Notifier{
		backend: "test",
		send: func(_ uint32, _ Urgency, title, body, _ string) uint32 {
			alerts = append(alerts, title+": "+body)
			return 0
		},
	}

	opts := Options{AppName: "test"}
	lowFired := map[string]int{}
	processEvent(session.Event{Kind: session.EventConnected}, sv, n, opts, earbudLowLevels, caseLowLevels, lowFired)

	if len(alerts) != 2 {
		t.Fatalf("alerts = %v, want connect + low battery", alerts)
	}
	if !strings.Contains(alerts[0], "Connected") {
		t.Fatalf("first alert = %q, want connect", alerts[0])
	}
	if !strings.Contains(alerts[1], "Battery low") {
		t.Fatalf("second alert = %q, want low battery", alerts[1])
	}
}

func TestProcessEventBatteryNoAlertWhenHealthy(t *testing.T) {
	sv := fakeSessionView{snap: session.Snapshot{
		Batteries: map[string]spp.Battery{
			"left":  {Percent: 90},
			"right": {Percent: 80, Charging: true},
		},
	}}

	calls := 0
	n := &Notifier{
		backend: "test",
		send: func(_ uint32, _ Urgency, _, _, _ string) uint32 {
			calls++
			return 0
		},
	}

	lowFired := map[string]int{}
	processEvent(session.Event{Kind: session.EventBattery}, sv, n, Options{AppName: "test"}, earbudLowLevels, caseLowLevels, lowFired)

	if calls != 0 {
		t.Fatalf("expected no notifications at healthy battery, got %d", calls)
	}
}

func TestCheckLowBatteryHysteresis(t *testing.T) {
	earbud := earbudLowLevels
	caseLv := caseLowLevels
	fired := map[string]int{}
	calls := 0
	rec := &Notifier{send: func(_ uint32, _ Urgency, _, _, _ string) uint32 { calls++; return 0 }}

	data := map[string]spp.Battery{"left": {Percent: 18}}
	checkLowBattery(rec, data, earbud, caseLv, fired)
	if calls != 1 {
		t.Fatalf("expected alert at 18%%, got %d", calls)
	}
	data["left"] = spp.Battery{Percent: 21}
	checkLowBattery(rec, data, earbud, caseLv, fired)
	if calls != 1 {
		t.Fatalf("expected no repeat alert at 21%%, got %d", calls)
	}
	data["left"] = spp.Battery{Percent: 18}
	checkLowBattery(rec, data, earbud, caseLv, fired)
	if calls != 1 {
		t.Fatalf("expected no repeat alert after oscillation, got %d", calls)
	}
	data["left"] = spp.Battery{Percent: 26}
	checkLowBattery(rec, data, earbud, caseLv, fired)
	if _, ok := fired["left"]; ok {
		t.Fatal("expected fired cleared after recovery above threshold")
	}
}

func TestProcessEventDisconnectClearsLowFired(t *testing.T) {
	sv := fakeSessionView{}
	n := &Notifier{
		backend: "test",
		send:    func(uint32, Urgency, string, string, string) uint32 { return 0 },
	}
	lowFired := map[string]int{"left": 20}
	opts := Options{AppName: "test"}

	processEvent(session.Event{Kind: session.EventDisconnected}, sv, n, opts, earbudLowLevels, caseLowLevels, lowFired)

	if len(lowFired) != 0 {
		t.Fatalf("lowFired = %v, want cleared after disconnect", lowFired)
	}
}
