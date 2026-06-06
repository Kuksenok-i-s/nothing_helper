//go:build gio

package view

import (
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"tws_manager/internal/ui/gio/state"
	"tws_manager/internal/ui/gio/theme"
	"tws_manager/internal/ui/gio/widgets"
)

// Layout renders the full application shell.
func Layout(gtx layout.Context, th *material.Theme, s *state.State) layout.Dimensions {
	for {
		ev, ok := gtx.Event(pointer.Filter{Kinds: pointer.Press})
		if !ok {
			break
		}
		if pe, ok := ev.(pointer.Event); ok && pe.Kind == pointer.Press {
			s.MarkUserInteraction()
		}
	}

	widgets.FillBackground(gtx, theme.Bg)
	snap := s.Snapshot()

	return widgets.InsetPanel(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Max.X = gtx.Dp(theme.SidebarW)
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				return Sidebar(gtx, th, s, snap)
			}),
			layout.Rigid(layout.Spacer{Width: theme.GapLG}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return Header(gtx, th, snap)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if snap.SudoPrompt == "" {
							return layout.Dimensions{}
						}
						return sudoPasswordPrompt(gtx, th, s, snap.SudoPrompt)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if !snap.DualPromptShown || snap.DualPrompt == "" {
							return layout.Dimensions{}
						}
						return dualSwitchPrompt(gtx, th, s, snap.DualPrompt)
					}),
					layout.Rigid(layout.Spacer{Height: theme.GapMD}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return Content(gtx, th, s, snap)
					}),
				)
			}),
		)
	})
}

func sudoPasswordPrompt(gtx layout.Context, th *material.Theme, s *state.State, prompt string) layout.Dimensions {
	for {
		ev, ok := s.SudoPassword.Update(gtx)
		if !ok {
			break
		}
		if _, ok := ev.(widget.SubmitEvent); ok {
			s.SubmitSudoPassword()
		}
	}
	return layout.Inset{Top: theme.GapMD}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return widgets.Card(gtx, theme.SurfaceAlt, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: theme.GapSM, Bottom: theme.GapSM, Left: theme.GapSM, Right: theme.GapSM}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				title := material.Body1(th, "Sudo password required")
				title.Color = theme.Fg
				body := material.Body2(th, prompt)
				body.Color = theme.FgMuted
				editor := material.Editor(th, &s.SudoPassword, "password")
				editor.Color = theme.Fg
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(title.Layout),
					layout.Rigid(layout.Spacer{Height: theme.GapXS}.Layout),
					layout.Rigid(body.Layout),
					layout.Rigid(layout.Spacer{Height: theme.GapSM}.Layout),
					layout.Rigid(editor.Layout),
					layout.Rigid(layout.Spacer{Height: theme.GapSM}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return widgets.Row(gtx, th,
							func(gtx layout.Context) layout.Dimensions {
								dims, clicked := widgets.Button(gtx, th, &s.SudoSubmit, "Unlock", widgets.VariantPrimary)
								if clicked {
									s.SubmitSudoPassword()
								}
								return dims
							},
							func(gtx layout.Context) layout.Dimensions {
								dims, clicked := widgets.Button(gtx, th, &s.SudoCancel, "Cancel", widgets.VariantGhost)
								if clicked {
									s.CancelSudoPassword()
								}
								return dims
							},
						)
					}),
				)
			})
		})
	})
}

func dualSwitchPrompt(gtx layout.Context, th *material.Theme, s *state.State, prompt string) layout.Dimensions {
	return layout.Inset{Top: theme.GapMD}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return widgets.Card(gtx, theme.SurfaceAlt, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: theme.GapSM, Bottom: theme.GapSM, Left: theme.GapSM, Right: theme.GapSM}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				title := material.Body1(th, "Dual connection")
				title.Color = theme.Fg
				body := material.Body2(th, prompt)
				body.Color = theme.FgMuted
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(title.Layout),
					layout.Rigid(layout.Spacer{Height: theme.GapXS}.Layout),
					layout.Rigid(body.Layout),
					layout.Rigid(layout.Spacer{Height: theme.GapSM}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return widgets.Row(gtx, th,
							func(gtx layout.Context) layout.Dimensions {
								dims, clicked := widgets.Button(gtx, th, &s.DualAcceptBtn, "Switch to PC", widgets.VariantPrimary)
								if clicked {
									s.AcceptDualPCPrimary()
								}
								return dims
							},
							func(gtx layout.Context) layout.Dimensions {
								dims, clicked := widgets.Button(gtx, th, &s.DualDeclineBtn, "Not now", widgets.VariantGhost)
								if clicked {
									s.DeclineDualPCPrimary()
								}
								return dims
							},
						)
					}),
				)
			})
		})
	})
}

func Content(gtx layout.Context, th *material.Theme, s *state.State, snap state.Snapshot) layout.Dimensions {
	return widgets.Card(gtx, theme.Surface, func(gtx layout.Context) layout.Dimensions {
		switch snap.Tab {
		case state.TabLog:
			return Log(gtx, th, s, snap)
		default:
			return Control(gtx, th, s, snap)
		}
	})
}
