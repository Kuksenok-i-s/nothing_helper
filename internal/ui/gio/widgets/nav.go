//go:build gio

package widgets

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"tws_manager/internal/ui/gio/theme"
)

// NavItem is a sidebar navigation entry.
type NavItem struct {
	Label  string
	Active bool
	Click  *widget.Clickable
}

// NavList renders a vertical navigation list.
// Returns layout dimensions and the index of a clicked item, or -1.
func NavList(gtx layout.Context, th *material.Theme, items []NavItem) (layout.Dimensions, int) {
	selected := -1
	children := make([]layout.FlexChild, 0, len(items))
	for i, item := range items {
		i, item := i, item
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			bg := color.NRGBA{}
			fg := theme.FgMuted
			if item.Active {
				bg = theme.AccentSoft
				fg = theme.Fg
			}
			// Read the click before laying out: Clickable.Layout drains pending
			// click events, so a post-layout Clicked would always be false.
			if item.Click != nil && item.Click.Clicked(gtx) {
				selected = i
			}
			dims := material.Clickable(gtx, item.Click, func(gtx layout.Context) layout.Dimensions {
				return layout.Stack{}.Layout(gtx,
					layout.Expanded(func(gtx layout.Context) layout.Dimensions {
						if bg.A == 0 {
							return layout.Dimensions{Size: gtx.Constraints.Min}
						}
						radius := gtx.Dp(theme.Corner)
						rr := clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, radius)
						defer rr.Push(gtx.Ops).Pop()
						paint.Fill(gtx.Ops, bg)
						return layout.Dimensions{Size: gtx.Constraints.Min}
					}),
					layout.Stacked(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Body2(th, item.Label)
						lbl.Color = fg
						return layout.Inset{Top: 10, Bottom: 10, Left: 14, Right: 14}.Layout(gtx, lbl.Layout)
					}),
				)
			})
			return dims
		}))
	}
	dims := layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	return dims, selected
}
