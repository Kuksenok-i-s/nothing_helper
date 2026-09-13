package session

import (
	"context"
	"testing"
	"time"
	"nothing_helper/internal/spp"
)

func TestCloseWaitsForFindStopBeforeClosingTransport(t *testing.T) {
	s, w := findSession(t)
	stopEntered, releaseStop := make(chan struct{}), make(chan struct{})
	w.onWrite = func(p spp.Packet) {
		if p.Cmd == spp.CmdGetStatus {
			s.recordEarbuds(spp.ParsedPacket{Earbuds: map[string]spp.EarbudStatus{"left": {Connected: true}}})
		}
		if p.Cmd == spp.CmdFindEarbud && p.Payload[1] == 0 {
			close(stopEntered)
			<-releaseStop
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	findDone := make(chan error, 1)
	go func() { findDone <- s.FindEarbud(ctx, "left") }()
	select {
	case <-w.started:
	case <-time.After(time.Second):
		t.Fatal("find not started")
	}
	closeDone := make(chan error, 1)
	go func() { closeDone <- s.Close() }()
	select {
	case <-stopEntered:
	case <-time.After(time.Second):
		t.Fatal("close did not stop find")
	}
	connected := s.Snapshot().Connected
	select {
	case <-closeDone:
		t.Error("Close returned before STOP completed")
	default:
	}
	close(releaseStop)
	if !connected {
		t.Error("transport closed before STOP completed")
	}
	select {
	case err := <-closeDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("Close blocked")
	}
	if err := <-findDone; err != nil {
		t.Fatal(err)
	}
	if s.Snapshot().Connected {
		t.Fatal("transport left open")
	}
}
