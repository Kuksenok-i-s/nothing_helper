package session

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nothing_helper/internal/bt"
	"nothing_helper/internal/spp"
	"nothing_helper/internal/trace"
)

type gatedPacketTransport struct {
	gate, started chan struct{}
	data          *bytes.Reader
	first         bool
}

func (r *gatedPacketTransport) Read(p []byte) (int, error) {
	if !r.first {
		r.first = true
		close(r.started)
		<-r.gate
	}
	return r.data.Read(p)
}
func (r *gatedPacketTransport) Write(p []byte) (int, error) { return len(p), nil }
func (r *gatedPacketTransport) Close() error                { return nil }

func TestReadLoopDiscardsPreviousTransportPacket(t *testing.T) {
	s := New(nil, false, false)
	r := &gatedPacketTransport{gate: make(chan struct{}), started: make(chan struct{}), data: bytes.NewReader(spp.BuildFrame(spp.ControlTXDefault, spp.CmdRspBattery, 1, []byte{1, 1, 42}))}
	old := bt.NewTestTransport(r, "AA:BB:CC:DD:EE:01", 15, "old")
	s.AttachTestLink(old, bt.Device{MAC: old.RemoteMAC()})
	done := make(chan struct{})
	go func() { s.readLoop(old); close(done) }()
	<-r.started
	next := bt.NewTestTransport(&gatedPacketTransport{}, "AA:BB:CC:DD:EE:02", 15, "new")
	s.swapConnectTransport(bt.Device{MAC: next.RemoteMAC()}, next, 15)
	close(r.gate)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("read loop did not return")
	}
	if len(s.Snapshot().Batteries) > 0 {
		t.Fatalf("old packet polluted new device snapshot: %+v", s.Snapshot())
	}
}

func TestIdentityLogOmitsRawPayload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trace.ndjson")
	l, e := trace.NewLogger(path, false)
	if e != nil {
		t.Fatal(e)
	}
	s := New(l, false, false)
	s.handleRaw(spp.BuildFrame(spp.ControlTXDefault, spp.CmdRspIdentity, 1, []byte{0xde, 0xad, 0xbe, 0xef}))
	l.Close()
	b, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(string(b), "de ad be ef") {
		t.Fatalf("raw identity remains in redacted log: %s", b)
	}
}

var _ io.Reader = (*gatedPacketTransport)(nil)
