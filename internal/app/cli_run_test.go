package app

import (
	"context"
	"errors"
	"testing"

	"tws_manager/internal/connect"
	"tws_manager/internal/dualpolicy"
	"tws_manager/internal/session"
	"tws_manager/internal/ui/tray"
	"tws_manager/internal/ui/tui"
)

func TestSetCLIHooks(t *testing.T) {
	called := false
	restore := SetCLIHooks(nil, func(context.Context, *session.Session, tui.Options) error {
		called = true
		return nil
	})
	t.Cleanup(restore)
	if err := runTUIHook(context.Background(), nil, tui.Options{}); err != nil {
		t.Fatalf("runTUIHook: %v", err)
	}
	if !called {
		t.Fatal("expected hook to run")
	}
}

func TestRunCLIInvokesTUIHook(t *testing.T) {
	oldTray := runTrayHook
	oldTUI := runTUIHook
	tuiCalled := false
	runTrayHook = func(context.Context, *session.Session, tray.Options) {}
	runTUIHook = func(context.Context, *session.Session, tui.Options) error {
		tuiCalled = true
		return nil
	}
	t.Cleanup(func() {
		runTrayHook = oldTray
		runTUIHook = oldTUI
	})

	rt := testRuntime(false)
	rt.Config.AutoDiscover = true
	services := &Services{
		Manager:       connect.New(rt.Session, connect.Options{RFCOMMPath: "/dev/rfcomm0", Channel: 15}),
		PCPrimaryMode: dualpolicy.ModeOff,
	}
	if err := RunCLI(context.Background(), rt, services); err != nil {
		t.Fatalf("RunCLI() = %v", err)
	}
	if !tuiCalled {
		t.Fatal("expected TUI hook to run")
	}
}

func TestRunCLIPropagatesTUIError(t *testing.T) {
	oldTray := runTrayHook
	oldTUI := runTUIHook
	want := errors.New("tui failed")
	runTrayHook = func(context.Context, *session.Session, tray.Options) {}
	runTUIHook = func(context.Context, *session.Session, tui.Options) error { return want }
	t.Cleanup(func() {
		runTrayHook = oldTray
		runTUIHook = oldTUI
	})

	rt := testRuntime(false)
	rt.Config.AutoDiscover = true
	services := &Services{
		Manager:       connect.New(rt.Session, connect.Options{RFCOMMPath: "/dev/rfcomm0", Channel: 15}),
		PCPrimaryMode: dualpolicy.ModeOff,
	}
	if err := RunCLI(context.Background(), rt, services); !errors.Is(err, want) {
		t.Fatalf("RunCLI() = %v, want %v", err, want)
	}
}
