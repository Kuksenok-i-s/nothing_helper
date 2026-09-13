package companion

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"tws_manager/internal/spp"
)

type micBackend struct {
	*fakeBackend
	requested map[string]string
	failMic   bool
	capture   func() (bool, error)
}

func (b *micBackend) ActiveCapture(context.Context, string) (bool, error) {
	if b.capture != nil {
		return b.capture()
	}
	return true, nil
}

func TestWalkieTalkieRequiresActiveCaptureBeforeWriting(t *testing.T) {
	for _, name := range []string{"idle", "probe error", "stream stopped", "disable while idle"} {
		t.Run(name, func(t *testing.T) {
			checks := 0
			b := &micBackend{fakeBackend: connectedBackend(t), requested: map[string]string{}}
			b.capture = func() (bool, error) {
				checks++
				if name == "probe error" {
					return false, fmt.Errorf("PipeWire unavailable")
				}
				return name == "stream stopped" && checks == 1, nil
			}
			c := newController(context.Background(), Options{}, b)
			value := "on"
			if name == "disable while idle" {
				value = "off"
			}
			if !c.Feature("walkie-talkie", value) {
				t.Fatal("not queued")
			}
			err := (<-c.queue).run()
			if value == "off" {
				if err != nil || checks != 0 {
					t.Fatalf("disable was gated: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("missing audio stream was accepted")
			}
			want := 0
			if name == "stream stopped" {
				want = 2
			}
			if len(b.calls) != want {
				t.Fatalf("unexpected writes: %v", b.calls)
			}
			for _, fields := range b.calls {
				if fields[0] == "walkie-talkie" {
					t.Fatal("mode enabled without audio")
				}
			}
		})
	}
}

func (b *micBackend) Execute(fields []string) error {
	if err := b.fakeBackend.Execute(fields); err != nil {
		return err
	}
	if fields[0] == "super-mic" && b.failMic {
		return fmt.Errorf("mic unavailable")
	}
	if fields[1] == "set" {
		delete(b.snap.Config, fields[0])
		b.requested[fields[0]] = fmt.Sprintf("enabled=%t", fields[2] == "on")
	} else {
		b.snap.Config[fields[0]] = b.requested[fields[0]]
	}
	return nil
}

func TestWalkieTalkieSequence(t *testing.T) {
	for _, value := range []string{"on", "off"} {
		b := &micBackend{fakeBackend: connectedBackend(t), requested: map[string]string{}}
		c := newController(context.Background(), Options{}, b)
		if !c.Feature("walkie-talkie", value) {
			t.Fatal("action rejected")
		}
		if _, ok := b.snap.Config["walkie-talkie"]; ok {
			t.Fatal("optimistic state")
		}
		if err := (<-c.queue).run(); err != nil {
			t.Fatal(err)
		}
		want := [][]string{{"walkie-talkie", "set", value}, {"walkie-talkie", "get"}}
		if value == "on" {
			want = append([][]string{{"super-mic", "set", "on"}, {"super-mic", "get"}}, want...)
		}
		if !reflect.DeepEqual(b.calls, want) {
			t.Fatalf("calls=%v", b.calls)
		}
	}
}

func TestWalkieTalkieStopsOnPrerequisiteFailure(t *testing.T) {
	b := &micBackend{fakeBackend: connectedBackend(t), requested: map[string]string{}, failMic: true}
	c := newController(context.Background(), Options{}, b)
	c.Feature("walkie-talkie", "on")
	if err := (<-c.queue).run(); err == nil {
		t.Fatal("missing failure")
	}
	if len(b.calls) != 1 || b.calls[0][0] != "super-mic" {
		t.Fatalf("sent dependent command: %v", b.calls)
	}
}

func TestWalkieTalkieWaitsForConfirmedSuperMic(t *testing.T) {
	b := connectedBackend(t) // Writes succeed, but this backend never responds.
	c := newController(context.Background(), Options{}, b)
	c.Feature("walkie-talkie", "on")
	if err := (<-c.queue).run(); err == nil {
		t.Fatal("missing readback was accepted")
	}
	if !reflect.DeepEqual(b.calls, [][]string{{"super-mic", "set", "on"}, {"super-mic", "get"}}) {
		t.Fatalf("Walkie Talkie started without Super Mic confirmation: %v", b.calls)
	}
}

func TestWalkieTalkieRejectsUnsupportedAndChangedConnection(t *testing.T) {
	b := connectedBackend(t)
	c := newController(context.Background(), Options{}, b)
	c.Feature("walkie-talkie", "on")
	b.snap.Device.MAC = "00:11:22:33:44:55"
	if err := (<-c.queue).run(); err == nil || len(b.calls) != 0 {
		t.Fatal("wrote to replacement device")
	}
	for _, model := range []string{"EarOne", "EarTwo", ""} {
		b.snap.Model, _ = spp.ResolveModelInfo(model)
		if c.Feature("walkie-talkie", "on") {
			t.Fatalf("accepted model %q", model)
		}
	}
}
