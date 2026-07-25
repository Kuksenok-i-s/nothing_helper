package app

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestBootstrapDefaults(t *testing.T) {
	dir := t.TempDir()
	rt, err := Bootstrap(context.Background(), Config{
		RFCOMMDevice: "/dev/rfcomm0",
		Channel:      15,
		CaptureDir:   dir,
	})
	if err != nil {
		t.Fatalf("Bootstrap() = %v", err)
	}
	defer func() { _ = rt.Close() }()
	if rt.Session == nil || rt.Logger == nil {
		t.Fatal("expected session and logger")
	}
	if !strings.Contains(rt.TracePath, dir) {
		t.Fatalf("TracePath=%q", rt.TracePath)
	}
}

func TestBootstrapUnknownModel(t *testing.T) {
	_, err := Bootstrap(context.Background(), Config{
		RFCOMMDevice: "/dev/rfcomm0",
		Channel:      15,
		CaptureDir:   t.TempDir(),
		ModelName:    "NotARealModel",
	})
	if err == nil || !strings.Contains(err.Error(), "unknown model") {
		t.Fatalf("Bootstrap() = %v", err)
	}
}

func TestBootstrapKnownModelAndNotifyPolling(t *testing.T) {
	rt, err := Bootstrap(context.Background(), Config{
		RFCOMMDevice: "/dev/rfcomm0",
		Channel:      15,
		CaptureDir:   t.TempDir(),
		ModelName:    "EarThree",
		Notify:       true,
	})
	if err != nil {
		t.Fatalf("Bootstrap() = %v", err)
	}
	defer func() { _ = rt.Close() }()
	if rt.Session.Snapshot().Model.Codename == "" {
		t.Fatal("expected model to be set")
	}
}

func TestBootstrapExplicitTracePath(t *testing.T) {
	path := t.TempDir() + "/custom.ndjson"
	rt, err := Bootstrap(context.Background(), Config{
		RFCOMMDevice: "/dev/rfcomm0",
		Channel:      15,
		CaptureDir:   t.TempDir(),
		TracePath:    path,
		QueryEvery:   time.Second,
	})
	if err != nil {
		t.Fatalf("Bootstrap() = %v", err)
	}
	defer func() { _ = rt.Close() }()
	if rt.TracePath != path {
		t.Fatalf("TracePath=%q", rt.TracePath)
	}
}
