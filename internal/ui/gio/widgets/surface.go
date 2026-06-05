//go:build gio

package widgets

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"tws_manager/internal/ui/gio/theme"
)

// FillBackground paints the full constraints area.
func FillBackground(gtx layout.Context, bg color.NRGBA) {
	defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
	paint.Fill(gtx.Ops, bg)
}

// Card wraps content in a rounded surface panel.
func Card(gtx layout.Context, bg color.NRGBA, w layout.Widget) layout.Dimensions {
	radius := gtx.Dp(theme.Corner)
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			rr := clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, radius)
			defer rr.Push(gtx.Ops).Pop()
			paint.Fill(gtx.Ops, bg)
			return layout.Dimensions{Size: gtx.Constraints.Min}
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{
				Top: theme.GapMD, Bottom: theme.GapMD,
				Left: theme.GapLG, Right: theme.GapLG,
			}.Layout(gtx, w)
		}),
	)
}

// InsetPanel adds uniform padding around content.
func InsetPanel(gtx layout.Context, w layout.Widget) layout.Dimensions {
	return layout.Inset{
		Top: theme.GapLG, Bottom: theme.GapLG,
		Left: theme.GapLG, Right: theme.GapLG,
	}.Layout(gtx, w)
}
