//go:build gio

package view

import (
	"fmt"
	"image"
	"image/color"
	"io"
	"strings"

	"gioui.org/io/clipboard"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"tws_manager/internal/trace"
	"tws_manager/internal/ui/gio/state"
	"tws_manager/internal/ui/gio/theme"
	"tws_manager/internal/ui/gio/widgets"
	"tws_manager/internal/ui/presenter"
)

// Log renders the read commands, a structured three-column packet view, and an
// optional raw-hex panel.
func Log(gtx layout.Context, th *material.Theme, s *state.State, snap state.Snapshot) layout.Dimensions {
	var getIdx []int
	for i, c := range snap.Commands {
		if presenter.IsGetCommand(c) {
			getIdx = append(getIdx, i)
		}
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return logActions(gtx, th, s, snap.LogText)
		}),
		layout.Rigid(layout.Spacer{Height: theme.GapMD}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return commandColumn(gtx, th, s, snap, "Read", getIdx, &s.GetList, widgets.VariantSecondary)
				}),
				layout.Rigid(layout.Spacer{Width: theme.GapMD}.Layout),
				layout.Flexed(3, func(gtx layout.Context) layout.Dimensions {
					return logColumns(gtx, th, s, snap)
				}),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if !s.RawHexToggle.Value {
				return layout.Dimensions{}
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(layout.Spacer{Height: theme.GapMD}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return rawHexPanel(gtx, th, s, snap)
				}),
			)
		}),
	)
}

func logActions(gtx layout.Context, th *material.Theme, s *state.State, logText string) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			dims, clicked := widgets.Button(gtx, th, &s.ExportBtn, "Export packets", widgets.VariantSecondary)
			if clicked {
				s.Handle(state.ActionExport)
			}
			return dims
		}),
		layout.Rigid(layout.Spacer{Width: theme.GapSM}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			dims, clicked := widgets.Button(gtx, th, &s.CopyBtn, "Copy to clipboard", widgets.VariantGhost)
			if clicked {
				if strings.TrimSpace(logText) == "" {
					s.SetStatus("nothing to copy")
				} else {
					gtx.Execute(clipboard.WriteCmd{
						Type: "application/text",
						Data: io.NopCloser(strings.NewReader(logText)),
					})
					s.SetStatus("log copied to clipboard")
				}
			}
			return dims
		}),
		layout.Rigid(layout.Spacer{Width: theme.GapLG}.Layout),
		layout.Rigid(material.CheckBox(th, &s.RawHexToggle, "show raw packets (TX/RX hex)").Layout),
	)
}

// logColumns renders three columns: human-readable last packets, the export
// buffer (all captured events), and the raw bytes of the last packets.
func logColumns(gtx layout.Context, th *material.Theme, s *state.State, snap state.Snapshot) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return logCard(gtx, th, "Last packets", func(gtx layout.Context) layout.Dimensions {
				return recentList(gtx, th, &s.HumanList, snap.Recent, humanLine, theme.Fg)
			})
		}),
		layout.Rigid(layout.Spacer{Width: theme.GapMD}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return logCard(gtx, th, "Export packets", func(gtx layout.Context) layout.Dimensions {
				return exportBuffer(gtx, th, &s.ExportList, snap.LogText)
			})
		}),
		layout.Rigid(layout.Spacer{Width: theme.GapMD}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return logCard(gtx, th, "raw", func(gtx layout.Context) layout.Dimensions {
				return recentList(gtx, th, &s.RawList, snap.Recent, rawLineFor(s.RawHexToggle.Value), theme.FgMuted)
			})
		}),
	)
}

// logCard wraps a titled scrollable body in a surface card.
func logCard(gtx layout.Context, th *material.Theme, title string, body layout.Widget) layout.Dimensions {
	return widgets.Card(gtx, theme.SurfaceAlt, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				h := material.Body1(th, title)
				h.Color = theme.Fg
				return layout.Inset{Bottom: theme.GapSM}.Layout(gtx, h.Layout)
			}),
			layout.Flexed(1, body),
		)
	})
}

func recentList(gtx layout.Context, th *material.Theme, list *widget.List, events []trace.Event, format func(trace.Event) string, c color.NRGBA) layout.Dimensions {
	if len(events) == 0 {
		lbl := material.Body2(th, "-")
		lbl.Color = theme.FgMuted
		return lbl.Layout(gtx)
	}
	list.Axis = layout.Vertical
	return material.List(th, list).Layout(gtx, len(events), func(gtx layout.Context, i int) layout.Dimensions {
		body := material.Body2(th, format(events[i]))
		body.Color = c
		return layout.Inset{Bottom: theme.GapSM}.Layout(gtx, body.Layout)
	})
}

