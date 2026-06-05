//go:build gio

package theme

import (
	"image/color"

	"gioui.org/unit"
	"gioui.org/widget/material"
)

// Spacing and layout constants.
const (
	GapXS      = unit.Dp(4)
	GapSM      = unit.Dp(8)
	GapMD      = unit.Dp(12)
	GapLG      = unit.Dp(16)
	GapXL      = unit.Dp(24)
	SidebarW   = unit.Dp(220)
	StatusW    = unit.Dp(300)
	HeaderH    = unit.Dp(120)
	Corner     = unit.Dp(8)
	MaxCmdBtns = 12
)

// Colors - dark minimal palette.
var (
	Bg         = color.NRGBA{R: 0x0E, G: 0x0E, B: 0x0E, A: 0xFF}
	Surface    = color.NRGBA{R: 0x18, G: 0x18, B: 0x18, A: 0xFF}
	SurfaceAlt = color.NRGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xFF}
	Border     = color.NRGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xFF}
	Fg         = color.NRGBA{R: 0xF2, G: 0xF2, B: 0xF2, A: 0xFF}
	FgMuted    = color.NRGBA{R: 0x99, G: 0x99, B: 0x99, A: 0xFF}
	Accent     = color.NRGBA{R: 0xE0, G: 0x32, B: 0x32, A: 0xFF}
	AccentSoft = color.NRGBA{R: 0x5A, G: 0x18, B: 0x18, A: 0xFF}
	Success    = color.NRGBA{R: 0x4C, G: 0xAF, B: 0x50, A: 0xFF}
	Error      = color.NRGBA{R: 0xEF, G: 0x53, B: 0x50, A: 0xFF}
)

// New returns a dark Material theme tuned for tws_manager.
func New() *material.Theme {
	th := material.NewTheme()
	th.Palette = material.Palette{
		Bg:         Bg,
		Fg:         Fg,
		ContrastBg: Accent,
		ContrastFg: Fg,
	}
	return th
}
