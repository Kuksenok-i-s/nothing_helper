package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"nothing_helper/internal/app"
	"nothing_helper/internal/bt"
	"nothing_helper/internal/connect"
	"nothing_helper/internal/ui/companion"
	"nothing_helper/internal/ui/tray"
)

func main() { os.Exit(runMain(os.Args[1:])) }
func runMain(args []string) int {
	fs := flag.NewFlagSet("nothing_helper", flag.ContinueOnError)
	flags := app.RegisterFlags(fs, app.ProfileGUI)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if !companion.Available() {
		fmt.Fprintln(os.Stderr, "GUI is not included; build with make build or go run -tags gio ./cmd/nothing_helper")
		return 1
	}
	cfg, err := app.ConfigFromFlags(flags)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	err = app.Run(ctx, cfg, func(ctx context.Context, rt *app.Runtime) error { return run(ctx, rt, stop) })
	if err != nil && ctx.Err() == nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}
func run(ctx context.Context, rt *app.Runtime, stop context.CancelFunc) error {
	services, err := app.WireServices(ctx, rt)
	if err != nil {
		return err
	}
	var initial bt.Device
	if rt.Config.Address != "" {
		initial, err = connect.DeviceFromAddress(rt.Config.Address, rt.Config.Channel)
		if err != nil {
			return err
		}
	}
	show := make(chan struct{}, 1)
	signalShow := func() {
		select {
		case show <- struct{}{}:
		default:
		}
	}
	opts := companion.Options{Manager: services.Manager, CaptureDir: rt.Config.CaptureDir, LogRaw: rt.Config.LogRaw, AutoConnect: rt.Config.AutoDiscover, InitialDevice: initial, PCPrimary: services.PCPrimaryMode, HideToTray: tray.Available(), ShowCh: show, OnQuit: func() {
		stop()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = rt.Shutdown(shutdownCtx)
	}}
	controller := companion.NewController(ctx, opts)
	opts.Controller = controller
	go tray.Run(ctx, rt.Session, tray.Options{AppName: "Nothing_helper", OnRefresh: controller.Refresh, OnDisconnect: controller.Disconnect, OnShowWindow: signalShow, OnReconnect: func() { controller.SetAuto(true); controller.Connect(); signalShow() }, OnQuit: stop})
	return companion.Run(ctx, opts)
}
