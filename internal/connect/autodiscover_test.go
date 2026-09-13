package connect

import (
	"context"
	"errors"
	"os"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"nothing_helper/internal/bt"
	"nothing_helper/internal/session"
)

func TestSleepCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if sleep(ctx, time.Second) {
		t.Fatal("sleep should return false when context is already cancelled")
	}

	ctx, cancel = context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if !sleep(ctx, time.Millisecond) {
		t.Fatal("sleep should complete before context timeout")
	}
}

func TestStatusReporterDedupes(t *testing.T) {
	var calls []string
	r := newStatusReporter(func(msg string) { calls = append(calls, msg) })
	r.report("a")
	r.report("a")
	r.report("b")
	r.reset()
	r.report("a")
	if len(calls) != 3 || calls[0] != "a" || calls[1] != "b" || calls[2] != "a" {
		t.Fatalf("calls = %v, want [a b a]", calls)
	}
}

func TestConnectBestContextCancelled(t *testing.T) {
	mgr := New(session.New(nil, false, false), Options{RFCOMMPath: "/dev/rfcomm0", Channel: 15})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := mgr.ConnectBest(ctx, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("ConnectBest() = %v, want context.Canceled", err)
	}
}

func TestConnectViaExisting(t *testing.T) {
	const mac = "AA:BB:CC:DD:EE:FF"
	mgr := New(session.New(nil, false, false), Options{RFCOMMPath: "/dev/rfcomm0", Channel: 15})
	var connected bool
	autoTestHooksVar = &autoTestHooks{
		connect: func(m *Manager, ctx context.Context, dev bt.Device) error {
			connected = dev.MAC == mac
			return nil
		},
	}
	t.Cleanup(func() { autoTestHooksVar = nil })

	oldConnected := hookIsDeviceConnected
	oldOutput := hookHasBluetoothAudioSink
	hookIsDeviceConnected = func(string) (bool, error) { return true, nil }
	hookHasBluetoothAudioSink = func(context.Context, string) (bool, error) { return true, nil }
	t.Cleanup(func() {
		hookIsDeviceConnected = oldConnected
		hookHasBluetoothAudioSink = oldOutput
	})

	var statuses []string
	err := mgr.connectViaExisting(context.Background(), bt.Device{MAC: mac}, func(msg string) {
		statuses = append(statuses, msg)
	})
	if err != nil || !connected {
		t.Fatalf("connectViaExisting() err=%v connected=%v", err, connected)
	}
	if len(statuses) == 0 || statuses[len(statuses)-1] != "auto: connecting via existing /dev/rfcomm0" {
		t.Fatalf("statuses = %v", statuses)
	}
}

func TestConnectViaExistingWaitingBluetooth(t *testing.T) {
	mgr := New(session.New(nil, false, false), Options{RFCOMMPath: "/dev/rfcomm0", Channel: 15})
	oldConnected := hookIsDeviceConnected
	hookIsDeviceConnected = func(string) (bool, error) { return false, nil }
	t.Cleanup(func() { hookIsDeviceConnected = oldConnected })

	err := mgr.connectViaExisting(context.Background(), bt.Device{MAC: "AA:BB:CC:DD:EE:FF"}, func(string) {})
	if !errors.Is(err, errWaitingForBluetooth) {
		t.Fatalf("connectViaExisting() = %v", err)
	}
}

func TestConnectViaExistingReadinessError(t *testing.T) {
	mgr := New(session.New(nil, false, false), Options{RFCOMMPath: "/dev/rfcomm0", Channel: 15})
	want := errors.New("readiness failed")
	oldConnected := hookIsDeviceConnected
	hookIsDeviceConnected = func(string) (bool, error) { return false, want }
	t.Cleanup(func() { hookIsDeviceConnected = oldConnected })

	var msg string
	err := mgr.connectViaExisting(context.Background(), bt.Device{MAC: "AA:BB:CC:DD:EE:FF"}, func(s string) { msg = s })
	if !errors.Is(err, want) {
		t.Fatalf("connectViaExisting() = %v", err)
	}
	if msg == "" {
		t.Fatal("expected status message")
	}
}

