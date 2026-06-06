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

	// OnExit runs exactly once after the window event loop ends, just before the
	// process terminates. On desktop app.Main never returns, so Run exits the
	// process itself; OnExit lets the caller flush logs and close the session.
	OnExit func()
}
