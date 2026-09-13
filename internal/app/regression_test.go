package app

import (
	"context"
	"io"
	"testing"
	"time"

	"tws_manager/internal/bt"
)

type shutdownTestLink struct{}

func (shutdownTestLink) Read([]byte) (int, error)    { return 0, io.EOF }
func (shutdownTestLink) Write(p []byte) (int, error) { return len(p), nil }
func (shutdownTestLink) Close() error                { return nil }

func TestLongRunShutsDown(t *testing.T) {
	var runtime *Runtime
	err := Run(context.Background(), Config{CaptureDir: t.TempDir()}, func(ctx context.Context, rt *Runtime) error {
		runtime = rt
		rt.Session.AttachTestLink(bt.NewTestTransport(shutdownTestLink{}, "AA:BB:CC:DD:EE:FF", 15, "test"), bt.Device{MAC: "AA:BB:CC:DD:EE:FF"})
		time.Sleep(2100 * time.Millisecond)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Session.Close()
	if runtime.Logger != nil {
		defer runtime.Logger.Close()
	}
	if runtime.Session.Snapshot().Connected {
		t.Fatal("Run returned after 2 seconds without closing active transport")
	}
}
