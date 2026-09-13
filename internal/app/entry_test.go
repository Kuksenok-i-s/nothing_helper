package app

import (
	"context"
	"testing"

	"nothing_helper/internal/dualpolicy"
	"nothing_helper/internal/session"
)

func testRuntime(notify bool) *Runtime {
	return &Runtime{
		Session: session.New(nil, false, false),
		Config: Config{
			RFCOMMDevice: "/dev/rfcomm0",
			Channel:      15,
			PCPrimary:    "off",
			Notify:       notify,
		},
	}
}

func TestWireServices(t *testing.T) {
	svc, err := WireServices(context.Background(), testRuntime(false))
	if err != nil {
		t.Fatalf("WireServices() = %v", err)
	}
	if svc.Manager == nil || svc.PCPrimaryMode != dualpolicy.ModeOff {
		t.Fatalf("services=%+v", svc)
	}
}

func TestWireServicesInvalidPCPrimary(t *testing.T) {
	rt := testRuntime(false)
	rt.Config.PCPrimary = "not-a-mode"
	_, err := WireServices(context.Background(), rt)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestWireServicesWithNotify(t *testing.T) {
	svc, err := WireServices(context.Background(), testRuntime(true))
	if err != nil {
		t.Fatalf("WireServices() = %v", err)
	}
	if svc.Manager == nil {
		t.Fatal("expected manager")
	}
}

func TestNotifyAppName(t *testing.T) {
	if got := notifyAppName(Config{}); got != "Nothing Ear" {
		t.Fatalf("notifyAppName() = %q", got)
	}
}
