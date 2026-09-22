package companion

import (
	"context"
	"testing"
	"time"

	"nothing_helper/internal/session"
	"nothing_helper/internal/spp"
)

func TestRedrawIgnoresUnchangedPoll(t *testing.T) {
	now := time.Now()
	s := Snapshot{Session: session.Snapshot{Connected: true, Earbuds: map[string]session.EarbudState{
		"left": {EarbudStatus: spp.EarbudStatus{Connected: true, InEar: true}, UpdatedAt: now},
	}}}
	var r redrawState
	r.record(s, "sound", now)
	s.Status = "received status"
	s.Logs = []string{"poll response"}
	s.Session.Earbuds = map[string]session.EarbudState{
		"left": {EarbudStatus: spp.EarbudStatus{Connected: true, InEar: true}, UpdatedAt: now.Add(time.Second)},
	}
	if r.changed(s, now.Add(time.Second)) {
		t.Fatal("unchanged poll requested redraw")
	}
	if !r.changed(s, now.Add(32*time.Second)) {
		t.Fatal("stale sensor did not change displayed state")
	}
	s.Session.Earbuds["left"] = session.EarbudState{EarbudStatus: spp.EarbudStatus{Connected: true}, UpdatedAt: now}
	if !r.changed(s, now) {
		t.Fatal("wear change did not request redraw")
	}
}

func TestRedrawKeepsVisibleChanges(t *testing.T) {
	for _, page := range []string{"sound", "devices", "log"} {
		for _, field := range []string{"battery", "config", "connection", "busy", "error", "password", "dual", "logs"} {
			t.Run(page+"/"+field, func(t *testing.T) {
				var r redrawState
				s := Snapshot{}
				r.record(s, page, time.Now())
				switch field {
				case "battery":
					s.Session.Batteries = map[string]spp.Battery{"left": {Percent: 50}}
				case "config":
					s.Session.Config = map[string]string{"anc": "off"}
				case "connection":
					s.Session.Connected = true
				case "busy":
					s.Busy = true
				case "error":
					s.Error = "failed"
				case "password":
					s.PasswordPrompt = "password"
				case "dual":
					s.DualPrompt = "switch?"
				case "logs":
					s.Logs = []string{"packet"}
					s.Status = "received"
				}
				want := field != "logs" || page == "log"
				if got := r.changed(s, time.Now()); got != want {
					t.Fatalf("redraw = %v, want %v", got, want)
				}
			})
		}
	}
}

type pollingBackend struct {
	*fakeBackend
	started chan struct{}
	release chan struct{}
}

func (b *pollingBackend) Status() error {
	close(b.started)
	<-b.release
	return nil
}

func TestBackgroundWearPollDoesNotDisableControls(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b := &pollingBackend{fakeBackend: connectedBackend(t), started: make(chan struct{}), release: make(chan struct{})}
	defer close(b.release)
	c := newController(ctx, Options{}, b)
	c.presenter.Status = "ready"
	c.presenter.Err = "previous error"
	c.refreshWear(true)
	go c.worker()
	select {
	case <-b.started:
	case <-time.After(time.Second):
		t.Fatal("poll did not start")
	}
	s := c.Snapshot()
	if s.Busy || s.Status != "ready" || s.Error != "previous error" {
		t.Fatalf("background poll changed foreground state: %+v", s)
	}
	select {
	case <-c.Changed():
		t.Fatal("background poll requested redraw")
	default:
	}
	c.refreshWear(true)
	if len(c.queue) != 0 {
		t.Fatal("duplicate background poll queued")
	}
	if !c.Feature("anc", "off") || !c.Snapshot().Busy {
		t.Fatal("foreground action must remain available and show busy state")
	}
}
