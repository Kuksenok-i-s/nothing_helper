//go:build systray

package tray

import (
	"context"
	"fmt"
	"strings"

	"github.com/getlantern/systray"

	"tws_manager/internal/session"
	"tws_manager/internal/spp"
)

type Options struct {
	AppName string
	// OnReconnect, if set, is invoked by the "Reconnect" menu item to trigger an
	// auto-discovery/connect attempt.
	OnReconnect func()
	// OnShowWindow, if set, is invoked by the "Show window" menu item.
	OnShowWindow func()
	// OnQuit, if set, is invoked by the "Quit" menu item to cancel the app context.
	OnQuit func()
}

func Run(ctx context.Context, s *session.Session, opts Options) {
	if opts.AppName == "" {
		opts.AppName = "tws_manager"
	}
	systray.Run(func() { onReady(ctx, s, opts) }, func() {})
}

func onReady(ctx context.Context, s *session.Session, opts Options) {
	systray.SetTitle("tws_manager")
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
	name := snap.Model.Product
	if name == "" {
		name = snap.Model.Codename
	}
	if name == "" {
		name = snap.Device.Name
	}
	if snap.Connected {
		if name == "" {
			name = "device"
		}
		status.SetTitle("Connected: " + name)
	} else {
		status.SetTitle("Disconnected")
	}

	bat := formatBatteries(snap.Batteries)
	battery.SetTitle("Battery: " + bat)
	systray.SetTooltip(tooltip(snap))
}

func tooltip(snap session.Snapshot) string {
	name := snap.Model.Codename
	if name == "" {
		name = snap.Device.Name
	}
	if name == "" {
		name = "tws_manager"
	}
	return strings.TrimSpace(fmt.Sprintf("%s · %s", name, formatBatteries(snap.Batteries)))
}

func formatBatteries(data map[string]spp.Battery) string {
	if len(data) == 0 {
		return "n/a"
	}
	parts := make([]string, 0, 3)
	labels := map[string]string{"left": "L", "right": "R", "case": "C", "stereo": "S"}
	for _, key := range []string{"left", "right", "case", "stereo"} {
		if b, ok := data[key]; ok {
			tag := fmt.Sprintf("%s%d%%", labels[key], b.Percent)
			if b.Charging {
				tag += "+"
			}
			parts = append(parts, tag)
		}
	}
	return strings.Join(parts, " ")
}
