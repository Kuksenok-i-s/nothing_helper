//go:build gio

package view

import (
	"fmt"
	"image/color"
	"strings"

	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"tws_manager/internal/spp"
	"tws_manager/internal/ui/gio/state"
	"tws_manager/internal/ui/gio/theme"
	"tws_manager/internal/ui/gio/widgets"
	"tws_manager/internal/ui/presenter"
)

// Control renders the device bar, the command panel on the left, and a
// status/info panel on the right.
func Control(gtx layout.Context, th *material.Theme, s *state.State, snap state.Snapshot) layout.Dimensions {
	s.SyncCommands()

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return deviceBar(gtx, th, s, snap)
		}),
		layout.Rigid(layout.Spacer{Height: theme.GapMD}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return controlCommands(gtx, th, s, snap)
				}),
				layout.Rigid(layout.Spacer{Width: theme.GapLG}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Max.X = gtx.Dp(theme.StatusW)
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return controlStatus(gtx, th, s, snap)
				}),
			)
		}),
	)
}

// deviceBar shows auto-connect/discover actions plus a device picker that lets
// the user choose among 2+ discovered devices directly on the Control screen.
func deviceBar(gtx layout.Context, th *material.Theme, s *state.State, snap state.Snapshot) layout.Dimensions {
	children := []layout.FlexChild{
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			dims, c := widgets.Button(gtx, th, &s.AutoBtn, "Auto-connect", widgets.VariantPrimary)
			if c {
				s.Handle(state.ActionAuto)
			}
			return dims
		}),
		layout.Rigid(layout.Spacer{Width: theme.GapSM}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			dims, c := widgets.Button(gtx, th, &s.DiscoverBtn, "Discover", widgets.VariantSecondary)
			if c {
				s.Handle(state.ActionDiscover)
			}
			return dims
		}),
	}

	// Device picker only appears once there are at least two candidates.
	if len(snap.Devices) >= 2 {
		children = append(children, layout.Rigid(layout.Spacer{Width: theme.GapLG}.Layout))
		for i, dev := range snap.Devices {
			i, dev := i, dev
			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				name := dev.Name
				if name == "" {
					name = dev.MAC
				}
				variant := widgets.VariantSecondary
				if snap.Session.Connected && dev.MAC != "" &&
					strings.EqualFold(dev.MAC, snap.Session.Device.MAC) {
					variant = widgets.VariantPrimary
				}
				dims, c := widgets.Button(gtx, th, s.DeviceClick(i), name, variant)
				if c {
					s.ConnectChosen(i)
				}
				return dims
			}))
			children = append(children, layout.Rigid(layout.Spacer{Width: theme.GapSM}.Layout))
		}
	}

	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx, children...)
}

func controlCommands(gtx layout.Context, th *material.Theme, s *state.State, snap state.Snapshot) layout.Dimensions {
	var setIdx []int
	for i, c := range snap.Commands {
		switch {
		case presenter.IsGetCommand(c):
			// READ commands now live on the Log tab.
		case presenter.IsToggleSetCommand(c):
			// Rendered as a switch in the Write column, not a button.
		case presenter.IsSetCommand(c), c.Advanced:
			setIdx = append(setIdx, i)
		}
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			dims, clicked := widgets.Button(gtx, th, &s.RefreshBattery, "Refresh battery", widgets.VariantSecondary)
			if clicked {
				s.Handle(state.ActionBattery)
			}
			return dims
		}),
		layout.Rigid(layout.Spacer{Height: theme.GapMD}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return writeColumn(gtx, th, s, snap, setIdx)
		}),
	)
}

// writeColumn shows on/off feature switches (which double as indicators) above
// the scrollable list of remaining SET command buttons.
func writeColumn(gtx layout.Context, th *material.Theme, s *state.State, snap state.Snapshot, setIdx []int) layout.Dimensions {
	toggles := presenter.ToggleFeatures(snap.Commands)
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			h := material.Body1(th, "Write")
			h.Color = theme.Fg
			return layout.Inset{Bottom: theme.GapSM}.Layout(gtx, h.Layout)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if len(toggles) == 0 {
				return layout.Dimensions{}
			}
			children := make([]layout.FlexChild, 0, len(toggles)+1)
			for _, tf := range toggles {
				tf := tf
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return toggleRow(gtx, th, s, tf, snap.Session.Config[tf.Feature])
				}))
			}
			children = append(children, layout.Rigid(layout.Spacer{Height: theme.GapSM}.Layout))
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return commandListBody(gtx, th, s, snap, setIdx, &s.SetList, widgets.VariantPrimary)
		}),
	)
}

