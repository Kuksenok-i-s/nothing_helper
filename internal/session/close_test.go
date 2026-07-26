package session

import (
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"tws_manager/internal/bt"
)

type blockingCloser struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (c *blockingCloser) Read([]byte) (int, error) {
	<-c.release
	return 0, io.EOF
}

func (c *blockingCloser) Write(p []byte) (int, error) { return len(p), nil }

func (c *blockingCloser) Close() error {
	c.once.Do(func() { close(c.started) })
	<-c.release
	return nil
}

func TestCloseReleasesLockBeforeTransportClose(t *testing.T) {
	s := New(nil, false, false)
	blocker := &blockingCloser{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	s.AttachTestLink(
		bt.NewTestTransport(blocker, "AA:BB:CC:DD:EE:FF", 15, "test"),
		bt.Device{MAC: "AA:BB:CC:DD:EE:FF"},
	)

	done := make(chan error, 1)
	go func() { done <- s.Close() }()

	select {
	case <-blocker.started:
	case <-time.After(time.Second):
		t.Fatal("transport.Close was not entered")
	}

	snapDone := make(chan struct{})
	go func() {
		_ = s.Snapshot()
		close(snapDone)
	}()
	select {
	case <-snapDone:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Snapshot blocked while transport.Close was stuck")
	}

	close(blocker.release)
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Close() = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Close did not return after transport unblocked")
	}
	if s.Snapshot().Connected {
		t.Fatal("expected disconnected after Close")
	}
}

func TestCloseTimesOutStuckTransport(t *testing.T) {
	old := transportCloseTimeout
	transportCloseTimeout = 30 * time.Millisecond
	t.Cleanup(func() { transportCloseTimeout = old })

	s := New(nil, false, false)
	blocker := &blockingCloser{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	s.AttachTestLink(
		bt.NewTestTransport(blocker, "AA:BB:CC:DD:EE:FF", 15, "test"),
		bt.Device{MAC: "AA:BB:CC:DD:EE:FF"},
	)

	start := time.Now()
	err := s.Close()
	elapsed := time.Since(start)
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("Close() = %v, want timeout", err)
	}
	if elapsed > 300*time.Millisecond {
		t.Fatalf("Close took %s, want near transportCloseTimeout", elapsed)
	}
	if s.Snapshot().Connected {
		t.Fatal("expected disconnected even when transport close times out")
	}
	close(blocker.release)
}

func TestCloseCloserWithTimeoutNil(t *testing.T) {
	if err := closeCloserWithTimeout(nil, time.Millisecond); err != nil {
		t.Fatalf("nil closer: %v", err)
	}
}