func TestConnectBestExistingRFCOMMNoMAC(t *testing.T) {
	mgr := New(session.New(nil, false, false), Options{RFCOMMPath: "/dev/rfcomm0", Channel: 15})
	autoTestHooksVar = &autoTestHooks{
		rfcommExists: func(m *Manager) (bool, error) { return true, nil },
		discover: func(m *Manager, ctx context.Context) ([]bt.Device, error) {
			return []bt.Device{{MAC: "AA:BB:CC:DD:EE:FF", Connected: true}}, nil
		},
		bind: func(m *Manager, ctx context.Context, dev bt.Device) error { return nil },
		connect: func(m *Manager, ctx context.Context, dev bt.Device) error {
			return nil
		},
	}
	t.Cleanup(func() { autoTestHooksVar = nil })

	oldConnected := hookIsDeviceConnected
	oldOutput := hookHasBluetoothAudioSink
	hookIsDeviceConnected = func(string) (bool, error) { return true, nil }
	hookHasBluetoothAudioSink = func(context.Context, string) (bool, error) { return true, nil }
	t.Cleanup(func() {
		hookIsDeviceConnected = oldConnected
		hookHasBluetoothAudioSink = oldOutput
	})

	if err := mgr.ConnectBest(context.Background(), nil); err != nil {
		t.Fatalf("ConnectBest() = %v", err)
	}
}

func TestConnectBestExistingStaleMACRescans(t *testing.T) {
	const stale, live = "AA:BB:CC:DD:EE:FF", "2C:BE:EE:4A:EC:9E"
	transportRef := "/dev/rfcomm0"
	if runtime.GOOS == "darwin" {
		transportRef = "rfcomm:" + stale + ":15"
	}
	cfgPath := t.TempDir() + "/devices.json"
	bt.SetConfigPathHook(func() string { return cfgPath })
	t.Cleanup(func() { bt.SetConfigPathHook(nil) })
	if err := bt.RememberDeviceMAC(transportRef, stale); err != nil {
		t.Fatal(err)
	}

	var boundMAC string
	mgr := New(session.New(nil, false, false), Options{RFCOMMPath: transportRef, Channel: 15})
	autoTestHooksVar = &autoTestHooks{
		rfcommExists: func(m *Manager) (bool, error) { return true, nil },
		discover: func(m *Manager, ctx context.Context) ([]bt.Device, error) {
			return []bt.Device{{MAC: live, Connected: true, SPP: true, Name: "Nothing Ear (3)"}}, nil
		},
		bind: func(m *Manager, ctx context.Context, dev bt.Device) error {
			boundMAC = dev.MAC
			return nil
		},
		connect: func(m *Manager, ctx context.Context, dev bt.Device) error { return nil },
	}
	t.Cleanup(func() { autoTestHooksVar = nil })

	oldConnected := hookIsDeviceConnected
	oldOutput := hookHasBluetoothAudioSink
	hookIsDeviceConnected = func(mac string) (bool, error) {
		return mac == live, nil
	}
	hookHasBluetoothAudioSink = func(_ context.Context, mac string) (bool, error) {
		return mac == live, nil
	}
	t.Cleanup(func() {
		hookIsDeviceConnected = oldConnected
		hookHasBluetoothAudioSink = oldOutput
	})

	var statuses []string
	if err := mgr.ConnectBest(context.Background(), func(msg string) { statuses = append(statuses, msg) }); err != nil {
		t.Fatalf("ConnectBest() = %v statuses=%v", err, statuses)
	}
	if boundMAC != live {
		t.Fatalf("bound MAC=%q want %s statuses=%v", boundMAC, live, statuses)
	}
}

