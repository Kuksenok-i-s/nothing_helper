//go:build gio

package gio

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/op"
	"gioui.org/unit"

	"tws_manager/internal/ui/gio/config"
	"tws_manager/internal/ui/gio/state"
	"tws_manager/internal/ui/gio/theme"
	"tws_manager/internal/ui/gio/view"
)

// Options configures the Gio window.
type Options = config.Options

var errWindowHidden = errors.New("window hidden to tray")

// Run starts the Gio application window until closed or ctx is cancelled.
//
// app.Main must run on the main goroutine and, on desktop platforms, never
// returns. The window event loop therefore runs in its own goroutine and is
// responsible for terminating the process on real shutdown: relying on
// app.Main to return (and on a deferred shutdown) would leave the process
// hanging after the window closes.
func Run(ctx context.Context, opts Options) error {
	if opts.AppName == "" {
		opts.AppName = "tws_manager"
	}

	var windowMu sync.Mutex
	var current *app.Window
	setCurrent := func(w *app.Window) {
		windowMu.Lock()
		current = w
		windowMu.Unlock()
	}
	closeCurrent := func() {
		windowMu.Lock()
		w := current
		windowMu.Unlock()
		if w != nil {
			w.Perform(system.ActionClose)
		}
	}

	go func() {
		<-ctx.Done()
		closeCurrent()
	}()

	go func() {
		st := state.New(ctx, nil, opts, opts.Manager.Session())
		go st.PumpEvents(ctx)

		for {
			if ctx.Err() != nil {
				quit(opts, 0)
				return
			}

			w := newWindow(opts)
			setCurrent(w)
			st.SetWindow(w)

			err := runWindow(ctx, w, st, opts)
			setCurrent(nil)
			st.SetWindow(nil)

			if ctx.Err() != nil {
				quit(opts, exitCode(err))
				return
			}
			if !errors.Is(err, errWindowHidden) {
				if err != nil {
					fmt.Fprintln(os.Stderr, "gio:", err)
					quit(opts, 1)
				} else {
					quit(opts, 0)
				}
				return
			}
			if !opts.HideToTray {
				quit(opts, 0)
				return
			}

			select {
			case <-ctx.Done():
				quit(opts, 0)
				return
			case <-opts.ShowCh:
			}
		}
	}()

	app.Main()
	return nil
}

func newWindow(opts Options) *app.Window {
	w := new(app.Window)
	w.Option(app.Title(opts.AppName), app.Size(unit.Dp(1024), unit.Dp(720)))
	return w
}

func quit(opts Options, code int) {
	if opts.OnQuit != nil {
		opts.OnQuit()
	}
	os.Exit(code)
}

func exitCode(err error) int {
	if err != nil && !errors.Is(err, context.Canceled) {
		return 1
	}
	return 0
}

func runWindow(ctx context.Context, w *app.Window, st *state.State, opts Options) error {
	th := theme.New()
	var ops op.Ops

	showDone := make(chan struct{})
	var showOnce sync.Once
	stopShow := func() { showOnce.Do(func() { close(showDone) }) }
	if opts.ShowCh != nil {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-showDone:
					return
				case <-opts.ShowCh:
					w.Perform(system.ActionRaise)
				}
			}
		}()
	}
	defer stopShow()

	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			stopShow()
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return errWindowHidden
		case app.FrameEvent:
			if ctx.Err() != nil {
				w.Perform(system.ActionClose)
				continue
			}
			gtx := app.NewContext(&ops, e)
			view.Layout(gtx, th, st)
			e.Frame(gtx.Ops)
		}
	}
}