// toggleRow renders one on/off feature switch. The switch position reflects the
// device state (indicator) and flipping it sends the matching SET command.
func toggleRow(gtx layout.Context, th *material.Theme, s *state.State, tf presenter.ToggleFeature, configValue string) layout.Dimensions {
	b := s.ToggleBool(tf.Feature)
	deviceOn := presenter.ToggleStateOn(tf.Feature, configValue)
	b.Value = s.SyncToggle(tf.Feature, deviceOn)
	if b.Update(gtx) {
		s.RunToggle(tf, b.Value)
	}

	stateText := "off"
	stateColor := theme.FgMuted
	if b.Value {
		stateText = "on"
		stateColor = theme.Success
	}

	return layout.Inset{Top: theme.GapXS, Bottom: theme.GapXS}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Body2(th, tf.Label)
				lbl.Color = theme.Fg
				sub := material.Caption(th, stateText)
				sub.Color = stateColor
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(lbl.Layout),
					layout.Rigid(sub.Layout),
				)
			}),
			layout.Rigid(layout.Spacer{Width: theme.GapSM}.Layout),
			layout.Rigid(material.Switch(th, b, tf.Label).Layout),
		)
	})
}

// commandColumn renders one titled, scrollable column of command buttons.
// indices map list positions to entries in snap.Commands (and s.CmdButtons).
func commandColumn(gtx layout.Context, th *material.Theme, s *state.State, snap state.Snapshot, title string, indices []int, list *widget.List, variant widgets.Variant) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			h := material.Body1(th, title)
			h.Color = theme.Fg
			return layout.Inset{Bottom: theme.GapSM}.Layout(gtx, h.Layout)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return commandListBody(gtx, th, s, snap, indices, list, variant)
		}),
	)
}

// commandListBody renders just the scrollable list of command buttons (no header).
func commandListBody(gtx layout.Context, th *material.Theme, s *state.State, snap state.Snapshot, indices []int, list *widget.List, variant widgets.Variant) layout.Dimensions {
	if len(indices) == 0 {
		lbl := material.Body2(th, "—")
		lbl.Color = theme.FgMuted
		return lbl.Layout(gtx)
	}
	list.Axis = layout.Vertical
	return material.List(th, list).Layout(gtx, len(indices), func(gtx layout.Context, pos int) layout.Dimensions {
		idx := indices[pos]
		cmd := snap.Commands[idx]
		return layout.Inset{Bottom: theme.GapSM, Right: theme.GapSM}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			dims, clicked := widgets.Button(gtx, th, &s.CmdButtons[idx], cmd.Title, variant)
			if clicked {
				go s.RunCommand(cmd)
			}
			return dims
		})
	})
}

// controlStatus renders detailed device information, statuses, and a dual-device
// connect menu on the right. The whole panel scrolls to fit long content.
func controlStatus(gtx layout.Context, th *material.Theme, s *state.State, snap state.Snapshot) layout.Dimensions {
	rows := buildStatusRows(th, s, snap)
	return widgets.Card(gtx, theme.SurfaceAlt, func(gtx layout.Context) layout.Dimensions {
		s.StatusList.Axis = layout.Vertical
		return material.List(th, &s.StatusList).Layout(gtx, len(rows), func(gtx layout.Context, i int) layout.Dimensions {
			return rows[i](gtx)
		})
	})
}

func buildStatusRows(th *material.Theme, s *state.State, snap state.Snapshot) []layout.Widget {
	sess := snap.Session

	conn := "offline"
	connColor := theme.FgMuted
	if sess.Connected {
		conn = "connected"
		connColor = theme.Success
	}

	device := firstNonEmpty(sess.Device.Name, sess.Device.MAC, "—")
	model := firstNonEmpty(sess.Model.Product, sess.Model.Codename, "unknown")

	channel := "—"
	if sess.Device.Channel > 0 {
		channel = fmt.Sprintf("%d", sess.Device.Channel)
	}

	rows := []layout.Widget{dualHeader(th, s, len(sess.DualList))}
	if len(sess.DualList) == 0 {
		rows = append(rows, hintRow(th, "No paired devices. Tap Refresh to query."))
	} else {
		for i, dev := range sess.DualList {
			rows = append(rows, dualDeviceRow(th, s, i, dev))
		}
	}

	rows = append(rows, divider(), sectionHeader(th, "Status"))
	if !sess.Connected {
		rows = append(rows, hintRow(th, "Device disconnected — use Auto-connect above to reconnect."))
	}
	rows = append(rows,
		kvRowColor(th, "Connection", conn, connColor),
		kvRow(th, "Device", device),
		kvRow(th, "MAC", orDash(sess.Device.MAC)),
		kvRow(th, "RFCOMM channel", channel),
		kvRow(th, "Paired", yesNo(sess.Device.Paired)),
		kvRow(th, "SPP", yesNo(sess.Device.SPP)),

		divider(),
		sectionHeader(th, "Model"),
		kvRow(th, "Product", model),
		kvRow(th, "Codename", orDash(sess.Model.Codename)),
		kvRow(th, "Protocol", orDash(sess.Model.Protocol)),
		kvRow(th, "Tier", orDash(sess.Model.Tier)),
		kvRow(th, "Battery source", orDash(sess.Model.BatteryCaseSource)),
		kvRow(th, "Features", orDash(strings.Join(sess.Model.Features, ", "))),
	)

	rows = append(rows, divider(), sectionHeader(th, "Battery"))
	rows = append(rows, batteryRows(th, sess.Batteries)...)

	rows = append(rows, divider(), sectionHeader(th, "Configuration"))
	rows = append(rows, configRows(th, sess.Config)...)

	rows = append(rows,
		divider(),
		sectionHeader(th, "Activity"),
		kvRow(th, "Captured packets", fmt.Sprintf("%d", snap.Packets)),
		captionRow(th, "Last status"),
		valueRow(th, orDash(snap.Status), theme.Fg),
	)
	if snap.ErrText != "" {
		rows = append(rows, valueRow(th, snap.ErrText, theme.Error))
	}
	return rows
}

