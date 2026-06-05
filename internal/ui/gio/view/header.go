//go:build gio

package view

import (
	"fmt"

	"gioui.org/layout"
	"gioui.org/widget/material"

	"tws_manager/internal/ui/gio/state"
	"tws_manager/internal/ui/gio/theme"
	"tws_manager/internal/ui/gio/widgets"
	"tws_manager/internal/ui/presenter"
)

// Header renders status cards and connection info.
func Header(gtx layout.Context, th *material.Theme, snap state.Snapshot) layout.Dimensions {
	model := snap.Session.Model.Codename
	if model == "" {
		model = "unknown"
	}
	connected := "offline"
	connColor := theme.FgMuted
	if snap.Session.Connected {
		connected = "connected"
		connColor = theme.Success
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			title := material.H6(th, "tws_manager")
			title.Color = theme.Fg
			return title.Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: theme.GapSM}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return statCard(gtx, th, "Device", fmt.Sprintf("%s\n%s", snap.Session.Device.Name, snap.Session.Device.MAC))
				}),
				layout.Rigid(layout.Spacer{Width: theme.GapSM}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return statCard(gtx, th, "Model", model)
				}),
				layout.Rigid(layout.Spacer{Width: theme.GapSM}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return statCard(gtx, th, "Battery", presenter.FormatBatteries(snap.Session.Batteries))
				}),
			)
		}),
		layout.Rigid(layout.Spacer{Height: theme.GapSM}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			status := material.Body2(th, snap.Status)
			status.Color = theme.FgMuted
			badge := material.Caption(th, connected)
			badge.Color = connColor
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, status.Layout),
				layout.Rigid(badge.Layout),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if snap.ErrText == "" {
				return layout.Dimensions{}
			}
			err := material.Body2(th, snap.ErrText)
			err.Color = theme.Error
			return err.Layout(gtx)
		}),
	)
}

func statCard(gtx layout.Context, th *material.Theme, label, value string) layout.Dimensions {
	return widgets.Card(gtx, theme.SurfaceAlt, func(gtx layout.Context) layout.Dimensions {
		lbl := material.Caption(th, label)
		lbl.Color = theme.FgMuted
		val := material.Body2(th, value)
		val.Color = theme.Fg
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(lbl.Layout),
			layout.Rigid(layout.Spacer{Height: theme.GapXS}.Layout),
			layout.Rigid(val.Layout),
		)
	})
}
