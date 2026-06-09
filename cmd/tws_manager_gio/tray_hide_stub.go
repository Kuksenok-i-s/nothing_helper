//go:build gio && !systray

package main

func hideToTray() bool { return false }
