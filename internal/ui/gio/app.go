//go:build gio

package gio

import (
	"context"
	"fmt"
	"os"

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

// Run starts the Gio application window until closed or ctx is cancelled.
//
// app.Main must run on the main goroutine and, on desktop platforms, never
// returns. The window event loop therefore runs in its own goroutine and is
// responsible for terminating the process once the window is gone: relying on
// app.Main to return (and on a deferred shutdown) would leave the process
// hanging after the window closes.
func Run(ctx context.Context, opts Options) error {
	if opts.AppName == "" {
		opts.AppName = "tws_manager"
	}

	w := new(app.Window)
	w.Option(app.Title(opts.AppName), app.Size(unit.Dp(1024), unit.Dp(720)))

	// Close the window when the context is cancelled (SIGINT/SIGTERM).
	go func() {
		<-ctx.Done()
		w.Perform(system.ActionClose)
	}()

	go func() {
		err := runWindow(ctx, w, opts)
		if opts.OnExit != nil {
			opts.OnExit()
		}
		if err != nil && ctx.Err() == nil {
			fmt.Fprintln(os.Stderr, "gio:", err)
			os.Exit(1)
		}
		os.Exit(0)
	}()

	app.Main()
	return nil
}

func runWindow(ctx context.Context, w *app.Window, opts Options) error {
	th := theme.New()
	var ops op.Ops

	st := state.New(ctx, w, opts, opts.Manager.Session())
	go st.PumpEvents(ctx)

	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return e.Err
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