func TestConnectBestDiscoverAndBind(t *testing.T) {
	const mac = "11:22:33:44:55:66"
	mgr := New(session.New(nil, false, false), Options{RFCOMMPath: "/dev/rfcomm0", Channel: 15})
	var bound, connected bool

	autoTestHooksVar = &autoTestHooks{
		rfcommExists: func(m *Manager) (bool, error) { return false, nil },
		discover: func(m *Manager, ctx context.Context) ([]bt.Device, error) {
			return []bt.Device{{MAC: mac, Connected: true, SPP: true}}, nil
		},
		bind: func(m *Manager, ctx context.Context, dev bt.Device) error {
			bound = true
			return nil
		},
		connect: func(m *Manager, ctx context.Context, dev bt.Device) error {
			connected = true
			return nil
		},
	}
	t.Cleanup(func() { autoTestHooksVar = nil })

	oldConnected := hookIsDeviceConnected
	oldOutput := hookHasBluetoothAudioSink
	hookIsDeviceConnected = func(string) (bool, error) { return true, nil }
	hookHasBluetoothAudioSink = func(context.Context, string) (bool, error) { return true, nil }
	t.Cleanup(func() {
		hookIsDeviceConnected = oldConnected
		hookHasBluetoothAudioSink = oldOutput
	})

	if err := mgr.ConnectBest(context.Background(), nil); err != nil {
		t.Fatalf("ConnectBest() = %v", err)
	}
	if !bound || !connected {
		t.Fatalf("bound=%v connected=%v, want both true", bound, connected)
	}
}

func TestConnectBestNoCandidate(t *testing.T) {
	mgr := New(session.New(nil, false, false), Options{RFCOMMPath: "/dev/rfcomm0", Channel: 15})
	autoTestHooksVar = &autoTestHooks{
		rfcommExists: func(m *Manager) (bool, error) { return false, nil },
		discover:     func(m *Manager, ctx context.Context) ([]bt.Device, error) { return nil, nil },
	}
	t.Cleanup(func() { autoTestHooksVar = nil })

	err := mgr.ConnectBest(context.Background(), nil)
	if !errors.Is(err, errNoCandidate) {
		t.Fatalf("ConnectBest() = %v, want errNoCandidate", err)
	}
}

func TestConnectBestWaitingForBluetooth(t *testing.T) {
	mgr := New(session.New(nil, false, false), Options{RFCOMMPath: "/dev/rfcomm0", Channel: 15})
	autoTestHooksVar = &autoTestHooks{
		rfcommExists: func(m *Manager) (bool, error) { return false, nil },
		discover: func(m *Manager, ctx context.Context) ([]bt.Device, error) {
			return []bt.Device{{MAC: "AA:BB:CC:DD:EE:FF", Connected: false}}, nil
		},
	}
	t.Cleanup(func() { autoTestHooksVar = nil })

	err := mgr.ConnectBest(context.Background(), nil)
	if !errors.Is(err, errWaitingForBluetooth) {
		t.Fatalf("ConnectBest() = %v, want errWaitingForBluetooth", err)
	}
}

func TestConnectBestReadinessWaits(t *testing.T) {
	mgr := New(session.New(nil, false, false), Options{RFCOMMPath: "/dev/rfcomm0", Channel: 15})
	autoTestHooksVar = &autoTestHooks{
		rfcommExists: func(m *Manager) (bool, error) { return false, nil },
		discover: func(m *Manager, ctx context.Context) ([]bt.Device, error) {
			return []bt.Device{{MAC: "AA:BB:CC:DD:EE:FF", Connected: true}}, nil
		},
	}
	t.Cleanup(func() { autoTestHooksVar = nil })

	oldConnected := hookIsDeviceConnected
	oldOutput := hookHasBluetoothAudioSink
	hookIsDeviceConnected = func(string) (bool, error) { return false, nil }
	hookHasBluetoothAudioSink = func(context.Context, string) (bool, error) { return true, nil }
	t.Cleanup(func() {
		hookIsDeviceConnected = oldConnected
		hookHasBluetoothAudioSink = oldOutput
	})

	err := mgr.ConnectBest(context.Background(), nil)
	if !errors.Is(err, errWaitingForBluetooth) {
		t.Fatalf("ConnectBest() = %v, want errWaitingForBluetooth", err)
	}

	hookIsDeviceConnected = func(string) (bool, error) { return true, nil }
	hookHasBluetoothAudioSink = func(context.Context, string) (bool, error) { return false, nil }
	err = mgr.ConnectBest(context.Background(), nil)
	if !errors.Is(err, errWaitingForAudioOutput) {
		t.Fatalf("ConnectBest() = %v, want errWaitingForAudioOutput", err)
	}
}

