package bt

import "testing"

func TestParsePrivilegeMode(t *testing.T) {
	tests := []struct {
		in      string
		want    PrivilegeMode
		wantErr bool
	}{
		{in: "", want: PrivilegeModeSudo},
		{in: "sudo", want: PrivilegeModeSudo},
		{in: "polkit", want: PrivilegeModePolkit},
		{in: "auto", want: PrivilegeModeAuto},
		{in: "none", want: PrivilegeModeNone},
		{in: "bad", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := ParsePrivilegeMode(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParsePrivilegeMode(%q) err=%v wantErr=%v", tt.in, err, tt.wantErr)
			}
			if err == nil && got != tt.want {
				t.Fatalf("ParsePrivilegeMode(%q)=%q want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestPolkitCommandArgs(t *testing.T) {
	bind, err := polkitBindArgs("0", "AA:BB:CC:DD:EE:FF", 15)
	if err != nil {
		t.Fatalf("polkitBindArgs() err=%v", err)
	}
	if got, want := bind[0], "bind"; got != want {
		t.Fatalf("bind action=%q want %q", got, want)
	}
	release, err := polkitReleaseArgs("0")
	if err != nil {
		t.Fatalf("polkitReleaseArgs() err=%v", err)
	}
	if got, want := release[0], "release"; got != want {
		t.Fatalf("release action=%q want %q", got, want)
	}
	fix, err := polkitFixPermsArgs("/dev/rfcomm0", "1000:1000")
	if err != nil {
		t.Fatalf("polkitFixPermsArgs() err=%v", err)
	}
	if got, want := fix[0], "fix-perms"; got != want {
		t.Fatalf("fix action=%q want %q", got, want)
	}
}
