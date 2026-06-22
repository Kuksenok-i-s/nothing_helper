//go:build darwin && systray

package tray

import (
	"context"
	"unsafe"

	"tws_manager/internal/session"
	"tws_manager/internal/spp"
)

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa
#include <stdlib.h>

void tray_darwin_schedule_init(const char *tooltip, const void *iconData, int iconLen,
                               int showWindow, int showReconnect);
void tray_darwin_set_status(const char *title);
void tray_darwin_set_battery(const char *title);
void tray_darwin_set_tooltip(const char *tooltip);
*/
import "C"

const (
	menuShowWindow = 1
	menuRefresh    = 2
	menuReconnect  = 3
	menuDisconnect = 4
	menuQuit       = 5
)

var menuClicks = make(chan int32, 8)

//export tray_go_menu_click
func tray_go_menu_click(tag int32) {
	select {
	case menuClicks <- tag:
	default:
	}
}

// Run adds a menu-bar status item using native Cocoa APIs. Unlike getlantern/systray,
// this does not take over NSApplication — Gio keeps the main event loop.
func Run(ctx context.Context, s *session.Session, opts Options) {
	if opts.AppName == "" {
		opts.AppName = "tws_manager"
	}

	showWindow := 0
	if opts.OnShowWindow != nil {
		showWindow = 1
	}
	showReconnect := 0
	if opts.OnReconnect != nil {
		showReconnect = 1
	}

	var iconPtr unsafe.Pointer
	iconLen := 0
	if len(iconPNG) > 0 {
		iconPtr = unsafe.Pointer(&iconPNG[0])
		iconLen = len(iconPNG)
	}
	tooltip := C.CString(opts.AppName)
	defer C.free(unsafe.Pointer(tooltip))
	C.tray_darwin_schedule_init(tooltip, iconPtr, C.int(iconLen),
		C.int(showWindow), C.int(showReconnect))

	applyDarwin(s.Snapshot())

	events := s.Subscribe()
	go darwinMenuLoop(ctx, s, opts)
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

func darwinMenuLoop(ctx context.Context, s *session.Session, opts Options) {
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
	st := C.CString(statusTitle(snap))
	defer C.free(unsafe.Pointer(st))
	bat := C.CString("Battery: " + formatBatteries(snap.Batteries))
	defer C.free(unsafe.Pointer(bat))
	tip := C.CString(tooltipForSnapshot(snap))
	defer C.free(unsafe.Pointer(tip))
	C.tray_darwin_set_status(st)
	C.tray_darwin_set_battery(bat)
	C.tray_darwin_set_tooltip(tip)
}
