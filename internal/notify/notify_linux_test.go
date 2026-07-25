//go:build linux

package notify

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"tws_manager/internal/session"
)

func TestSendGdbusSuccess(t *testing.T) {
	oldOut := execCommandOutput
	oldLook := execLookPath
	execLookPath = func(name string) (string, error) {
		if name == "gdbus" {
			return "/usr/bin/gdbus", nil
		}
		return "", errors.New("missing")
	}
	execCommandOutput = func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name != "gdbus" {
			return nil, fmt.Errorf("unexpected %s", name)
		}
		return []byte("(uint32 99,)\n"), nil
	}
	t.Cleanup(func() {
		execCommandOutput = oldOut
		execLookPath = oldLook
	})

	n := New("test", "icon")
	if n.backend != "gdbus" {
		t.Fatalf("backend=%q", n.backend)
	}
	id := n.sendGdbus(0, UrgencyNormal, "title", "body", "icon")
	if id != 99 {
		t.Fatalf("id=%d want 99", id)
	}
}

func TestSendGdbusFallbackToNotifySend(t *testing.T) {
	oldOut := execCommandOutput
	oldRun := execCommandRun
	oldLook := execLookPath
	oldWarn := Warnf
	execLookPath = func(name string) (string, error) {
		if name == "gdbus" || name == "notify-send" {
			return "/usr/bin/" + name, nil
		}
		return "", errors.New("missing")
	}
	calls := 0
	execCommandOutput = func(_ context.Context, name string, _ ...string) ([]byte, error) {
		if name == "gdbus" {
			return nil, errors.New("gdbus failed")
		}
		return nil, errors.New("unexpected output")
	}
	execCommandRun = func(_ context.Context, name string, _ ...string) error {
		if name == "notify-send" {
			calls++
			return nil
		}
		return errors.New("unexpected run")
	}
	var warns []string
	Warnf = func(format string, args ...any) {
		warns = append(warns, fmt.Sprintf(format, args...))
	}
	t.Cleanup(func() {
		execCommandOutput = oldOut
		execCommandRun = oldRun
		execLookPath = oldLook
		Warnf = oldWarn
	})

	n := New("test", "icon")
	id := n.sendGdbus(0, UrgencyCritical, "title", "body", "icon")
	if id != 0 || calls != 1 {
		t.Fatalf("id=%d calls=%d", id, calls)
	}
	if len(warns) == 0 || !strings.Contains(warns[0], "gdbus") {
		t.Fatalf("warns=%v", warns)
	}
}

func TestComponentLabel(t *testing.T) {
	tests := map[string]string{
		"left": "Left earbud", "right": "Right earbud", "case": "Case",
		"stereo": "Headphones", "tws": "Earbuds", "watch": "Watch",
		"id_7": "Device 7", "other": "other",
	}
	for in, want := range tests {
		if got := componentLabel(in); got != want {
			t.Fatalf("componentLabel(%q)=%q want %q", in, got, want)
		}
	}
}

func TestWatchReturnsWhenContextDone(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	events := make(chan session.Event)
	close(events)
	sv := fakeSessionView{}
	n := &Notifier{send: func(uint32, Urgency, string, string, string) uint32 { return 0 }}
	watch(ctx, events, sv, n, Options{AppName: "test"}, earbudLowLevels, caseLowLevels)
}

func TestWatchProcessesEventUntilClosed(t *testing.T) {
	ctx := context.Background()
	events := make(chan session.Event, 1)
	events <- session.Event{Kind: session.EventProgress}
	close(events)
	sv := fakeSessionView{}
	n := &Notifier{send: func(uint32, Urgency, string, string, string) uint32 { return 0 }}
	watch(ctx, events, sv, n, Options{AppName: "test"}, earbudLowLevels, caseLowLevels)
}
