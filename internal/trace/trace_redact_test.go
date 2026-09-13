package trace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nothing_helper/internal/spp"
)

func TestLoggerRedactsRawByDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trace.ndjson")
	tr, err := NewLogger(path, false)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tr.Close() }()

	pkt := spp.Packet{Cmd: spp.CmdGetBattery}
	tr.LogTX([]byte{0x55, 0x60, 0x01, 0x01, 0xc0, 0x07, 0x00, 0x00, 0x00, 0xe9, 0xbf}, pkt, Context{})
	_ = tr.Close()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if strings.Contains(content, "raw_hex") || strings.Contains(content, "raw_without_crc") {
		t.Fatalf("expected redacted log, got %s", content)
	}
	if !strings.Contains(content, "summary") {
		t.Fatalf("expected summary in log, got %s", content)
	}
}

func TestPayloadSummariesDoNotBypassRawRedaction(t *testing.T) {
	cases := []struct {
		name    string
		cmd     uint16
		payload []byte
	}{
		{"identity", spp.CmdRspIdentity, []byte{0xde, 0xad, 0xbe, 0xef}},
		{"firmware", spp.CmdRspFirmware, []byte("private-payload")},
		{"unknown text", 0x4999, []byte("private-payload")},
		{"unknown binary", 0x4999, []byte{0xde, 0xad, 0xbe, 0xef}},
		{"malformed battery", spp.CmdRspBattery, []byte{0xde, 0xad, 0xbe, 0xef}},
		{"malformed ANC", spp.CmdRspANC, []byte{0xde, 0xad, 0xbe, 0xef}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, logRaw := range []bool{false, true} {
				dir := t.TempDir()
				logPath := filepath.Join(dir, "trace.ndjson")
				logger, err := NewLogger(logPath, logRaw)
				if err != nil {
					t.Fatal(err)
				}
				raw := spp.BuildFrame(spp.ControlTXDefault, tc.cmd, 1, tc.payload)
				pkt, err := spp.DecodePacket(raw)
				if err != nil {
					t.Fatal(err)
				}
				event := logger.LogRX(raw, pkt, spp.ParsePacket(pkt, spp.DefaultModel()), nil, Context{})
				if err := logger.Close(); err != nil {
					t.Fatal(err)
				}
				exportPath := filepath.Join(dir, "export.json")
				if err := Export(exportPath, []Event{event}, "", logRaw); err != nil {
					t.Fatal(err)
				}
				for _, path := range []string{logPath, exportPath} {
					data, err := os.ReadFile(path)
					if err != nil {
						t.Fatal(err)
					}
					content := string(data)
					if strings.Contains(content, "de ad be ef") || strings.Contains(content, "private-payload") {
						t.Fatalf("payload leaked through summary: %s", content)
					}
					if strings.Contains(content, "raw_hex") != logRaw {
						t.Fatalf("raw_hex presence does not match logRaw=%t: %s", logRaw, content)
					}
				}
			}
		})
	}
}
