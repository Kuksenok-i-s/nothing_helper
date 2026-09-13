package companion

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"tws_manager/internal/bt"
	"tws_manager/internal/session"
	"tws_manager/internal/spp"
	"tws_manager/internal/trace"
)

type fakeBackend struct {
	mu    sync.Mutex
	snap  session.Snapshot
	calls [][]string
	err   error
	sent  chan struct{}
}

func (b *fakeBackend) Snapshot() session.Snapshot                          { b.mu.Lock(); defer b.mu.Unlock(); return b.snap }
func (b *fakeBackend) Subscribe() <-chan session.Event                     { return nil }
func (b *fakeBackend) Battery() error                                      { return b.Execute([]string{"battery", "get"}) }
func (b *fakeBackend) ActiveCapture(context.Context, string) (bool, error) { return true, nil }
func (b *fakeBackend) Execute(fields []string) error {
	b.mu.Lock()
	b.calls = append(b.calls, append([]string(nil), fields...))
	err := b.err
	b.mu.Unlock()
	if b.sent != nil {
		select {
		case b.sent <- struct{}{}:
		default:
		}
	}
	return err
}
func connectedBackend(t *testing.T) *fakeBackend {
	t.Helper()
	model, ok := spp.ResolveModelInfo("EarThree")
	if !ok {
		t.Fatal("model not found")
	}
	return &fakeBackend{snap: session.Snapshot{Connected: true, Device: bt.Device{MAC: "AA:BB:CC:DD:EE:01"}, Model: model, Config: map[string]string{"anc": "mode=off"}}}
}
func TestFeatureRequiresConnectedSupportedDevice(t *testing.T) {
	b := connectedBackend(t)
	c := newController(context.Background(), Options{}, b)
	b.snap.Connected = false
	if c.Feature("anc", "off") {
		t.Fatal("disconnected action accepted")
	}
	b.snap.Connected = true
	b.snap.Model, _ = spp.ResolveModelInfo("EarOne")
	if c.Feature("spatial", "on") {
		t.Fatal("unsupported action accepted")
	}
}
func TestFeatureReadsBackWithoutOptimisticState(t *testing.T) {
	b := connectedBackend(t)
	c := newController(context.Background(), Options{}, b)
	if !c.Feature("anc", "strong") {
		t.Fatal("action rejected")
	}
	if got := c.Snapshot().Session.Config["anc"]; got != "mode=off" {
		t.Fatalf("invented confirmed state: %s", got)
	}
	if err := (<-c.queue).run(); err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"anc", "set", "strong"}, {"anc", "get"}}
	if !reflect.DeepEqual(b.calls, want) {
		t.Fatalf("calls = %v", b.calls)
	}
}
func TestQueuedActionDoesNotReachReplacedDevice(t *testing.T) {
	b := connectedBackend(t)
	c := newController(context.Background(), Options{}, b)
	c.Feature("anc", "strong")
	b.snap.Device.MAC = "AA:BB:CC:DD:EE:02"
	if err := (<-c.queue).run(); err == nil {
		t.Fatal("expected changed-connection error")
	}
	if len(b.calls) != 0 {
		t.Fatal("old command sent to new device")
	}
}
func TestReadbackCancelledOnShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b := connectedBackend(t)
	b.sent = make(chan struct{}, 1)
	c := newController(ctx, Options{}, b)
	c.Feature("anc", "strong")
	job := <-c.queue
	done := make(chan error, 1)
	go func() { done <- job.run() }()
	<-b.sent
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.calls) != 1 {
		t.Fatalf("unexpected readback: %v", b.calls)
	}
}
func TestWorkerSerializesAndSurfacesErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b := connectedBackend(t)
	b.err = errors.New("device rejected command")
	c := newController(ctx, Options{}, b)
	c.Feature("anc", "strong")
	go c.worker()
	deadline := time.After(time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("worker did not finish")
		case <-c.Changed():
			snap := c.Snapshot()
			if !snap.Busy {
				if !strings.Contains(snap.Error, "rejected") {
					t.Fatalf("error = %q", snap.Error)
				}
				return
			}
		}
	}
}
func TestExportRespectsRawFlag(t *testing.T) {
	for _, raw := range []bool{false, true} {
		dir := t.TempDir()
		c := newController(context.Background(), Options{CaptureDir: dir, LogRaw: raw}, connectedBackend(t))
		c.presenter.LastEvents = []trace.Event{{Direction: "rx", RawHex: "deadbeef", Summary: "battery"}}
		c.Export()
		if err := (<-c.queue).run(); err != nil {
			t.Fatal(err)
		}
		paths, err := filepath.Glob(filepath.Join(dir, "*_packets.json"))
		if err != nil || len(paths) != 1 {
			t.Fatalf("exports = %v, %v", paths, err)
		}
		data, err := os.ReadFile(paths[0])
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "deadbeef") != raw {
			t.Fatalf("raw=%t export=%s", raw, data)
		}
	}
}
func TestANCMode(t *testing.T) {
	for input, want := range map[string]string{"": "", "mode=off last_level=high": "off", "mode=transparency": "transparency", "mode=high": "strong", "mode=mid": "strong", "mode=low": "strong", "mode=adaptive": "strong"} {
		if got := ANCMode(input); got != want {
			t.Fatalf("ANCMode(%q)=%q want %q", input, got, want)
		}
	}
}
func TestPasswordCancellationDoesNotRetainSecret(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := newController(ctx, Options{}, connectedBackend(t))
	done := make(chan error, 1)
	go func() { _, err := c.Password("Password required"); done <- err }()
	<-c.Changed()
	c.AnswerPassword("")
	if err := <-done; err == nil {
		t.Fatal("cancel accepted")
	}
	if c.Snapshot().PasswordPrompt != "" {
		t.Fatal("password prompt not cleared")
	}
}

func TestWorkerKeepsSetReadbackPairsInOrder(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b := connectedBackend(t)
	c := newController(ctx, Options{}, b)
	c.Feature("anc", "strong")
	c.Feature("eq", "0")
	go c.worker()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("queue did not drain")
		case <-c.Changed():
			if c.Snapshot().Busy {
				continue
			}
			b.mu.Lock()
			calls := append([][]string(nil), b.calls...)
			b.mu.Unlock()
			want := [][]string{{"anc", "set", "strong"}, {"anc", "get"}, {"eq", "set", "0"}, {"eq", "get"}}
			if !reflect.DeepEqual(calls, want) {
				t.Fatalf("commands interleaved: %v", calls)
			}
			return
		}
	}
}