func batteryRows(th *material.Theme, data map[string]spp.Battery) []layout.Widget {
	order := []string{"left", "right", "case", "stereo"}
	rows := make([]layout.Widget, 0, len(order))
	any := false
	for _, name := range order {
		b, ok := data[name]
		if !ok {
			continue
		}
		any = true
		value := fmt.Sprintf("%d%%", b.Percent)
		color := theme.Fg
		if b.Charging {
			value += " (charging)"
			color = theme.Success
		}
		rows = append(rows, kvRowColor(th, capitalize(name), value, color))
	}
	if !any {
		rows = append(rows, hintRow(th, "No battery data yet."))
	}
	return rows
}

// configRows shows decoded device configuration (ANC, low latency, EQ, spatial)
// populated by the post-connect probe.
func configRows(th *material.Theme, config map[string]string) []layout.Widget {
	type item struct{ key, label string }
	items := []item{
		{"anc", "ANC"},
		{"lag", "Low latency"},
		{"eq", "EQ"},
		{"spatial", "Spatial audio"},
	}
	rows := make([]layout.Widget, 0, len(items))
	any := false
	for _, it := range items {
		if v, ok := config[it.key]; ok && strings.TrimSpace(v) != "" {
			any = true
			rows = append(rows, kvRow(th, it.label, v))
		} else {
			rows = append(rows, kvRow(th, it.label, "—"))
		}
	}
	if !any {
		rows = append(rows, hintRow(th, "Waiting for device config…"))
	}
	return rows
}

func dualDeviceRow(th *material.Theme, s *state.State, idx int, dev spp.DualDevice) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		click := s.DualClick(idx)

		label := "Connect"
		variant := widgets.VariantPrimary
		stateText := "disconnected"
		stateColor := theme.FgMuted
		if dev.Connected {
			label = "Disconnect"
			variant = widgets.VariantDanger
			stateText = "connected"
			stateColor = theme.Success
		}
		if dev.Owner {
			stateText += " · owner"
		}
		name := firstNonEmpty(dev.Name, dev.MAC, "—")

		return layout.Inset{Top: theme.GapXS, Bottom: theme.GapXS}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					n := material.Body2(th, name)
					n.Color = theme.Fg
					sub := material.Caption(th, dev.MAC+" · "+stateText)
					sub.Color = stateColor
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(n.Layout),
						layout.Rigid(sub.Layout),
					)
				}),
				layout.Rigid(layout.Spacer{Width: theme.GapSM}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					dims, clicked := widgets.Button(gtx, th, click, label, variant)
					if clicked {
						s.RunDualAction(dev)
					}
					return dims
				}),
			)
		})
	}
}

func dualHeader(th *material.Theme, s *state.State, count int) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				h := material.Body1(th, fmt.Sprintf("Connect devices (%d)", count))
				h.Color = theme.Fg
				return h.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				dims, clicked := widgets.Button(gtx, th, &s.DualRefreshBtn, "Refresh", widgets.VariantGhost)
				if clicked {
					s.RefreshDualList()
				}
				return dims
			}),
		)
	}
}

func sectionHeader(th *material.Theme, text string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		h := material.Body1(th, text)
		h.Color = theme.Fg
		return layout.Inset{Bottom: theme.GapXS}.Layout(gtx, h.Layout)
	}
}

func kvRow(th *material.Theme, label, value string) layout.Widget {
	return kvRowColor(th, label, value, theme.Fg)
}

func kvRowColor(th *material.Theme, label, value string, valueColor color.NRGBA) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: theme.GapXS, Bottom: theme.GapXS}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			lbl := material.Caption(th, label)
			lbl.Color = theme.FgMuted
			val := material.Body2(th, value)
			val.Color = valueColor
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Baseline}.Layout(gtx,
				layout.Rigid(lbl.Layout),
				layout.Flexed(1, layout.Spacer{Width: theme.GapSM}.Layout),
				layout.Rigid(val.Layout),
			)
		})
	}
}

func captionRow(th *material.Theme, text string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		lbl := material.Caption(th, text)
		lbl.Color = theme.FgMuted
		return lbl.Layout(gtx)
	}
}

func valueRow(th *material.Theme, text string, c color.NRGBA) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		val := material.Body2(th, text)
		val.Color = c
		return val.Layout(gtx)
	}
}

func hintRow(th *material.Theme, text string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		lbl := material.Body2(th, text)
		lbl.Color = theme.FgMuted
		return layout.Inset{Top: theme.GapXS, Bottom: theme.GapXS}.Layout(gtx, lbl.Layout)
	}
}

func divider() layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Spacer{Height: theme.GapMD}.Layout(gtx)
	}
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
