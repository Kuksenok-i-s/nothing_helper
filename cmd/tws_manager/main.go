package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"tws_manager/internal/app"
	"tws_manager/internal/ui/tray"
	"tws_manager/internal/ui/tui"
)

func main() {
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags := app.RegisterFlags(fs, app.ProfileCLI)
	_ = fs.Parse(os.Args[1:])

	cfg, err := app.ConfigFromFlags(flags)
	if err != nil {
		fatalf("invalid flags: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx, cfg, run); err != nil && ctx.Err() == nil {
		fatalf("%v", err)
	}
}

func run(ctx context.Context, rt *app.Runtime) error {
	services, err := app.WireServices(ctx, rt)
	if err != nil {
		return err
	}
	mgr := services.Manager

	app.StartAutoConnect(ctx, mgr, rt.Config, func(msg string) {
		fmt.Fprintln(os.Stderr, msg)
	})
	if !rt.Config.AutoDiscover || rt.Config.Address != "" {
		connectDevice, ok, err := preflightRFCOMM(mgr, rt.Config.Address)
		if err != nil {
			return fmt.Errorf("rfcomm preflight: %w", err)
		}
		if ok {
			dev := connectDevice
			go func() {
				if err := mgr.Connect(ctx, dev); err != nil && ctx.Err() == nil {
					fmt.Fprintf(os.Stderr, "connect %s: %v\n", dev.MAC, err)
				}
			}()
		}
	}

	go tray.Run(ctx, rt.Session, tray.Options{
		AppName: "Nothing Ear",
		OnReconnect: func() {
			app.StartTrayReconnect(ctx, mgr, func(msg string) { fmt.Fprintln(os.Stderr, msg) })
		},
	})
	return tui.Run(ctx, rt.Session, tui.Options{
		Manager:      mgr,
		CaptureDir:   rt.Config.CaptureDir,
		AllowUnsafe:  rt.Config.AllowUnsafe,
		LogRaw:       rt.Config.LogRaw,
		AutoDiscover: rt.Config.AutoDiscover,
		PCPrimary:    services.PCPrimaryMode,
		Ctx:          ctx,
	})
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}
