//go:build linux

package app

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"nothing_helper/internal/bt"
	"nothing_helper/internal/session"
	"nothing_helper/internal/spp"
)

func TestBootstrapRawCaptureRequiresLogRaw(t *testing.T) {
	for _, logRaw := range []bool{false, true} {
		t.Run(fmt.Sprintf("logRaw=%t", logRaw), func(t *testing.T) {
			dir := t.TempDir()
			bt.SetConfigPathHook(func() string { return filepath.Join(dir, "devices.json") })
			defer bt.SetConfigPathHook(nil)
			pr, pw, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			defer pr.Close()
			defer pw.Close()
			restore := bt.SetRFCOMMHooks(func(string, time.Duration) (*os.File, error) { return pr, nil }, nil)
			defer restore()
			rt, err := Bootstrap(context.Background(), Config{CaptureDir: dir, LogRaw: logRaw})
			if err != nil {
				t.Fatal(err)
			}
			defer rt.Close()
			if err := rt.Session.Connect(bt.Device{}, "/dev/rfcomm0", 15); err != nil {
				t.Fatal(err)
			}
			frame := spp.BuildFrame(spp.ControlTXDefault, spp.CmdRspBattery, 1, []byte{1, 1, 42})
			if _, err := pw.Write(frame); err != nil {
				t.Fatal(err)
			}
			deadline := time.After(time.Second)
		waitPacket:
			for {
				select {
				case event := <-rt.Session.Events():
					if event.Kind == session.EventBattery {
						break waitPacket
					}
				case <-deadline:
					t.Fatal("battery packet was not processed")
				}
			}
			files, err := filepath.Glob(filepath.Join(dir, "stream_*.bin"))
			if err != nil {
				t.Fatal(err)
			}
			if !logRaw {
				if len(files) != 0 {
					t.Fatalf("raw capture created without --log-raw: %v", files)
				}
				return
			}
			if len(files) != 1 {
				t.Fatalf("expected one raw capture, got %v", files)
			}
			data, err := os.ReadFile(files[0])
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(data, frame) {
				t.Fatalf("capture = %x, want %x", data, frame)
			}
		})
	}
}
