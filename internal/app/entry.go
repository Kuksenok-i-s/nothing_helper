package app

import (
	"context"
	"fmt"
	"os"

	"nothing_helper/internal/bt"
	"nothing_helper/internal/connect"
	"nothing_helper/internal/dualpolicy"
	"nothing_helper/internal/notify"
	"nothing_helper/internal/session"
)

// Services bundles connect/notify helpers wired from entrypoints.
type Services struct {
	Manager       *connect.Manager
	PCPrimaryMode dualpolicy.Mode
}

// WireServices configures privileges, optional notify watcher, and connect manager.
func WireServices(ctx context.Context, rt *Runtime) (*Services, error) {
	pcPrimaryMode, err := dualpolicy.ParseMode(rt.Config.PCPrimary)
	if err != nil {
		return nil, err
	}
	if err := bt.ConfigurePrivileges(rt.Config.PrivilegeMode, rt.Config.PrivilegeHelperPath); err != nil {
		return nil, err
	}
	if bt.CurrentPrivilegeMode() == bt.PrivilegeModeSudo {
		if ok, err := WarmupPrivileges(); err != nil {
			Warnf("%v (RFCOMM auto-recovery may prompt for sudo again)", err)
		} else if ok {
			fmt.Fprintln(os.Stderr, "sudo credentials cached for this session")
		}
	}
	mgr := connect.New(rt.Session, connect.Options{
		RFCOMMPath: rt.Config.RFCOMMDevice,
		Channel:    rt.Config.Channel,
	})
	if rt.Config.Notify {
		go notify.Run(ctx, rt.Session, notify.Options{AppName: notifyAppName(rt.Config)})
	}
	return &Services{Manager: mgr, PCPrimaryMode: pcPrimaryMode}, nil
}

func notifyAppName(cfg Config) string {
	return "Nothing Ear"
}

// StartAutoConnect launches background autoconnect when enabled.
func StartAutoConnect(ctx context.Context, mgr *connect.Manager, cfg Config, onStatus func(string)) {
	if !cfg.AutoDiscover || cfg.Address != "" {
		return
	}
	go mgr.AutoConnect(ctx, connect.AutoOptions{OnStatus: onStatus})
}

// StartTrayReconnect is a helper for tray OnReconnect callbacks.
func StartTrayReconnect(ctx context.Context, mgr *connect.Manager, onStatus func(string)) {
	_ = mgr.ConnectBest(ctx, onStatus)
}

// SessionSnapshot is a test seam around session.Snapshot.
func SessionSnapshot(sess *session.Session) session.Snapshot {
	return sess.Snapshot()
}
