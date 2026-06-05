//go:build gio

package widgets

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"tws_manager/internal/ui/gio/theme"
)

// ListItem is a selectable row in a list.
type ListItem struct {
	Title    string
	Subtitle string
	Selected bool
	Click    *widget.Clickable
}

// SelectList renders selectable list rows.
func SelectList(gtx layout.Context, th *material.Theme, items []ListItem, onSelect func(int)) layout.Dimensions {
	if len(items) == 0 {
		lbl := material.Body2(th, "No devices found — run Discover.")
		lbl.Color = theme.FgMuted
		return lbl.Layout(gtx)
	}
	children := make([]layout.FlexChild, 0, len(items))
	for i, item := range items {
		i, item := i, item
		if item.Click == nil {
			continue
		}
		btn := item.Click
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			bg := theme.SurfaceAlt
			if item.Selected {
				bg = theme.AccentSoft
			}
			// Read the click before laying out: Clickable.Layout drains pending
			// click events, so a post-layout Clicked would always be false.
			if btn.Clicked(gtx) {
				onSelect(i)
			}
			dims := material.Clickable(gtx, btn, func(gtx layout.Context) layout.Dimensions {
				return layout.Stack{}.Layout(gtx,
					layout.Expanded(func(gtx layout.Context) layout.Dimensions {
						radius := gtx.Dp(theme.Corner)
						rr := clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, radius)
						defer rr.Push(gtx.Ops).Pop()
						paint.Fill(gtx.Ops, bg)
						return layout.Dimensions{Size: gtx.Constraints.Min}
					}),
					layout.Stacked(func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Top: 10, Bottom: 10, Left: 12, Right: 12}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							title := material.Body1(th, item.Title)
							title.Color = theme.Fg
							sub := material.Caption(th, item.Subtitle)
							sub.Color = theme.FgMuted
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(title.Layout),
								layout.Rigid(sub.Layout),
							)
						})
					}),
				)
			})
			return dims
		}))
		if i < len(items)-1 {
			children = append(children, layout.Rigid(layout.Spacer{Height: theme.GapSM}.Layout))
		}
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

// CommandList renders command buttons in a grid-like vertical list.
func CommandList(gtx layout.Context, th *material.Theme, labels []string, clicks []widget.Clickable, onClick func(int)) layout.Dimensions {
	children := make([]layout.FlexChild, 0, len(labels))
	for i, label := range labels {
		i, label := i, label
		btn := &clicks[i]
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			dims, clicked := Button(gtx, th, btn, label, VariantSecondary)
			if clicked {
				onClick(i)
			}
			return dims
		}))
		if i < len(labels)-1 {
			children = append(children, layout.Rigid(layout.Spacer{Height: theme.GapSM}.Layout))
		}
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}
