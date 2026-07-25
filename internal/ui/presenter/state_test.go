package presenter

import (
	"errors"
	"strings"
	"testing"

	"tws_manager/internal/bt"
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

func TestApplyEventConnectedAndDisconnected(t *testing.T) {
	s := NewState(false)
	s.ApplyEvent(session.Event{
		Kind:   session.EventConnected,
		Device: bt.Device{Name: "Ear"},
	}, session.Snapshot{})
	if s.Status != "connected to Ear" {
		t.Fatalf("status=%q", s.Status)
	}

	s.AutoReconnect = true
	s.ApplyEvent(session.Event{
		Kind:   session.EventDisconnected,
		Device: bt.Device{Name: "Ear"},
	}, session.Snapshot{})
	if s.Status == "" {
		t.Fatal("expected disconnect status")
	}
}

func TestApplyEventPacketRXAndBattery(t *testing.T) {
	s := NewState(false)
	s.ApplyEvent(session.Event{
		Kind:    session.EventPacketRX,
		Packet:  spp.Packet{Cmd: spp.CmdRspBattery},
		Trigger: "battery",
	}, session.Snapshot{})
	if s.Status == "" {
		t.Fatal("expected rx status")
	}

	s.ApplyEvent(session.Event{Kind: session.EventBattery}, session.Snapshot{})
	if s.Status != "battery updated" {
		t.Fatalf("status=%q", s.Status)
	}
}

func TestApplyEventModel(t *testing.T) {
	s := NewState(false)
	s.ApplyEvent(session.Event{
		Kind:    session.EventModel,
		Trigger: "model probe",
	}, session.Snapshot{Model: spp.ModelInfo{Codename: "ear"}})
	if s.Status != "model detected: ear" {
		t.Fatalf("status=%q", s.Status)
	}
}

func TestApplyEventLineVariants(t *testing.T) {
	s := NewState(true)
	s.ApplyEvent(session.Event{
		Kind:    session.EventProgress,
		Source:  "scan",
		Trigger: "scan c001",
		Parsed:  spp.ParsedPacket{Summary: "parsed"},
		Raw:     []byte{0x01, 0x02},
	}, session.Snapshot{})
	if len(s.LogLines) != 1 {
		t.Fatal("expected log line")
	}

	s2 := NewState(false)
	s2.ApplyEvent(session.Event{
		Kind:  session.EventError,
		Error: errors.New("fail"),
		Trace: trace.Event{Summary: "trace summary"},
	}, session.Snapshot{})
	if s2.LogLines[0] == "" {
		t.Fatal("expected trace summary line")
	}
}

func TestApplyEventTrimLogLines(t *testing.T) {
	s := NewState(false)
	for i := 0; i < maxLogLines+5; i++ {
		s.ApplyEvent(session.Event{Kind: session.EventProgress, Trigger: "tick"}, session.Snapshot{})
	}
	if len(s.LogLines) != maxLogLines {
		t.Fatalf("log lines=%d want %d", len(s.LogLines), maxLogLines)
	}
}

func TestApplyEventTrimLastEvents(t *testing.T) {
	s := NewState(false)
	for i := 0; i < maxLastEvents+5; i++ {
		s.ApplyEvent(session.Event{
			Kind:  session.EventPacketTX,
			Trace: trace.Event{Direction: "tx", Summary: "sent"},
		}, session.Snapshot{})
	}
	if len(s.LastEvents) != maxLastEvents {
		t.Fatalf("last events=%d want %d", len(s.LastEvents), maxLastEvents)
	}
}

func TestApplyEventConnectedWithoutName(t *testing.T) {
	s := NewState(false)
	s.ApplyEvent(session.Event{Kind: session.EventConnected, Device: bt.Device{}}, session.Snapshot{})
	if s.Status != "connected" {
		t.Fatalf("status=%q", s.Status)
	}
}

func TestApplyEventModelFallback(t *testing.T) {
	s := NewState(false)
	s.ApplyEvent(session.Event{Kind: session.EventModel, Trigger: "unknown model"}, session.Snapshot{})
	if s.Status != "unknown model" {
		t.Fatalf("status=%q", s.Status)
	}
}

func TestApplyEventDisconnectManual(t *testing.T) {
	s := NewState(false)
	s.ApplyEvent(session.Event{
		Kind:   session.EventDisconnected,
		Device: bt.Device{MAC: "AA:BB:CC:DD:EE:FF"},
	}, session.Snapshot{})
	if !strings.Contains(s.Status, "Auto-connect") {
		t.Fatalf("status=%q", s.Status)
	}
}

func TestApplyEventTraceTimeFill(t *testing.T) {
	s := NewState(false)
	s.ApplyEvent(session.Event{
		Kind:  session.EventPacketTX,
		Trace: trace.Event{Direction: "tx", Summary: "sent"},
	}, session.Snapshot{})
	if s.LastEvents[0].Time == "" {
		t.Fatal("expected trace time to be filled")
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

func TestLogText(t *testing.T) {
	s := NewState(false)
	s.LogLines = []string{"a", "b"}
	if got := s.LogText(); got != "a\nb" {
		t.Fatalf("LogText() = %q", got)
	}
}

func TestFormatEventLineVariants(t *testing.T) {
	line := formatEventLine(session.Event{
		Kind:    session.EventProgress,
		Source:  "scan",
		Trigger: "scan c001",
		Parsed:  spp.ParsedPacket{Summary: "parsed"},
		Raw:     []byte{0x01},
	}, true)
	if !strings.Contains(line, "parsed") || !strings.Contains(line, "raw=") {
		t.Fatalf("line=%q", line)
	}
}

func TestApplyConnectedStatus(t *testing.T) {
	s := NewState(false)
	s.applyConnectedStatus(bt.Device{Name: "Ear"})
	if s.Status != "connected to Ear" {
		t.Fatalf("status=%q", s.Status)
	}
	s.applyConnectedStatus(bt.Device{})
	if s.Status != "connected" {
		t.Fatalf("status=%q", s.Status)
	}
}

func TestApplyPacketStatus(t *testing.T) {
	s := NewState(false)
	s.applyPacketStatus(session.Event{Kind: session.EventPacketTX, Packet: spp.Packet{Cmd: spp.CmdGetBattery}})
	if !strings.Contains(s.Status, "sent") {
		t.Fatalf("status=%q", s.Status)
	}
}

func TestApplyErrorStatusNil(t *testing.T) {
	s := NewState(false)
	s.applyErrorStatus(session.Event{Kind: session.EventError})
	if s.Err != "" || s.Status != "" {
		t.Fatalf("status=%q err=%q", s.Status, s.Err)
	}
}
