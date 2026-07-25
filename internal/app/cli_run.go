package app

import (
	"context"
	"fmt"
	"os"

	"tws_manager/internal/connect"
	"tws_manager/internal/session"
	"tws_manager/internal/ui/tray"
	"tws_manager/internal/ui/tui"
)

var (
	runTrayHook = tray.Run
	runTUIHook  = tui.Run
)

// SetCLIHooks overrides tray/TUI entrypoints for tests. Returns a restore func.
func SetCLIHooks(tray func(context.Context, *session.Session, tray.Options), tui func(context.Context, *session.Session, tui.Options) error) func() {
	oldTray, oldTUI := runTrayHook, runTUIHook
	if tray != nil {
		runTrayHook = tray
	}
	if tui != nil {
		runTUIHook = tui
	}
	return func() {
		runTrayHook, runTUIHook = oldTray, oldTUI
	}
}

// RunCLI wires autoconnect, preflight, tray, and TUI for the CLI entrypoint.
func RunCLI(ctx context.Context, rt *Runtime, services *Services) error {
	mgr := services.Manager

	StartAutoConnect(ctx, mgr, rt.Config, func(msg string) {
		fmt.Fprintln(os.Stderr, msg)
	})
	if !rt.Config.AutoDiscover || rt.Config.Address != "" {
		if err := connect.RunPreflightConnect(ctx, mgr, rt.Config.AutoDiscover, rt.Config.Address, connect.DefaultPreflightIO(), mgr.Connect); err != nil {
			return err
		}
	}

	go runTrayHook(ctx, rt.Session, tray.Options{
		AppName: "Nothing Ear",
		OnReconnect: func() {
			StartTrayReconnect(ctx, mgr, func(msg string) { fmt.Fprintln(os.Stderr, msg) })
		},
	})
	return runTUIHook(ctx, rt.Session, tui.Options{
		Manager:      mgr,
		CaptureDir:   rt.Config.CaptureDir,
		AllowUnsafe:  rt.Config.AllowUnsafe,
		LogRaw:       rt.Config.LogRaw,
		AutoDiscover: rt.Config.AutoDiscover,
		PCPrimary:    services.PCPrimaryMode,
		Ctx:          ctx,
	})
}
