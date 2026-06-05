//go:build gio

package view

import (
	"gioui.org/layout"
	"gioui.org/widget/material"

	"tws_manager/internal/ui/gio/state"
	"tws_manager/internal/ui/gio/theme"
	"tws_manager/internal/ui/gio/widgets"
)

// Sidebar renders navigation and quick disconnect.
func Sidebar(gtx layout.Context, th *material.Theme, s *state.State, snap state.Snapshot) layout.Dimensions {
	tabs := []state.Tab{state.TabControl, state.TabLog}
	nav := []widgets.NavItem{
		{Label: "Control", Active: snap.Tab == state.TabControl, Click: &s.TabControl},
		{Label: "Log", Active: snap.Tab == state.TabLog, Click: &s.TabLog},
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			logo := material.H6(th, "tws_manager")
			logo.Color = theme.Accent
			return logo.Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: theme.GapXL}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			dims, idx := widgets.NavList(gtx, th, nav)
			if idx >= 0 {
				s.SetTab(tabs[idx])
			}
			return dims
		}),
		layout.Flexed(1, layout.Spacer{}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			dims, clicked := widgets.Button(gtx, th, &s.DisconnectBtn, "Disconnect", widgets.VariantDanger)
			if clicked {
				s.Handle(state.ActionDisconnect)
			}
			return dims
		}),
	)
}
