//go:build gio

package state

import (
	"testing"
	"time"

	"tws_manager/internal/session"
	"tws_manager/internal/spp"
)

type fakeWindow struct {
	invalidates int
}

func (w *fakeWindow) Invalidate() {
	w.invalidates++
}

func TestStatusPollDueRequiresActiveConnectedGUI(t *testing.T) {
	now := time.Unix(100, 0)

	if statusPollDue(now, time.Time{}, false, true) {
		t.Fatal("poll due while GUI inactive")
	}
	if statusPollDue(now, time.Time{}, true, false) {
		t.Fatal("poll due while disconnected")
	}
	if !statusPollDue(now, time.Time{}, true, true) {
		t.Fatal("first active connected poll should be due")
	}
}

func TestStatusPollDueRespectsCadence(t *testing.T) {
	last := time.Unix(100, 0)

	if statusPollDue(last.Add(guiStatusRequestEvery-time.Second), last, true, true) {
		t.Fatal("poll due before cadence elapsed")
	}
	if !statusPollDue(last.Add(guiStatusRequestEvery), last, true, true) {
		t.Fatal("poll should be due at cadence boundary")
	}
}

func TestInvalidateCoalescesUntilFrame(t *testing.T) {
	w := &fakeWindow{}
	s := &State{window: w}

	s.invalidate()
	s.invalidate()
	if w.invalidates != 1 {
		t.Fatalf("invalidates before frame = %d, want 1", w.invalidates)
	}

	s.BeginFrame()
	s.invalidate()
	if w.invalidates != 2 {
		t.Fatalf("invalidates after frame = %d, want 2", w.invalidates)
	}
}

func TestSetTabSkipsUnchangedInvalidate(t *testing.T) {
	w := &fakeWindow{}
	s := &State{window: w, activeTab: TabControl}

	s.SetTab(TabControl)
	if w.invalidates != 0 {
		t.Fatalf("unchanged tab invalidates = %d, want 0", w.invalidates)
	}

	s.SetTab(TabLog)
	if w.invalidates != 1 {
		t.Fatalf("changed tab invalidates = %d, want 1", w.invalidates)
	}
}

func TestEventUpdatesCommandsOnlyForCommandMetadata(t *testing.T) {
	if eventUpdatesCommands(session.Event{Kind: session.EventPacketRX}) {
		t.Fatal("plain packet event should not rebuild command metadata")
	}
	if !eventUpdatesCommands(session.Event{Kind: session.EventModel}) {
		t.Fatal("model event should rebuild command metadata")
	}
	if !eventUpdatesCommands(session.Event{
		Kind:   session.EventPacketRX,
		Parsed: spp.ParsedPacket{DualList: &spp.DualDeviceList{}},
	}) {
		t.Fatal("dual list packet should rebuild command metadata")
	}
}
