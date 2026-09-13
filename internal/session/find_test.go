package session

import (
	"context"
	"io"
	"sync"
	"testing"
	"time"

	"nothing_helper/internal/bt"
	"nothing_helper/internal/spp"
)

type findWire struct {
	mu      sync.Mutex
	packets []spp.Packet
	onWrite func(spp.Packet)
	started chan struct{}
	once    sync.Once
}

func (w *findWire) Read([]byte) (int, error) { return 0, io.EOF }
func (w *findWire) Close() error             { return nil }
func (w *findWire) Write(raw []byte) (int, error) {
	pkt, err := spp.DecodePacket(raw)
	if err != nil {
		return 0, err
	}
	w.mu.Lock()
	w.packets = append(w.packets, pkt)
	w.mu.Unlock()
	if w.onWrite != nil {
		w.onWrite(pkt)
	}
	if pkt.Cmd == spp.CmdFindEarbud && pkt.Payload[1] == 1 && w.started != nil {
		w.once.Do(func() { close(w.started) })
	}
	return len(raw), nil
}
func findSession(t *testing.T) (*Session, *findWire) {
	t.Helper()
	s := New(nil, false, false)
	w := &findWire{started: make(chan struct{})}
	s.AttachTestLink(bt.NewTestTransport(w, "AA:BB:CC:DD:EE:FF", 15, "test"), bt.Device{MAC: "AA:BB:CC:DD:EE:FF"})
	s.model, _ = spp.ResolveModelInfo("EarThree")
	return s, w
}
func TestFindGuardCannotBeBypassedWithUnsafe(t *testing.T) {
	cases := []struct {
		name    string
		status  spp.EarbudStatus
		age     time.Duration
		missing bool
	}{
		{name: "unknown", missing: true},
		{name: "in ear", status: spp.EarbudStatus{InEar: true, Connected: true}},
		{name: "in case", status: spp.EarbudStatus{InCase: true, Connected: true}},
		{name: "disconnected", status: spp.EarbudStatus{}},
		{name: "stale", status: spp.EarbudStatus{Connected: true}, age: 3 * time.Second},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, w := findSession(t)
			s.allowUnsafe = true
			if !tc.missing {
				s.earbuds = map[string]EarbudState{"left": {tc.status, time.Now().Add(-tc.age)}}
			}
			if err := s.Send(spp.Packet{Cmd: spp.CmdFindEarbud, Payload: []byte{2, 1}}, Meta{}); err == nil {
				t.Fatal("unsafe search allowed")
			}
			if len(w.packets) != 0 {
				t.Fatal("blocked command was written")
			}
			if err := s.Send(spp.Packet{Cmd: spp.CmdFindEarbud, Payload: []byte{2, 0}}, Meta{}); err != nil {
				t.Fatalf("stop blocked: %v", err)
			}
		})
	}
}
func TestFindFreshCheckAndAutomaticStop(t *testing.T) {
	for _, name := range []string{"cancel", "insertion", "stale"} {
		insert := name == "insertion"
		t.Run(name, func(t *testing.T) {
			s, w := findSession(t)
			// The response comes only after the status query is written.
			w.onWrite = func(pkt spp.Packet) {
				if pkt.Cmd == spp.CmdGetStatus {
					s.recordEarbuds(spp.ParsedPacket{Earbuds: map[string]spp.EarbudStatus{"left": {Connected: true}}})
				}
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- s.FindEarbud(ctx, "left") }()
			select {
			case <-w.started:
			case <-time.After(time.Second):
				t.Fatal("search did not start")
			}
			if insert {
				s.recordEarbuds(spp.ParsedPacket{Earbuds: map[string]spp.EarbudStatus{"left": {Connected: true, InEar: true}}})
			} else if name == "stale" {
				s.mu.Lock()
				state := s.earbuds["left"]
				state.UpdatedAt = time.Now().Add(-3 * time.Second)
				s.earbuds["left"] = state
				s.mu.Unlock()
			} else {
				cancel()
			}
			select {
			case err := <-done:
				if insert && err == nil {
					t.Fatal("missing insertion reason")
				}
			case <-time.After(time.Second):
				t.Fatal("search failed to stop")
			}
			w.mu.Lock()
			defer w.mu.Unlock()
			first, last := w.packets[0], w.packets[len(w.packets)-1]
			if first.Cmd != spp.CmdGetStatus {
				t.Fatal("sensor not queried before search")
			}
			if last.Cmd != spp.CmdFindEarbud || last.Payload[0] != 2 || last.Payload[1] != 0 {
				t.Fatal("missing left-earbud stop")
			}
		})
	}
}
func TestFindDoesNotUseCachedStatusWithoutResponse(t *testing.T) {
	s, w := findSession(t)
	s.recordEarbuds(spp.ParsedPacket{Earbuds: map[string]spp.EarbudStatus{"right": {Connected: true}}})
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if err := s.FindEarbud(ctx, "right"); err == nil {
		t.Fatal("expected cancellation waiting for sensor")
	}
	for _, p := range w.packets {
		if p.Cmd == spp.CmdFindEarbud && p.Payload[1] == 1 {
			t.Fatal("cached status used to start sound")
		}
	}
}
func TestEarbudStateSnapshotAndDisconnect(t *testing.T) {
	s, _ := findSession(t)
	s.recordEarbuds(spp.ParsedPacket{Earbuds: map[string]spp.EarbudStatus{"left": {InEar: true, Connected: true}}})
	snap := s.Snapshot()
	delete(snap.Earbuds, "left")
	if !s.Snapshot().Earbuds["left"].InEar {
		t.Fatal("snapshot aliases state")
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if len(s.Snapshot().Earbuds) != 0 {
		t.Fatal("stale wear state survived disconnect")
	}
}

func TestFindStopsAtDurationLimit(t *testing.T) {
	s, w := findSession(t)
	w.onWrite = func(pkt spp.Packet) {
		if pkt.Cmd == spp.CmdGetStatus {
			s.recordEarbuds(spp.ParsedPacket{Earbuds: map[string]spp.EarbudStatus{"right": {Connected: true}}})
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	started := time.Now()
	if err := s.FindEarbud(ctx, "right"); err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(started)
	if elapsed < 10*time.Second || elapsed > 11*time.Second {
		t.Fatalf("unexpected signal duration: %s", elapsed)
	}
	last := w.packets[len(w.packets)-1]
	if last.Cmd != spp.CmdFindEarbud || last.Payload[0] != 3 || last.Payload[1] != 0 {
		t.Fatal("duration limit did not stop right earbud")
	}
}
