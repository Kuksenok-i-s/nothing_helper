package trace

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tws_manager/internal/spp"
)

func TestLoggerCollectsCRCSamples(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trace.ndjson")
	tr, err := NewLogger(path, true)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer tr.Close()
	pkt := spp.Packet{Cmd: spp.CmdGetBattery}
	tr.LogTX([]byte{0x55, 0x60, 0x01, 0x01, 0xc0, 0x07, 0x00, 0x00, 0x00, 0xe9, 0xbf}, pkt, Context{})
	var out strings.Builder
	tr.PrintCRCSamples(&out)
	if got := out.String(); !strings.Contains(got, "55600101c007000000 e9bf") {
		t.Fatalf("crc samples = %q, want TX sample", got)
	}
}

func TestLogRXIncludesFrameHeaderFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trace.ndjson")
	tr, err := NewLogger(path, true)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	raw := spp.BuildFrame(spp.ControlCRC|spp.ControlMultiFrame, spp.CmdRspProtocolVersion, 1, []byte("1.0.0\n"))
	pkt, err := spp.DecodePacket(raw)
	if err != nil {
		t.Fatalf("DecodePacket() error = %v", err)
	}
	parsed := spp.ParsePacket(pkt, spp.DefaultModel())
	tr.LogRX(raw, pkt, parsed, nil, Context{})
	if err := tr.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var event Event
	if err := json.Unmarshal(bytes.TrimSpace(data), &event); err != nil {
		t.Fatalf("Unmarshal() error = %v: %s", err, data)
	}
	if event.Control != "0060" || event.Length != 6 || event.FSN != "01" {
		t.Fatalf("header fields = control %q length %d fsn %q", event.Control, event.Length, event.FSN)
	}
	if !event.HasCRC || event.CRCValid == nil || !*event.CRCValid || !event.MultiFrame {
		t.Fatalf("crc/multiframe fields = has_crc=%v crc_valid=%v multi_frame=%v", event.HasCRC, event.CRCValid, event.MultiFrame)
	}
}

func TestLoggerLoadsExistingCRCSamples(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trace.ndjson")
	if err := os.WriteFile(path, []byte(`{"time":"now","direction":"rx","raw_without_crc":"5560010140060000312e302e300a","crc":"b896"}`+"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	tr, err := NewLogger(path, true)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer tr.Close()
	var out strings.Builder
	tr.PrintCRCSamples(&out)
	if got := out.String(); !strings.Contains(got, "5560010140060000312e302e300a b896") {
		t.Fatalf("crc samples = %q, want loaded sample", got)
	}
}