func TestAutoConnectCancelsDuringSleep(t *testing.T) {
	sess := session.New(nil, false, false)
	mgr := New(sess, Options{RFCOMMPath: "/dev/rfcomm0", Channel: 15})
	ctx, cancel := context.WithCancel(context.Background())

	var attempts atomic.Int32
	autoTestHooksVar = &autoTestHooks{
		rfcommExists: func(m *Manager) (bool, error) { return false, nil },
		discover: func(m *Manager, ctx context.Context) ([]bt.Device, error) {
			attempts.Add(1)
			cancel()
			return nil, errNoCandidate
		},
	}
	t.Cleanup(func() { autoTestHooksVar = nil })

	mgr.runAutoConnectLoop(ctx, time.Second, newStatusReporter(nil))
	if attempts.Load() < 1 {
		t.Fatal("expected at least one connect attempt")
	}
}

func TestConnectBestRebindExistingRFCOMM(t *testing.T) {
	const mac = "AA:BB:CC:DD:EE:FF"
	const rfcommPath = "/dev/rfcomm98"
	mgr := New(session.New(nil, false, false), Options{RFCOMMPath: rfcommPath, Channel: 15})
	var released, bound bool
	oldRelease := hookReleaseRFCOMMDevice
	hookReleaseRFCOMMDevice = func(string) error { released = true; return nil }
	t.Cleanup(func() { hookReleaseRFCOMMDevice = oldRelease })

	autoTestHooksVar = &autoTestHooks{
		rfcommExists: func(m *Manager) (bool, error) { return true, nil },
		discover: func(m *Manager, ctx context.Context) ([]bt.Device, error) {
			return []bt.Device{{MAC: mac, Connected: true, SPP: true}}, nil
		},
		bind: func(m *Manager, ctx context.Context, dev bt.Device) error {
			bound = true
			return nil
		},
		connect: func(m *Manager, ctx context.Context, dev bt.Device) error { return nil },
	}
	t.Cleanup(func() { autoTestHooksVar = nil })

	oldConnected := hookIsDeviceConnected
	oldOutput := hookHasBluetoothAudioSink
	hookIsDeviceConnected = func(string) (bool, error) { return true, nil }
	hookHasBluetoothAudioSink = func(context.Context, string) (bool, error) { return true, nil }
	t.Cleanup(func() {
		hookIsDeviceConnected = oldConnected
		hookHasBluetoothAudioSink = oldOutput
	})

	if err := mgr.ConnectBest(context.Background(), nil); err != nil {
		t.Fatalf("ConnectBest() = %v", err)
	}
	if !released || !bound {
		t.Fatalf("released=%v bound=%v, want both true", released, bound)
	}
}

