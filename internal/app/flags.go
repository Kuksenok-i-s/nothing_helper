package app

import (
	"flag"
	"time"
)

// Profile selects CLI defaults for TUI vs Gio entrypoints.
type Profile int

const (
	ProfileCLI Profile = iota
	ProfileGUI
)

// FlagValues holds parsed CLI flag values before validation.
type FlagValues struct {
	DevicePath          string
	Address             string
	Channel             int
	TracePath           string
	ModelName           string
	AllowUnsafe         bool
	NoProbe             bool
	LogRaw              bool
	QueryEvery          time.Duration
	CaptureDir          string
	AutoDiscover        bool
	Notify              bool
	PrivilegeHelper     string
	PrivilegeHelperPath string
	PCPrimary           string
	PprofAddr           string
}

// RegisterFlags defines shared flags and applies profile-specific defaults.
func RegisterFlags(fs *flag.FlagSet, profile Profile) *FlagValues {
	v := &FlagValues{}
	applyProfileDefaults(fs, v, profile)
	autoDefault := profile == ProfileGUI
	notifyDefault := profile == ProfileGUI
	fs.StringVar(&v.Address, "addr", "", "Bluetooth device MAC; skips discovery and binds/open RFCOMM")
	fs.IntVar(&v.Channel, "channel", 15, "RFCOMM channel used when creating --device with --addr")
	fs.StringVar(&v.TracePath, "log", "", "write TX/RX trace events as NDJSON")
	fs.StringVar(&v.ModelName, "model", "", "known model codename, product name, or Fast Pair ID")
	fs.BoolVar(&v.AllowUnsafe, "unsafe", false, "allow unsafe SET/scan actions in UI")
	fs.BoolVar(&v.NoProbe, "no-probe", false, "skip automatic identity/battery probes after connect")
	fs.BoolVar(&v.LogRaw, "log-raw", false, "include raw packet bytes in trace/export logs")
	fs.DurationVar(&v.QueryEvery, "query-every", 0, "send GET_BATTERY periodically, e.g. 30s")
	fs.StringVar(&v.CaptureDir, "capture-dir", captureDirDefault(profile), "directory for JSON packet exports")
	fs.BoolVar(&v.AutoDiscover, "auto", autoDefault, "auto-discover and connect to a Nothing device")
	fs.BoolVar(&v.Notify, "notify", notifyDefault, "show desktop notifications for battery/connection events")
	fs.StringVar(&v.PrivilegeHelperPath, "privilege-helper-path", "", "optional absolute path to polkit helper binary")
	fs.StringVar(&v.PCPrimary, "pc-primary", "ask", "dual PC-primary policy: ask|off")
	fs.StringVar(&v.PprofAddr, "pprof-addr", "", "serve Go pprof on this address, e.g. 127.0.0.1:6060 (disabled when empty)")
	return v
}

// ConfigFromFlags validates flags and returns runtime config.
func ConfigFromFlags(v *FlagValues) (Config, error) {
	cfg, err := ValidateFlags(v.DevicePath, v.Address, v.Channel, v.CaptureDir, v.TracePath)
	if err != nil {
		return Config{}, err
	}
	cfg.ModelName = v.ModelName
	cfg.AllowUnsafe = v.AllowUnsafe
	cfg.ProbeEnabled = !v.NoProbe
	cfg.LogRaw = v.LogRaw
	cfg.QueryEvery = v.QueryEvery
	cfg.AutoDiscover = v.AutoDiscover
	cfg.Notify = v.Notify
	cfg.PrivilegeMode = v.PrivilegeHelper
	cfg.PrivilegeHelperPath = v.PrivilegeHelperPath
	cfg.PCPrimary = v.PCPrimary
	cfg.PprofAddr = v.PprofAddr
	return cfg, nil
}
