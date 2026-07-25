package spp

import "testing"

func TestPrintableText(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
		want    string
		ok      bool
	}{
		{name: "empty", payload: nil, want: "", ok: false},
		{name: "empty slice", payload: []byte{}, want: "", ok: false},
		{name: "ascii", payload: []byte("hello"), want: "hello", ok: true},
		{name: "whitespace", payload: []byte("a\nb\tc\r"), want: "a\nb\tc\r", ok: true},
		{name: "utf8", payload: []byte("café"), want: "café", ok: true},
		{name: "invalid utf8", payload: []byte{0xff, 0xfe}, want: "", ok: false},
		{name: "control char", payload: []byte{0x01, 'a'}, want: "", ok: false},
		{name: "null byte", payload: []byte("ab\x00c"), want: "", ok: false},
		{name: "lone continuation", payload: []byte{0x80}, want: "", ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := printableText(tt.payload)
			if ok != tt.ok || got != tt.want {
				t.Fatalf("printableText(%q) = (%q, %v), want (%q, %v)", tt.payload, got, ok, tt.want, tt.ok)
			}
		})
	}
}