func TestAutoConnectIdlesWhenConnected(t *testing.T) {
	sess := session.New(nil, false, false)
	f, err := os.CreateTemp(t.TempDir(), "rfcomm")
	if err != nil {
		t.Fatal(err)
	}
	sess.AttachTestLink(bt.NewTestTransport(f, "AA:BB:CC:DD:EE:FF", 15, "/dev/rfcomm0"), bt.Device{MAC: "AA:BB:CC:DD:EE:FF"})

	mgr := New(sess, Options{RFCOMMPath: "/dev/rfcomm0", Channel: 15})
	ctx, cancel := context.WithCancel(context.Background())
	var discoverCalls atomic.Int32
	autoTestHooksVar = &autoTestHooks{
		discover: func(m *Manager, ctx context.Context) ([]bt.Device, error) {
			discoverCalls.Add(1)
			return nil, errNoCandidate
		},
	}
	t.Cleanup(func() { autoTestHooksVar = nil })

	cancelTimer := time.AfterFunc(40*time.Millisecond, cancel)
	t.Cleanup(func() { cancelTimer.Stop() })
	mgr.runAutoConnectLoop(ctx, 15*time.Millisecond, newStatusReporter(nil))
	if discoverCalls.Load() != 0 {
		t.Fatalf("discover calls = %d, want 0 while connected", discoverCalls.Load())
	}
}

func TestAutoConnectReportsStatus(t *testing.T) {
	sess := session.New(nil, false, false)
	mgr := New(sess, Options{RFCOMMPath: "/dev/rfcomm0", Channel: 15})
	ctx, cancel := context.WithCancel(context.Background())

	var statuses []string
	autoTestHooksVar = &autoTestHooks{
		rfcommExists: func(m *Manager) (bool, error) { return false, nil },
		discover:     func(m *Manager, ctx context.Context) ([]bt.Device, error) { return nil, nil },
	}
	t.Cleanup(func() { autoTestHooksVar = nil })

	cancelTimer := time.AfterFunc(30*time.Millisecond, cancel)
	t.Cleanup(func() { cancelTimer.Stop() })

	mgr.runAutoConnectLoop(ctx, 10*time.Millisecond, newStatusReporter(func(msg string) {
		statuses = append(statuses, msg)
	}))

	found := false
	for _, s := range statuses {
		if s == "auto: no compatible TWS device found" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("statuses = %v, want no-candidate message", statuses)
	}
}

func TestAutoConnectWaitingStatusMessages(t *testing.T) {
	tests := []struct {
		name            string
		deviceConnected bool
		hasAudioSink    bool
		want            string
	}{
		{name: "bluetooth", deviceConnected: false, hasAudioSink: true, want: "Bluetooth disconnected"},
		{name: "audio", deviceConnected: true, hasAudioSink: false, want: "A2DP"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sess := session.New(nil, false, false)
			mgr := New(sess, Options{RFCOMMPath: "/dev/rfcomm0", Channel: 15})
			ctx, cancel := context.WithCancel(context.Background())

			autoTestHooksVar = &autoTestHooks{
				rfcommExists: func(m *Manager) (bool, error) { return false, nil },
				discover: func(m *Manager, ctx context.Context) ([]bt.Device, error) {
					return []bt.Device{{MAC: "AA:BB:CC:DD:EE:FF", Connected: true}}, nil
				},
			}
			oldConnected := hookIsDeviceConnected
			oldOutput := hookHasBluetoothAudioSink
			hookIsDeviceConnected = func(string) (bool, error) { return tt.deviceConnected, nil }
			hookHasBluetoothAudioSink = func(context.Context, string) (bool, error) { return tt.hasAudioSink, nil }
			t.Cleanup(func() {
				autoTestHooksVar = nil
				hookIsDeviceConnected = oldConnected
				hookHasBluetoothAudioSink = oldOutput
				cancel()
			})

			var statuses []string
			cancelTimer := time.AfterFunc(30*time.Millisecond, cancel)
			t.Cleanup(func() { cancelTimer.Stop() })
			mgr.runAutoConnectLoop(ctx, 5*time.Millisecond, newStatusReporter(func(msg string) {
				statuses = append(statuses, msg)
			}))

			found := false
			for _, s := range statuses {
				if strings.Contains(s, tt.want) {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("statuses = %v, want substring %q", statuses, tt.want)
			}
		})
	}
}
