package companion

import (
	"context"
	"testing"
	"time"
	"nothing_helper/internal/session"
	"nothing_helper/internal/spp"
)

func TestUnknownCaseChargeHidden(t *testing.T) {
	s := session.Snapshot{Connected: true, Batteries: map[string]spp.Battery{}}
	if _, ok := CaseCharge(s); ok {
		t.Fatal("missing case shown")
	}
	s.Batteries["case"] = spp.Battery{Percent: 0}
	if n, ok := CaseCharge(s); !ok || n != 0 {
		t.Fatal("known zero charge must be shown")
	}
	s.Connected = false
	if _, ok := CaseCharge(s); ok {
		t.Fatal("disconnected cached charge shown")
	}
}
func TestWearUnknownIsNotOutOfEar(t *testing.T) {
	now := time.Now()
	s := session.Snapshot{Connected: true, Earbuds: map[string]session.EarbudState{}}
	if _, ok := WearKnown(s, "left", now); ok {
		t.Fatal("missing sensor known")
	}
	s.Earbuds["left"] = session.EarbudState{EarbudStatus: spp.EarbudStatus{Connected: true}, UpdatedAt: now}
	if _, ok := WearKnown(s, "left", now); !ok {
		t.Fatal("fresh out-of-ear status unknown")
	}
	if _, ok := WearKnown(s, "left", now.Add(31*time.Second)); ok {
		t.Fatal("stale sensor known")
	}
	s.Connected = false
	if _, ok := WearKnown(s, "left", now); ok {
		t.Fatal("disconnected sensor known")
	}
}

type cancelFinder struct {
	*fakeBackend
	started chan struct{}
	stopped chan struct{}
}

func (b *cancelFinder) Find(ctx context.Context, side string) error {
	close(b.started)
	<-ctx.Done()
	close(b.stopped)
	return nil
}
func TestFindStopCancelsActiveWorker(t *testing.T) {
	b := &cancelFinder{fakeBackend: connectedBackend(t), started: make(chan struct{}), stopped: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := newController(ctx, Options{}, b)
	go c.worker()
	c.Find("left")
	select {
	case <-b.started:
	case <-time.After(time.Second):
		t.Fatal("search did not start")
	}
	if c.Snapshot().FindSide != "left" {
		t.Fatal("search state absent")
	}
	c.StopFind()
	select {
	case <-b.stopped:
	case <-time.After(time.Second):
		t.Fatal("stop blocked by worker")
	}
}
