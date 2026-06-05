package main

import "testing"

func TestParseOwner(t *testing.T) {
	if _, _, err := parseOwner("1000:1000"); err != nil {
		t.Fatalf("parseOwner valid: %v", err)
	}
	tests := []string{"", "1000", "a:b", "1:-2", "1:"}
	for _, tc := range tests {
		t.Run(tc, func(t *testing.T) {
			if _, _, err := parseOwner(tc); err == nil {
				t.Fatalf("parseOwner(%q) expected error", tc)
			}
		})
	}
}

func TestNormalizeDeviceAndNumber(t *testing.T) {
	dev, num, err := normalizeDeviceAndNumber("/dev/rfcomm2", "")
	if err != nil {
		t.Fatalf("normalize with device: %v", err)
	}
	if dev != "/dev/rfcomm2" || num != "2" {
		t.Fatalf("got dev=%q num=%q", dev, num)
	}
	if _, _, err := normalizeDeviceAndNumber("", "2"); err != nil {
		t.Fatalf("normalize with number: %v", err)
	}
	if _, _, err := normalizeDeviceAndNumber("/dev/rfcomm1", "2"); err == nil {
		t.Fatal("expected mismatch error")
	}
}

func TestRunValidationErrors(t *testing.T) {
	cases := [][]string{
		{},
		{"unknown"},
		{"bind", "--number", "0", "--addr", "bad", "--channel", "15", "--owner", "1000:1000"},
		{"bind", "--number", "0", "--addr", "AA:BB:CC:DD:EE:FF", "--channel", "15"},
		{"release"},
		{"fix-perms", "--device", "/tmp/not-rfcomm", "--owner", "1000:1000"},
	}
	for _, args := range cases {
		t.Run(joinArgs(args), func(t *testing.T) {
			if err := run(args); err == nil {
				t.Fatalf("run(%v) expected error", args)
			}
		})
	}
}

func joinArgs(args []string) string {
	if len(args) == 0 {
		return "empty"
	}
	out := args[0]
	for _, a := range args[1:] {
		out += "_" + a
	}
	return out
}
