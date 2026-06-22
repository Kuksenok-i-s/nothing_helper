//go:build linux

package app

import "flag"

func applyProfileDefaults(fs *flag.FlagSet, v *FlagValues, profile Profile) {
	autoDefault := false
	notifyDefault := false
	privDefault := "sudo"
	if profile == ProfileGUI {
		autoDefault = true
		notifyDefault = true
		privDefault = "auto"
	}
	fs.StringVar(&v.DevicePath, "device", "/dev/rfcomm0", "RFCOMM device")
	fs.StringVar(&v.PrivilegeHelper, "privilege-helper", privDefault, "privilege backend for rfcomm operations: sudo|polkit|auto|none")
	_ = autoDefault
	_ = notifyDefault
}