func exportBuffer(gtx layout.Context, th *material.Theme, list *widget.List, text string) layout.Dimensions {
	if strings.TrimSpace(text) == "" {
		lbl := material.Body2(th, "No events yet.")
		lbl.Color = theme.FgMuted
		return lbl.Layout(gtx)
	}
	lines := strings.Split(text, "\n")
	list.Axis = layout.Vertical
	return material.List(th, list).Layout(gtx, len(lines), func(gtx layout.Context, i int) layout.Dimensions {
		body := material.Body2(th, lines[i])
		body.Color = theme.FgMuted
		return body.Layout(gtx)
	})
}

func rawHexPanel(gtx layout.Context, th *material.Theme, s *state.State, snap state.Snapshot) layout.Dimensions {
	return widgets.Card(gtx, theme.SurfaceAlt, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				h := material.Body1(th, "raw_hex")
				h.Color = theme.Fg
				return layout.Inset{Bottom: theme.GapSM}.Layout(gtx, h.Layout)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				rows := rawPairRows(th, snap.RawPackets)
				if len(rows) == 0 {
					lbl := material.Body2(th, "-")
					lbl.Color = theme.FgMuted
					return lbl.Layout(gtx)
				}
				s.RawHexList.Axis = layout.Vertical
				return material.List(th, &s.RawHexList).Layout(gtx, len(rows), func(gtx layout.Context, i int) layout.Dimensions {
					return rows[i](gtx)
				})
			}),
		)
	})
}

// rawPairRows renders raw packets as RAW_TX / RAW_RX entries, inserting a
// visual separator before each TX so request/response pairs are grouped.
func rawPairRows(th *material.Theme, events []trace.Event) []layout.Widget {
	var rows []layout.Widget
	for _, ev := range events {
		if ev.Direction == "tx" && len(rows) > 0 {
			rows = append(rows, rawPairSeparator())
		}
		rows = append(rows, rawPacketRow(th, ev))
	}
	return rows
}

func rawPairSeparator() layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: theme.GapSM, Bottom: theme.GapSM}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			line := image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Dp(unit.Dp(1)))
			paint.FillShape(gtx.Ops, theme.Border, clip.Rect(line).Op())
			return layout.Dimensions{Size: line.Max}
		})
	}
}

func rawPacketRow(th *material.Theme, ev trace.Event) layout.Widget {
	label := rawLabel(ev.Direction)
	labelColor := theme.Success // RX: incoming response
	if ev.Direction == "tx" {
		labelColor = theme.Accent // TX: outgoing request
	}
	cmd := ev.CommandName
	if cmd == "" {
		cmd = ev.Command
	}
	raw := ev.RawHex
	if raw == "" {
		raw = "-"
	}
	return func(gtx layout.Context) layout.Dimensions {
		head := material.Body2(th, fmt.Sprintf("%s  %s  %s", label, ev.Time, cmd))
		head.Color = labelColor
		body := material.Body2(th, raw)
		body.Color = theme.Fg
		return layout.Inset{Bottom: theme.GapXS}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(head.Layout),
				layout.Rigid(body.Layout),
			)
		})
	}
}

func rawLabel(direction string) string {
	if direction == "rx" {
		return "RAW_RX"
	}
	return "RAW_TX"
}

func humanLine(ev trace.Event) string {
	cmd := ev.CommandName
	if cmd == "" {
		cmd = ev.Command
	}
	line := fmt.Sprintf("%s %s %s", ev.Time, directionArrow(ev.Direction), cmd)
	if ev.Summary != "" {
		line += "\n   " + ev.Summary
	} else if ev.Error != "" {
		line += "\n   error: " + ev.Error
	}
	return line
}

func rawLineFor(logRaw bool) func(trace.Event) string {
	return func(ev trace.Event) string {
		if !logRaw {
			return fmt.Sprintf("%s %s (hidden)", rawLabel(ev.Direction), ev.Time)
		}
		raw := ev.RawHex
		if raw == "" {
			raw = "-"
		}
		return fmt.Sprintf("%s %s\n%s", rawLabel(ev.Direction), ev.Time, raw)
	}
}

func directionArrow(direction string) string {
	if direction == "rx" {
		return "←"
	}
	return "→"
}
