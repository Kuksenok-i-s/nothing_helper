package main

import (
	"context"
	"testing"

	"tws_manager/internal/app"
	"tws_manager/internal/session"
	"tws_manager/internal/ui/tray"
	"tws_manager/internal/ui/tui"
)

func TestRunMainHelp(t *testing.T) {
	if code := runMain([]string{"-h"}); code != 0 {
		t.Fatalf("runMain(-h) = %d", code)
	}
}

func TestRunMainMissingArgs(t *testing.T) {
	if code := runMain([]string{"--device"}); code != 2 {
		t.Fatalf("runMain() = %d, want 2", code)
	}
}

func TestRunMainSuccess(t *testing.T) {
	restore := app.SetCLIHooks(
		func(context.Context, *session.Session, tray.Options) {},
		func(context.Context, *session.Session, tui.Options) error { return nil },
	)
	t.Cleanup(restore)

	// --auto without --addr skips interactive RFCOMM preflight (and its
	// background connect goroutine), which would race across tests.
	code := runMain([]string{
		"--device", "/dev/rfcomm0",
		"--channel", "15",
		"--capture-dir", t.TempDir(),
		"--privilege-helper", "none",
		"--auto",
	})
	if code != 0 {
		t.Fatalf("runMain() = %d", code)
	}
}

func TestRunFunction(t *testing.T) {
	restore := app.SetCLIHooks(nil, func(context.Context, *session.Session, tui.Options) error { return nil })
	t.Cleanup(restore)

	cfg := app.Config{
		RFCOMMDevice:  "/dev/rfcomm0",
		Channel:       15,
		CaptureDir:    t.TempDir(),
		PrivilegeMode: "none",
		AutoDiscover:  true,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := app.Run(ctx, cfg, run)
	if err != nil {
		t.Fatalf("Run() = %v", err)
	}
}
