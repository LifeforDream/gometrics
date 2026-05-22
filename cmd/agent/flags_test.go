package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConstructAddress(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		address string
		secure  bool
		want    string
	}{
		{
			name:    "bare host, insecure",
			address: "localhost:8080",
			secure:  false,
			want:    "http://localhost:8080",
		},
		{
			name:    "bare host, secure",
			address: "localhost:8080",
			secure:  true,
			want:    "https://localhost:8080",
		},
		{
			name:    "already has http scheme",
			address: "http://localhost:8080",
			secure:  false,
			want:    "http://localhost:8080",
		},
		{
			name:    "already has https scheme",
			address: "https://localhost:8080",
			secure:  false,
			want:    "https://localhost:8080",
		},
		{
			name:    "uppercase scheme is preserved as-is",
			address: "HTTP://localhost:8080",
			secure:  false,
			want:    "HTTP://localhost:8080",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agentOptions.address = tt.address
			agentOptions.secure = tt.secure
			got := constructAddress()
			assert.Equal(t, tt.want, got)
		})
	}
}
