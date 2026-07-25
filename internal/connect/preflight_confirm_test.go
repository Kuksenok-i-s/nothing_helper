package connect

import "testing"

func TestDefaultConfirmAcceptsYes(t *testing.T) {
	old := defaultReadLineHook
	defaultReadLineHook = func() (string, error) { return "yes\n", nil }
	t.Cleanup(func() { defaultReadLineHook = old })

	if !defaultConfirm("Create RFCOMM?") {
		t.Fatal("expected confirm true")
	}
}

func TestDefaultConfirmRejectsNo(t *testing.T) {
	old := defaultReadLineHook
	defaultReadLineHook = func() (string, error) { return "no\n", nil }
	t.Cleanup(func() { defaultReadLineHook = old })

	if defaultConfirm("Create RFCOMM?") {
		t.Fatal("expected confirm false")
	}
}
