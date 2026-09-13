package tray

// Options configures the tray runner.
type Options struct {
	AppName string
	// OnRefresh and OnDisconnect route actions through the companion queue.
	OnRefresh    func()
	OnDisconnect func()
	// OnReconnect, if set, is invoked by the "Reconnect" menu item to trigger an
	// auto-discovery/connect attempt.
	OnReconnect func()
	// OnShowWindow, if set, is invoked by the "Show window" menu item.
	OnShowWindow func()
	// OnQuit, if set, is invoked by the "Quit" menu item to cancel the app context.
	OnQuit func()
}
