//go:build darwin && systray

package tray

import (
	"context"

	"tws_manager/internal/session"
	"tws_manager/internal/spp"
	"tws_manager/internal/ui/tray/macosnative"
)

const (
	menuShowWindow = 1
	menuRefresh    = 2
	menuReconnect  = 3
	menuDisconnect = 4
	menuQuit       = 5
)

// Run adds a menu-bar status item using native Cocoa APIs. Unlike getlantern/systray,
// this does not take over NSApplication — Gio keeps the main event loop.
func Run(ctx context.Context, s *session.Session, opts Options) {
	if opts.AppName == "" {
		opts.AppName = "tws_manager"
	}

	menuClicks := make(chan int32, 8)
	macosnative.MenuClickHandler = func(tag int32) {
		select {
		case menuClicks <- tag:
		default:
		}
	}

	macosnative.ScheduleInit(opts.AppName, iconPNG, opts.OnShowWindow != nil, opts.OnReconnect != nil)
	applyDarwin(s.Snapshot())

	events := s.Subscribe()
	go darwinMenuLoop(ctx, s, opts, menuClicks)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-events:
				if !ok {
					return
				}
				applyDarwin(s.Snapshot())
			}
		}
	}()
	<-ctx.Done()
}

func darwinMenuLoop(ctx context.Context, s *session.Session, opts Options, menuClicks <-chan int32) {
	for {
		select {
		case <-ctx.Done():
			return
		case tag := <-menuClicks:
			switch tag {
			case menuShowWindow:
				if opts.OnShowWindow != nil {
					opts.OnShowWindow()
				}
			case menuRefresh:
				_ = s.SendCommand(spp.CmdGetBattery, session.Meta{Source: "tray", Trigger: "battery refresh"})
			case menuReconnect:
				if opts.OnReconnect != nil {
					go opts.OnReconnect()
				}
			case menuDisconnect:
				_ = s.Close()
			case menuQuit:
				if opts.OnQuit != nil {
					opts.OnQuit()
				} else {
					_ = s.Close()
				}
				return
			}
		}
	}
}

func applyDarwin(snap session.Snapshot) {
	macosnative.SetStatus(statusTitle(snap))
	macosnative.SetBattery("Battery: " + formatBatteries(snap.Batteries))
	macosnative.SetTooltip(tooltipForSnapshot(snap))
}
