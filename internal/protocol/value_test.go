package protocol

import (
	"encoding/json"
	"testing"
)

func TestParseInt(t *testing.T) {
	tests := []struct {
		name string
		in   interface{}
		want int
		ok   bool
	}{
		{name: "float64", in: float64(123), want: 123, ok: true},
		{name: "int", in: 456, want: 456, ok: true},
		{name: "string", in: "789", want: 789, ok: true},
		{name: "string with spaces", in: " 42 ", want: 42, ok: true},
		{name: "json number", in: json.Number("9001"), want: 9001, ok: true},
		{name: "decimal string", in: "12.5", ok: false},
		{name: "decimal float", in: 12.5, ok: false},
		{name: "empty string", in: "", ok: false},
		{name: "invalid string", in: "abc", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseInt(tt.in)
			if ok != tt.ok {
				t.Fatalf("ParseInt(%v) ok = %v, want %v", tt.in, ok, tt.ok)
			}
			if got != tt.want {
				t.Fatalf("ParseInt(%v) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}
