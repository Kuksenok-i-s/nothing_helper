//go:build gio

package companion

import (
	"context"
	"image"
	"testing"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

func TestEarbudClickStartsAndStopsWithoutConfirmation(t *testing.T) {
	b := &cancelFinder{fakeBackend: connectedBackend(t), started: make(chan struct{}), stopped: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := newController(ctx, Options{}, b)
	go c.worker()
	v := newView()
	v.p = colors(false)
	var ops op.Ops
	draw := func() {
		ops.Reset()
		gtx := layout.Context{Ops: &ops, Now: time.Now(), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(image.Pt(400, 220))}
		v.deviceArt(gtx, c, c.Snapshot())
	}
	v.click("find-right").Click()
	draw()
	select {
	case <-b.started:
	case <-time.After(time.Second):
		t.Fatal("earbud click did not start search directly")
	}
	if s := c.Snapshot(); s.FindSide != "right" || !s.Busy {
		t.Fatal("expected active right earbud")
	}
	// A busy worker must not disable stopping via the same silhouette.
	v.click("find-right").Click()
	draw()
	select {
	case <-b.stopped:
	case <-time.After(time.Second):
		t.Fatal("second click did not stop active search")
	}
}
