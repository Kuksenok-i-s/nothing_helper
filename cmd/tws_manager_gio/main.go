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
	"tws_manager/internal/notify"
	"tws_manager/internal/ui/gio"
	"tws_manager/internal/ui/tray"
)

func main() {
	devicePath := flag.String("device", "/dev/rfcomm0", "RFCOMM device")
	address := flag.String("addr", "", "Bluetooth device MAC for RFCOMM bind")
	channel := flag.Int("channel", 15, "RFCOMM channel")
	tracePath := flag.String("log", "", "write TX/RX trace events as NDJSON")
	modelName := flag.String("model", "", "known model codename, product name, or Fast Pair ID")
	allowUnsafe := flag.Bool("unsafe", false, "allow unsafe SET/scan actions in UI")
	noProbe := flag.Bool("no-probe", false, "skip automatic identity/battery probes after connect")
	logRaw := flag.Bool("log-raw", false, "include raw packet bytes in trace/export logs")
	queryEvery := flag.Duration("query-every", 0, "send GET_BATTERY periodically, e.g. 30s")
	captureDir := flag.String("capture-dir", "captures", "directory for JSON packet exports")
	autoDiscover := flag.Bool("auto", true, "auto-discover and connect to a compatible TWS device")
	notifyEnabled := flag.Bool("notify", true, "show desktop notifications for battery/connection events")
	privilegeHelper := flag.String("privilege-helper", "auto", "privilege backend for rfcomm operations: sudo|polkit|auto|none")
	privilegeHelperPath := flag.String("privilege-helper-path", "", "optional absolute path to polkit helper binary")
	flag.Parse()

	cfg, err := app.ValidateFlags(*devicePath, *address, *channel, *captureDir, *tracePath)
	if err != nil {
		fatalf("invalid flags: %v", err)
	}
	cfg.ModelName = *modelName
	cfg.AllowUnsafe = *allowUnsafe
	cfg.ProbeEnabled = !*noProbe
	cfg.LogRaw = *logRaw
	cfg.QueryEvery = *queryEvery
	cfg.AutoDiscover = *autoDiscover
	cfg.Notify = *notifyEnabled
	cfg.PrivilegeMode = *privilegeHelper
	cfg.PrivilegeHelperPath = *privilegeHelperPath

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx, cfg, run); err != nil && ctx.Err() == nil {
		fatalf("%v", err)
	}
}

func run(ctx context.Context, rt *app.Runtime) error {
	if err := bt.ConfigurePrivileges(rt.Config.PrivilegeMode, rt.Config.PrivilegeHelperPath); err != nil {
		return err
	}
	if bt.CurrentPrivilegeMode() == bt.PrivilegeModeSudo {
		if ok, err := app.WarmupPrivileges(); err != nil {
			app.Warnf("%v (RFCOMM auto-recovery may prompt for sudo again)", err)
		} else if ok {
			fmt.Fprintln(os.Stderr, "sudo credentials cached for this session")
		}
	}

	mgr := connect.New(rt.Session, connect.Options{
		RFCOMMPath: rt.Config.RFCOMMDevice,
		Channel:    rt.Config.Channel,
	})

	if rt.Config.Notify {
		go notify.Run(ctx, rt.Session, notify.Options{AppName: "tws_manager"})
	}

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

	go tray.Run(ctx, rt.Session, tray.Options{
		AppName: "tws_manager",
		OnReconnect: func() {
			_ = mgr.ConnectBest(ctx, func(msg string) { fmt.Fprintln(os.Stderr, msg) })
		},
	})
	return gio.Run(ctx, gio.Options{
		Manager:       mgr,
		CaptureDir:    rt.Config.CaptureDir,
		AllowUnsafe:   rt.Config.AllowUnsafe,
		LogRaw:        rt.Config.LogRaw,
		AppName:       "tws_manager",
		AutoConnect:   rt.Config.AutoDiscover,
		InitialDevice: initialDevice,
		// Run terminates the process itself (app.Main never returns), so flush
		// the trace log and close the RFCOMM session here. Shutdown is
		// idempotent, so this is safe alongside app.Run's signal handler.
		OnExit: func() { _ = rt.Shutdown(context.Background()) },
	})
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}
