package session

import (
	"os"
	"testing"
	"time"

	"tws_manager/internal/bt"
	"tws_manager/internal/spp"
)

func TestInitialProbeSendsHandshake(t *testing.T) {
	spp.ResetFSN()
	oldSleep := probeSleep
	probeSleep = func(time.Duration) {}
	t.Cleanup(func() { probeSleep = oldSleep })

	s := New(nil, false, false)
	f, err := os.CreateTemp(t.TempDir(), "rfcomm")
	if err != nil {
		t.Fatal(err)
	}
	dev := bt.Device{MAC: "AA:BB:CC:DD:EE:FF", Name: "Ear"}
	s.AttachTestLink(bt.NewTestTransport(f, dev.MAC, 15, "/dev/rfcomm0"), dev)

	s.initialProbe(dev)

	s.mu.Lock()
	pending := len(s.pending)
	s.mu.Unlock()
	if pending == 0 {
		t.Fatal("expected pending commands from initial probe")
	}
}

func TestProbeConfigSkipsUnsupportedFeatures(t *testing.T) {
	oldSleep := probeSleep
	probeSleep = func(time.Duration) {}
	t.Cleanup(func() { probeSleep = oldSleep })

	s := New(nil, false, false)
	f, err := os.CreateTemp(t.TempDir(), "rfcomm")
	if err != nil {
		t.Fatal(err)
	}
	dev := bt.Device{MAC: "AA:BB:CC:DD:EE:FF", Name: "Ear"}
	s.AttachTestLink(bt.NewTestTransport(f, dev.MAC, 15, "/dev/rfcomm0"), dev)
	s.mu.Lock()
	s.model = spp.ModelInfo{Codename: "Minimal", Features: []string{"anc"}}
	s.mu.Unlock()

	s.probeConfig(dev)
}

func TestProbeConfigExitsWhenDisconnected(t *testing.T) {
	oldSleep := probeSleep
	probeSleep = func(time.Duration) {}
	t.Cleanup(func() { probeSleep = oldSleep })

	s := New(nil, false, false)
	dev := bt.Device{MAC: "AA:BB:CC:DD:EE:FF", Name: "Ear"}
	s.mu.Lock()
	s.device = dev
	s.transport = nil
	s.mu.Unlock()

	s.probeConfig(dev)
}
