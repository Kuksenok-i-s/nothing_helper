package spp

import (
	"testing"
	"time"
)

func TestValidateANCModeValue(t *testing.T) {
	for _, v := range []byte{0, 1, 2, 3, 4, 5, 252, 253, 254, 255} {
		if err := ValidateANCModeValue(v); err != nil {
			t.Fatalf("ValidateANCModeValue(%d) = %v", v, err)
		}
	}
	if err := ValidateANCModeValue(255); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseANCModeArg("255"); err != nil {
		t.Fatalf("numeric 255 should be allowed: %v", err)
	}
	if _, err := ParseANCModeArg("comfortable"); err != nil {
		t.Fatalf("comfortable: %v", err)
	}
	if _, err := ParseANCModeArg("99"); err == nil {
		t.Fatal("expected error for ANC mode 99")
	}
	// Real Ear devices report/accept 7 for transparency (confirmed via 0xE003 events).
	if v, err := ParseANCModeArg("transparency"); err != nil || v != 7 {
		t.Fatalf("transparency = %d, %v; want 7", v, err)
	}
	if got, err := BuildANCSetPayload(DefaultModel(), []string{"transparency"}); err != nil || len(got) != 3 || got[0] != 1 || got[1] != 7 || got[2] != 0 {
		t.Fatalf("BuildANCSetPayload(transparency) = %v, %v; want [1 7 0]", got, err)
	}
	earThree, _ := ResolveModelInfo("EarThree")
	if got, err := BuildANCSetPayload(earThree, []string{"off"}); err != nil || !bytesEqual(got, []byte{1, 5, 0}) {
		t.Fatalf("EarThree anc off = %v, %v; want [1 5 0]", got, err)
	}
	earOne, _ := ResolveModelInfo("EarOne")
	if got, err := BuildANCSetPayload(earOne, []string{"off"}); err != nil || !bytesEqual(got, []byte{1, 0, 0}) {
		t.Fatalf("EarOne anc off = %v, %v; want [1 0 0]", got, err)
	}
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestValidateEQModeByModel(t *testing.T) {
	donphan, _ := ResolveModelInfo("Donphan")
	twos, _ := ResolveModelInfo("EarTwos")

	if err := ValidateEQModeValue(donphan, 3); err != nil {
		t.Fatal(err)
	}
	if err := ValidateEQModeValue(donphan, 4); err == nil {
		t.Fatal("Donphan should reject EQ mode 4")
	}
	if err := ValidateEQModeValue(twos, 7); err != nil {
		t.Fatal(err)
	}
	if err := ValidateEQModeValue(twos, 8); err == nil {
		t.Fatal("EarTwos should reject EQ mode 8")
	}
}

func TestValidateScanRange(t *testing.T) {
	if err := ValidateScanRange(0xC001, 0xC005, 50*time.Millisecond); err == nil {
		t.Fatal("expected delay error")
	}
	if err := ValidateScanRange(0xC001, 0xC030, time.Second); err == nil {
		t.Fatal("expected range size error")
	}
	if err := ValidateScanRange(0xC001, 0xF001, time.Second); err == nil {
		t.Fatal("expected unsafe command in range")
	}
	if err := ValidateScanRange(0xC001, 0xC005, 500*time.Millisecond); err != nil {
		t.Fatal(err)
	}
}
