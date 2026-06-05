package presenter

import (
	"errors"
	"testing"

	"tws_manager/internal/session"
	"tws_manager/internal/spp"
	"tws_manager/internal/trace"
)

func TestApplyEventProgressAndError(t *testing.T) {
	s := NewState(false)
	snap := session.Snapshot{}
	s.ApplyEvent(session.Event{Kind: session.EventProgress, Trigger: "opening"}, snap)
	if s.Status != "opening" {
		t.Fatalf("status=%q", s.Status)
	}
	s.ApplyEvent(session.Event{Kind: session.EventError, Error: errors.New("boom")}, snap)
	if s.Err != "boom" {
		t.Fatalf("err=%q", s.Err)
	}
}

func TestApplyEventLogAndExportBuffer(t *testing.T) {
	s := NewState(false)
	snap := session.Snapshot{}
	s.ApplyEvent(session.Event{
		Kind:  session.EventPacketTX,
		Trace: trace.Event{Direction: "tx", Summary: "sent"},
	}, snap)
	if len(s.LogLines) != 1 {
		t.Fatalf("log lines=%d", len(s.LogLines))
	}
	if len(s.LastEvents) != 1 {
		t.Fatalf("last events=%d", len(s.LastEvents))
	}
}

func TestFormatBatteries(t *testing.T) {
	got := FormatBatteries(map[string]spp.Battery{
		"left": {Percent: 80, Charging: true},
	})
	if got == "n/a" || got == "" {
		t.Fatalf("format=%q", got)
	}
}
