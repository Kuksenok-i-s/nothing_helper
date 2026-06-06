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
	"tws_manager/internal/dualpolicy"
	"tws_manager/internal/notify"
	"tws_manager/internal/ui/tray"
	"tws_manager/internal/ui/tui"
)

func main() {
	devicePath := flag.String("device", "/dev/rfcomm0", "RFCOMM device")
	address := flag.String("addr", "", "Bluetooth device MAC; skips discovery and binds/open RFCOMM")
	channel := flag.Int("channel", 15, "RFCOMM channel used when creating --device with --addr")
	tracePath := flag.String("log", "", "write TX/RX trace events as NDJSON")
	modelName := flag.String("model", "", "known model codename, product name, or Fast Pair ID")
	allowUnsafe := flag.Bool("unsafe", false, "allow unsafe SET/scan actions in UI")
	noProbe := flag.Bool("no-probe", false, "skip automatic identity/battery probes after connect")
	logRaw := flag.Bool("log-raw", false, "include raw packet bytes in trace/export logs")
	queryEvery := flag.Duration("query-every", 0, "send GET_BATTERY periodically, e.g. 30s")
	captureDir := flag.String("capture-dir", "captures", "directory for JSON packet exports")
	autoDiscover := flag.Bool("auto", false, "auto-discover and connect to a Nothing device")
	notifyEnabled := flag.Bool("notify", false, "show desktop notifications for battery/connection events")
	privilegeHelper := flag.String("privilege-helper", "sudo", "privilege backend for rfcomm operations: sudo|polkit|auto|none")
	privilegeHelperPath := flag.String("privilege-helper-path", "", "optional absolute path to polkit helper binary")
	pcPrimary := flag.String("pc-primary", "ask", "dual PC-primary policy: ask|off")
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
	cfg.PCPrimary = *pcPrimary

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx, cfg, run); err != nil && ctx.Err() == nil {
		fatalf("%v", err)
	}
}

func run(ctx context.Context, rt *app.Runtime) error {
	pcPrimaryMode, err := dualpolicy.ParseMode(rt.Config.PCPrimary)
	if err != nil {
		return err
	}
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
		go notify.Run(ctx, rt.Session, notify.Options{AppName: "Nothing Ear"})
	}

	if rt.Config.AutoDiscover && rt.Config.Address == "" {
		go mgr.AutoConnect(ctx, connect.AutoOptions{OnStatus: func(msg string) {
			fmt.Fprintln(os.Stderr, msg)
		}})
	} else {
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
			_ = mgr.ConnectBest(ctx, func(msg string) { fmt.Fprintln(os.Stderr, msg) })
		},
	})
	return tui.Run(ctx, rt.Session, tui.Options{
		Manager:      mgr,
		CaptureDir:   rt.Config.CaptureDir,
		AllowUnsafe:  rt.Config.AllowUnsafe,
		LogRaw:       rt.Config.LogRaw,
		AutoDiscover: rt.Config.AutoDiscover,
		PCPrimary:    pcPrimaryMode,
		Ctx:          ctx,
	})
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}
