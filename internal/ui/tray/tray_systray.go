//go:build systray && !darwin

package tray

import (
	"context"

	"github.com/getlantern/systray"

	"nothing_helper/internal/session"
	"nothing_helper/internal/spp"
)

func Run(ctx context.Context, s *session.Session, opts Options) {
	if opts.AppName == "" {
		opts.AppName = "Nothing_helper"
	}
	systray.Run(func() { onReady(ctx, s, opts) }, func() {})
}

func onReady(ctx context.Context, s *session.Session, opts Options) {
	systray.SetTitle("Nothing_helper")
	if len(iconPNG) > 0 {
		systray.SetIcon(iconPNG)
	}
	systray.SetTooltip(opts.AppName)

	status := systray.AddMenuItem("Disconnected", "Connection status")
	status.Disable()
	battery := systray.AddMenuItem("Battery: n/a", "Battery levels")
	battery.Disable()
	systray.AddSeparator()

	showWindow := systray.AddMenuItem("Open companion", "Open the compact controls")
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
	quit := systray.AddMenuItem("Quit", "Quit Nothing_helper")

	events := s.Subscribe()
	apply(s.Snapshot(), status, battery)

	go func() {
		for {
			select {
			case <-ctx.Done():
				// Quit the tray loop first; session teardown belongs to app.Shutdown.
				systray.Quit()
				return
			case <-showWindow.ClickedCh:
				if opts.OnShowWindow != nil {
					opts.OnShowWindow()
				}
			case <-refresh.ClickedCh:
				if opts.OnRefresh != nil {
					opts.OnRefresh()
				} else {
					_ = s.SendCommand(spp.CmdGetBattery, session.Meta{Source: "tray", Trigger: "battery refresh"})
				}
			case <-reconnect.ClickedCh:
				if opts.OnReconnect != nil {
					go opts.OnReconnect()
				}
			case <-disconnect.ClickedCh:
				if opts.OnDisconnect != nil {
					opts.OnDisconnect()
				} else {
					_ = s.Close()
				}
			case <-quit.ClickedCh:
				if opts.OnQuit != nil {
					opts.OnQuit()
				} else {
					// CLI tray has no OnQuit: drop the link in the background
					// so systray.Quit is not blocked by RFCOMM teardown.
					go func() { _ = s.Close() }()
				}
				systray.Quit()
				return
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-events:
				if !ok {
					return
				}
				apply(s.Snapshot(), status, battery)
			}
		}
	}()
}

func apply(snap session.Snapshot, status, battery *systray.MenuItem) {
	status.SetTitle(statusTitle(snap))
	bat := formatBatteries(snap.Batteries)
	battery.SetTitle("Battery: " + bat)
	systray.SetTooltip(tooltipForSnapshot(snap))
}
