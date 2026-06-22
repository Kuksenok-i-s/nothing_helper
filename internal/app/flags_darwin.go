//go:build darwin

package app

import "flag"

func applyProfileDefaults(fs *flag.FlagSet, v *FlagValues, profile Profile) {
	autoDefault := false
	notifyDefault := false
	if profile == ProfileGUI {
		autoDefault = true
		notifyDefault = true
	}
	fs.StringVar(&v.DevicePath, "device", "", "transport ref (MAC or rfcomm:MAC:CH); optional on macOS")
	fs.StringVar(&v.PrivilegeHelper, "privilege-helper", "none", "privilege backend (darwin: none only)")
	_ = autoDefault
	_ = notifyDefault
}
