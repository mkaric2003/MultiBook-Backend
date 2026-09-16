package auth

import "testing"

func TestBearerToken(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
		ok     bool
	}{
		{name: "valid", header: "Bearer token-value", want: "token-value", ok: true},
		{name: "case insensitive scheme", header: "bearer token-value", want: "token-value", ok: true},
		{name: "missing", header: "", ok: false},
		{name: "wrong scheme", header: "Basic token-value", ok: false},
		{name: "too many values", header: "Bearer one two", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := bearerToken(tt.header)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("bearerToken(%q) = (%q, %t), want (%q, %t)", tt.header, got, ok, tt.want, tt.ok)
			}
		})
	}
}
