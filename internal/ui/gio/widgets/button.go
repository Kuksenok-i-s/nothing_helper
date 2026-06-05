//go:build gio

package widgets

import (
	"image/color"

	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"tws_manager/internal/ui/gio/theme"
)

// Variant controls button appearance.
type Variant int

const (
	VariantPrimary Variant = iota
	VariantSecondary
	VariantGhost
	VariantDanger
)

// Button renders a styled button and reports click.
//
// The click is read with Clicked BEFORE the clickable is laid out: in Gio
// widget.Clickable.Layout drains all pending click events internally, so a
// Clicked call placed after Layout would always observe false.
func Button(gtx layout.Context, th *material.Theme, click *widget.Clickable, label string, v Variant) (layout.Dimensions, bool) {
	clicked := click != nil && click.Clicked(gtx)

	style := material.Button(th, click, label)
	style.CornerRadius = theme.Corner
	style.Inset = layout.Inset{Top: 8, Bottom: 8, Left: 14, Right: 14}

	switch v {
	case VariantPrimary:
		style.Background = theme.Accent
		style.Color = theme.Fg
	case VariantSecondary:
		style.Background = theme.SurfaceAlt
		style.Color = theme.Fg
	case VariantGhost:
		style.Background = color.NRGBA{}
		style.Color = theme.FgMuted
	case VariantDanger:
		style.Background = theme.AccentSoft
		style.Color = theme.Error
	}

	dims := style.Layout(gtx)
	return dims, clicked
}

// Row lays out buttons horizontally with spacing.
func Row(gtx layout.Context, th *material.Theme, items ...layout.Widget) layout.Dimensions {
	children := make([]layout.FlexChild, 0, len(items)*2-1)
	for i, w := range items {
		if i > 0 {
			children = append(children, layout.Rigid(layout.Spacer{Width: theme.GapSM}.Layout))
		}
		w := w
		children = append(children, layout.Rigid(w))
	}
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx, children...)
}
