package app

import (
	"context"
	"errors"
	"testing"
)

func TestRunBootstrapAndShutdown(t *testing.T) {
	cfg := Config{
		RFCOMMDevice: "/dev/rfcomm0",
		Channel:      15,
		CaptureDir:   t.TempDir(),
	}
	err := Run(context.Background(), cfg, func(ctx context.Context, rt *Runtime) error {
		if rt.Session == nil {
			t.Fatal("expected session")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Run() = %v", err)
	}
}

func TestRunPropagatesError(t *testing.T) {
	want := errors.New("run failed")
	cfg := Config{
		RFCOMMDevice: "/dev/rfcomm0",
		Channel:      15,
		CaptureDir:   t.TempDir(),
	}
	err := Run(context.Background(), cfg, func(context.Context, *Runtime) error {
		return want
	})
	if !errors.Is(err, want) {
		t.Fatalf("Run() = %v", err)
	}
}

func TestStartAutoConnectSkip(t *testing.T) {
	rt := testRuntime(true)
	StartAutoConnect(context.Background(), nil, rt.Config, func(string) {
		t.Fatal("status should not be called")
	})
}
