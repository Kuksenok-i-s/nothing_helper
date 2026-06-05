package trace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tws_manager/internal/spp"
)

func TestLoggerRedactsRawByDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trace.ndjson")
	tr, err := NewLogger(path, false)
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Close()

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
