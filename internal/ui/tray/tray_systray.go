//go:build systray && !darwin

package tray

import (
	"context"

	"github.com/getlantern/systray"

	"tws_manager/internal/session"
	"tws_manager/internal/spp"
)

func Run(ctx context.Context, s *session.Session, opts Options) {
	if opts.AppName == "" {
		opts.AppName = "tws_manager"
	}
	systray.Run(func() { onReady(ctx, s, opts) }, func() {})
}

func onReady(ctx context.Context, s *session.Session, opts Options) {
	systray.SetTitle("tws_manager")
	if len(iconPNG) > 0 {
		systray.SetIcon(iconPNG)
	}
	systray.SetTooltip(opts.AppName)

	status := systray.AddMenuItem("Disconnected", "Connection status")
	status.Disable()
	battery := systray.AddMenuItem("Battery: n/a", "Battery levels")
	battery.Disable()
	systray.AddSeparator()

	showWindow := systray.AddMenuItem("Show window", "Open the main window")
	if opts.OnShowWindow == nil {
		showWindow.Hide()
	}
	refresh := systray.AddMenuItem("Refresh battery", "Send GET_BATTERY")
	reconnect := systray.AddMenuItem("Reconnect", "Auto-discover and connect")
	if opts.OnReconnect == nil {
		reconnect.Hide()
	}
	disconnect := systray.AddMenuItem("Disconnect", "Close active RFCOMM connection")
	systray.AddSeparator()
	quit := systray.AddMenuItem("Quit", "Quit tws_manager")

	events := s.Subscribe()
	apply(s.Snapshot(), status, battery)

	go func() {
		for {
			select {
			case <-ctx.Done():
				_ = s.Close()
				systray.Quit()
				return
			case <-showWindow.ClickedCh:
				if opts.OnShowWindow != nil {
					opts.OnShowWindow()
				}
			case <-refresh.ClickedCh:
				_ = s.SendCommand(spp.CmdGetBattery, session.Meta{Source: "tray", Trigger: "battery refresh"})
			case <-reconnect.ClickedCh:
				if opts.OnReconnect != nil {
					go opts.OnReconnect()
				}
			case <-disconnect.ClickedCh:
				_ = s.Close()
			case <-quit.ClickedCh:
				if opts.OnQuit != nil {
					opts.OnQuit()
				} else {
					_ = s.Close()
					systray.Quit()
				}
				return
			}
		}
	}()

	go func() {
		for range events {
			apply(s.Snapshot(), status, battery)
		}
	}()
}

func apply(snap session.Snapshot, status, battery *systray.MenuItem) {
	status.SetTitle(statusTitle(snap))
	bat := formatBatteries(snap.Batteries)
	battery.SetTitle("Battery: " + bat)
	systray.SetTooltip(tooltipForSnapshot(snap))
}
