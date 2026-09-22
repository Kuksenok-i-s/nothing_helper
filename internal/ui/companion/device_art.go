//go:build gio

package companion

import (
	"fmt"
	"image"
	"time"

	"gioui.org/io/semantic"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"nothing_helper/internal/spp"
)

func (v *view) deviceArt(gtx layout.Context, c *Controller, s Snapshot) layout.Dimensions {
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		charge, hasCase := CaseCharge(s.Session)
		height := unit.Dp(145)
		if hasCase {
			height = 175
		}
		size := image.Pt(gtx.Dp(232), gtx.Dp(height))
		paint.FillShape(gtx.Ops, v.p.line, clip.UniformRRect(image.Rectangle{Max: size}, gtx.Dp(32)).Op(gtx.Ops))
		paint.FillShape(gtx.Ops, v.p.surface, clip.UniformRRect(image.Rectangle{Min: image.Pt(1, 1), Max: size.Sub(image.Pt(1, 1))}, gtx.Dp(32)).Op(gtx.Ops))
		for i, side := range []string{"left", "right"} {
			x := unit.Dp(39 + i*96)
			at := func(y unit.Dp, w, h unit.Dp, draw layout.Widget) {
				off := op.Offset(image.Pt(gtx.Dp(x), gtx.Dp(y))).Push(gtx.Ops)
				sub := gtx
				sub.Constraints = layout.Exact(image.Pt(gtx.Dp(w), gtx.Dp(h)))
				draw(sub)
				off.Pop()
			}
			at(12, 58, 18, func(gtx layout.Context) layout.Dimensions {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					label := "L"
					if side == "right" {
						label = "R"
					}
					return v.label(gtx, 11, label, true)
				})
			})
			state, known := WearKnown(s.Session, side, time.Now())
			if known {
				// Expire the wear indicator even when no session event arrives.
				gtx.Execute(op.InvalidateCmd{At: state.UpdatedAt.Add(30*time.Second + time.Nanosecond)})
			}
			at(35, 58, 76, func(gtx layout.Context) layout.Dimensions {
				click := v.click("find-" + side)
				active := s.FindSide == side
				enabled := active || s.Session.Connected && spp.SupportsFind(s.Session.Model) && !s.Busy && !(known && state.InEar)
				if !enabled {
					gtx = gtx.Disabled()
				}
				for click.Clicked(gtx) {
					if active {
						c.StopFind()
					} else {
						c.Find(side)
					}
				}
				return click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					name := v.tr("Найти левый наушник")
					if side == "right" {
						name = v.tr("Найти правый наушник")
					}
					if active {
						name = v.tr("Остановить поиск")
					}
					semantic.LabelOp(name).Add(gtx.Ops)
					if active {
						paint.FillShape(gtx.Ops, v.p.accent, clip.UniformRRect(image.Rectangle{Min: image.Pt(-2, -2), Max: gtx.Constraints.Min.Add(image.Pt(2, 2))}, gtx.Dp(29)).Op(gtx.Ops))
					}
					paint.FillShape(gtx.Ops, v.p.raised, clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, gtx.Dp(27)).Op(gtx.Ops))
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						text := "—"
						b, ok := s.Session.Batteries[side]
						if ok && s.Session.Connected && b.Percent >= 0 && b.Percent <= 100 {
							text = fmt.Sprintf("%d%%", b.Percent)
						}
						return v.heading(gtx, 18, text)
					})
				})
			})
			at(118, 58, 10, func(gtx layout.Context) layout.Dimensions {
				color := v.p.muted
				label := v.tr("Положение неизвестно")
				if known {
					color = hex(0xed4945)
					label = v.tr("Вне уха")
					if state.InEar {
						color = hex(0x44cd69)
						label = v.tr("В ухе")
					}
				}
				semantic.DescriptionOp(label).Add(gtx.Ops)
				dot := image.Rect(gtx.Dp(25), 0, gtx.Dp(33), gtx.Dp(8))
				paint.FillShape(gtx.Ops, color, clip.Ellipse(dot).Op(gtx.Ops))
				return layout.Dimensions{Size: gtx.Constraints.Min}
			})
		}
		if hasCase {
			off := op.Offset(image.Pt(0, gtx.Dp(140))).Push(gtx.Ops)
			sub := gtx
			sub.Constraints = layout.Exact(image.Pt(size.X, gtx.Dp(22)))
			layout.Center.Layout(sub, func(gtx layout.Context) layout.Dimensions {
				return v.label(gtx, 14, fmt.Sprintf(v.tr("Кейс %d%%"), charge), false)
			})
			off.Pop()
		}
		return layout.Dimensions{Size: size}
	})
}
