package app

import (
	"context"
	"errors"
	"testing"
)

func TestRuntimeShutdownIdempotent(t *testing.T) {
	rt, err := Bootstrap(context.Background(), Config{
		RFCOMMDevice: "/dev/rfcomm0",
		Channel:      15,
		CaptureDir:   t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := rt.Shutdown(context.Background()); err != nil {
		t.Fatalf("first shutdown: %v", err)
	}
	if err := rt.Shutdown(context.Background()); err != nil {
		t.Fatalf("second shutdown: %v", err)
	}
}

func TestRuntimeShutdownCancelledContext(t *testing.T) {
	rt, err := Bootstrap(context.Background(), Config{
		RFCOMMDevice: "/dev/rfcomm0",
		Channel:      15,
		CaptureDir:   t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := rt.Shutdown(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Shutdown() = %v", err)
	}
}

func TestRuntimeShutdownNil(t *testing.T) {
	var rt *Runtime
	if err := rt.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown(nil) = %v", err)
	}
}
