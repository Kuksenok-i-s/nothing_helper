//go:build gio

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"tws_manager/internal/app"
	"tws_manager/internal/bt"
	"tws_manager/internal/connect"
	"tws_manager/internal/ui/gio"
	"tws_manager/internal/ui/tray"
)

func main() {
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags := app.RegisterFlags(fs, app.ProfileGUI)
	_ = fs.Parse(os.Args[1:])

	cfg, err := app.ConfigFromFlags(flags)
	if err != nil {
		fatalf("invalid flags: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx, cfg, func(ctx context.Context, rt *app.Runtime) error {
		return run(ctx, rt, stop)
	}); err != nil && ctx.Err() == nil {
		fatalf("%v", err)
	}
}

func run(ctx context.Context, rt *app.Runtime, stop context.CancelFunc) error {
	services, err := app.WireServices(ctx, rt)
	if err != nil {
		return err
	}
	mgr := services.Manager

	var initialDevice bt.Device
	if rt.Config.Address != "" {
		dev, err := connect.DeviceFromAddress(rt.Config.Address, rt.Config.Channel)
		if err != nil {
			return fmt.Errorf("addr: %w", err)
		}
		initialDevice = dev
	} else if !rt.Config.AutoDiscover {
		if exists, _ := mgr.RFCOMMExists(); exists {
			initialDevice = mgr.DeviceForExistingRFCOMM("")
		}
	}

	showCh := make(chan struct{}, 1)
	signalShow := func() {
		select {
		case showCh <- struct{}{}:
		default:
		}
	}

	go tray.Run(ctx, rt.Session, tray.Options{
		AppName: "tws_manager",
		OnReconnect: func() {
			app.StartTrayReconnect(ctx, mgr, func(msg string) { fmt.Fprintln(os.Stderr, msg) })
		},
		OnShowWindow: signalShow,
		OnQuit:       stop,
	})
	return gio.Run(ctx, gio.Options{
		Manager:       mgr,
		CaptureDir:    rt.Config.CaptureDir,
		AllowUnsafe:   rt.Config.AllowUnsafe,
		LogRaw:        rt.Config.LogRaw,
		AppName:       "tws_manager",
		AutoConnect:   rt.Config.AutoDiscover,
		InitialDevice: initialDevice,
		PCPrimary:     services.PCPrimaryMode,
		HideToTray:    hideToTray(),
		ShowCh:        showCh,
		OnQuit: func() { _ = rt.Shutdown(context.Background()) },
	})
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}
