//go:build gio

package companion

import (
	"image"
	"image/color"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type palette struct{ bg, surface, raised, fg, muted, line, accent, onAccent color.NRGBA }

func colors(light bool) palette {
	if light {
		return palette{hex(0xf5f5f3), hex(0xffffff), hex(0xe9e9e6), hex(0x222222), hex(0x656563), hex(0xdcdcd8), hex(0xbc2929), hex(0xffffff)}
	}
	return palette{hex(0x151515), hex(0x202020), hex(0x2d2d2d), hex(0xf2f2ee), hex(0xaaaaa5), hex(0x353535), hex(0xc83a36), hex(0xffffff)}
}
func hex(v uint32) color.NRGBA {
	return color.NRGBA{R: byte(v >> 16), G: byte(v >> 8), B: byte(v), A: 255}
}
func (v *view) label(gtx layout.Context, size unit.Sp, text string, muted bool) layout.Dimensions {
	l := material.Label(v.th, size, text)
	l.Color = v.p.fg
	if muted {
		l.Color = v.p.muted
	}
	return l.Layout(gtx)
}
func (v *view) heading(gtx layout.Context, size unit.Sp, text string) layout.Dimensions {
	l := material.Label(v.th, size, text)
	l.Color = v.p.fg
	l.Font.Weight = font.Medium
	return l.Layout(gtx)
}
func space(h unit.Dp) layout.FlexChild { return layout.Rigid(layout.Spacer{Height: h}.Layout) }
func surface(gtx layout.Context, bg color.NRGBA, radius unit.Dp, inset unit.Dp, w layout.Widget) layout.Dimensions {
	return layout.Stack{}.Layout(gtx, layout.Expanded(func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, bg, clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, gtx.Dp(radius)).Op(gtx.Ops))
		return layout.Dimensions{Size: gtx.Constraints.Min}
	}), layout.Stacked(func(gtx layout.Context) layout.Dimensions { return layout.UniformInset(inset).Layout(gtx, w) }))
}
func (v *view) button(gtx layout.Context, id, label string, selected, enabled bool, fn func()) layout.Dimensions {
	click := v.click(id)
	if !enabled {
		gtx = gtx.Disabled()
	}
	for click.Clicked(gtx) {
		if fn != nil {
			fn()
		}
	}
	b := material.Button(v.th, click, label)
	b.TextSize = 12
	b.CornerRadius = 9
	b.Inset = layout.Inset{Top: 10, Bottom: 10, Left: 11, Right: 11}
	b.Background = v.p.raised
	b.Color = v.p.fg
	if selected {
		b.Background = v.p.accent
		b.Color = v.p.onAccent
	}
	return b.Layout(gtx)
}
func (v *view) click(id string) *widget.Clickable {
	if v.clicks[id] == nil {
		v.clicks[id] = new(widget.Clickable)
	}
	return v.clicks[id]
}
func (v *view) line(gtx layout.Context) layout.Dimensions {
	size := image.Pt(gtx.Constraints.Max.X, gtx.Dp(1))
	paint.FillShape(gtx.Ops, v.p.line, clip.Rect{Max: size}.Op())
	return layout.Dimensions{Size: size}
}
func (v *view) row(gtx layout.Context, label, sub string, right layout.Widget) layout.Dimensions {
	return layout.Inset{Top: 12, Bottom: 12}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, layout.Rigid(func(gtx layout.Context) layout.Dimensions { return v.label(gtx, 14, label, false) }), space(3), layout.Rigid(func(gtx layout.Context) layout.Dimensions { return v.label(gtx, 11, sub, true) }))
		}), layout.Rigid(layout.Spacer{Width: 12}.Layout), layout.Rigid(right))
	})
}
