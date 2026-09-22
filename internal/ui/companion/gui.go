//go:build gio

package companion

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/op"
	"gioui.org/unit"
)

func Available() bool { return true }

// Run owns the native main loop. Window closure hides the companion only when
// a real tray is available; the tray's Quit action always tears down the session.
func Run(ctx context.Context, opts Options) error {
	if opts.Manager == nil {
		return fmt.Errorf("connection manager is required")
	}
	c := opts.Controller
	if c == nil {
		c = NewController(ctx, opts)
	}
	c.Start()
	var mu sync.Mutex
	var current *app.Window
	var redraw redrawState
	wake := make(chan struct{}, 1)
	go func() {
		for {
			select {
			case <-ctx.Done():
				mu.Lock()
				w := current
				mu.Unlock()
				if w != nil {
					w.Perform(system.ActionClose)
				}
				return
			case <-c.Changed():
				mu.Lock()
				w := current
				mu.Unlock()
				if w != nil && redraw.changed(c.Snapshot(), time.Now()) {
					w.Invalidate()
				}
			case <-opts.ShowCh:
				mu.Lock()
				w := current
				mu.Unlock()
				if w != nil {
					w.Perform(system.ActionRaise)
				} else {
					select {
					case wake <- struct{}{}:
					default:
					}
				}
			}
		}
	}()
	go func() {
		ui := newView()
		for {
			if ctx.Err() != nil {
				finish(opts, 0)
				return
			}
			w := new(app.Window)
			w.Option(app.Title("Nothing_helper"), app.Decorated(true), app.Size(unit.Dp(430), unit.Dp(780)), app.MinSize(unit.Dp(340), unit.Dp(480)))
			mu.Lock()
			current = w
			mu.Unlock()
			err := windowLoop(ctx, w, c, ui, &redraw)
			mu.Lock()
			current = nil
			mu.Unlock()
			if err != nil {
				err = retryRenderer(err, opts.OnQuit)
				fmt.Fprintln(os.Stderr, "companion:", err)
				finish(opts, 1)
				return
			}
			if !opts.HideToTray || ctx.Err() != nil {
				finish(opts, 0)
				return
			}
			select {
			case <-ctx.Done():
				finish(opts, 0)
				return
			case <-wake:
			}
		}
	}()
	app.Main()
	return nil
}
func finish(opts Options, code int) {
	if opts.OnQuit != nil {
		opts.OnQuit()
	}
	os.Exit(code)
}
func windowLoop(ctx context.Context, w *app.Window, c *Controller, ui *view, redraw *redrawState) error {
	var ops op.Ops
	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			if ctx.Err() != nil {
				w.Perform(system.ActionClose)
				continue
			}
			gtx := app.NewContext(&ops, e)
			snap := c.Snapshot()
			redraw.record(snap, ui.page, gtx.Now)
			ui.Layout(gtx, c, snap)
			e.Frame(gtx.Ops)
		}
	}
}
