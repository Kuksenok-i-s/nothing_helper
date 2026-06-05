package app

import (
	"context"
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
