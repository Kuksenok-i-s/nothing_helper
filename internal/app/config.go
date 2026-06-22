package app

import (
	"time"
)

// Config holds validated CLI/runtime settings shared by TUI and Gio entrypoints.
type Config struct {
	RFCOMMDevice        string
	Address             string
	Channel             int
	TracePath           string
	CaptureDir          string
	PrivilegeMode       string
	PrivilegeHelperPath string
	ModelName           string
	AllowUnsafe         bool
	ProbeEnabled        bool
	LogRaw              bool
	QueryEvery          time.Duration
	AutoDiscover        bool
	Notify              bool
	PCPrimary           string
}
