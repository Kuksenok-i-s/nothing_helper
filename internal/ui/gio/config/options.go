//go:build gio

package config

import (
	"tws_manager/internal/bt"
	"tws_manager/internal/connect"
	"tws_manager/internal/dualpolicy"
)

// Options configures the Gio window.
type Options struct {
	Manager       *connect.Manager
	CaptureDir    string
	AllowUnsafe   bool
	LogRaw        bool
	AppName       string
	AutoConnect   bool
	InitialDevice bt.Device
	PCPrimary     dualpolicy.Mode

	// HideToTray keeps the process alive when the user closes the window;
	// the tray icon and RFCOMM session stay active until Quit or SIGTERM.
	HideToTray bool
	// ShowCh receives signals from the tray "Show window" action (buffer 1).
	ShowCh chan struct{}

	// OnQuit runs once before the process terminates on real shutdown
	// (ctx cancelled or window closed without HideToTray). Not called on hide-to-tray.
	OnQuit func()
}
