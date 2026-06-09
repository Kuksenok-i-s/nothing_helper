package actions

import (
	"context"
	"fmt"
	"strings"

	"tws_manager/internal/session"
	"tws_manager/internal/spp"
	"tws_manager/internal/trace"
	"tws_manager/internal/ui/presenter"
)

// ExecOpts configures command execution from UI layers.
type ExecOpts struct {
	Comment string
	Source  string
}

// ExecResult describes how Execute handled a command.
type ExecResult struct {
	Warnings []string
	Err      error
}

// Execute sends a presenter command through the session.
//
// All catalog commands are safe by construction (see presenter.BuildCommands);
// unsafe enforcement happens in Session.authorizeCommand / FeatureCommandPacket,
// so no UI-level gating is needed here.
func Execute(sess *session.Session, cmd presenter.Command, opts ExecOpts) ExecResult {
	if presenter.IsScanCommand(cmd) {
		return ExecResult{Err: fmt.Errorf("scan commands must use ExecuteScan")}
	}
	source := opts.Source
	if source == "" {
		source = "ui"
	}
	if len(cmd.Fields) > 0 {
		pkt, warnings, err := sess.FeaturePacket(cmd.Fields)
		if err != nil {
			return ExecResult{Err: err}
		}
		comment := opts.Comment
		if len(warnings) > 0 {
			comment = strings.TrimSpace(comment + " " + strings.Join(warnings, "; "))
		}
		trigger := cmd.Title
		if trigger == "" {
			trigger = strings.Join(cmd.Fields, " ")
		}
		if err := sess.Send(pkt, session.Meta{Source: source, Trigger: trigger, UserComment: comment}); err != nil {
			return ExecResult{Err: err}
		}
		return ExecResult{Warnings: warnings}
	}
	if err := sess.SendCommand(cmd.Cmd, session.Meta{Source: source, Trigger: cmd.Title, UserComment: opts.Comment}); err != nil {
		return ExecResult{Err: err}
	}
	return ExecResult{}
}

// ExecuteScan runs a validated GET scan range.
func ExecuteScan(ctx context.Context, sess *session.Session, fields []string) error {
	start, end, delay, err := spp.ParseScanCommand(fields)
	if err != nil {
		return err
	}
	return sess.RunQueryScan(ctx, start, end, delay)
}

// ExportPackets writes trace events to JSON.
func ExportPackets(path string, events []trace.Event, comment string, includeRaw bool) error {
	if len(events) == 0 {
		return fmt.Errorf("no packets to export")
	}
	return trace.Export(path, events, comment, includeRaw)
}
