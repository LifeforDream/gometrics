package utils

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenSHA256(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		key  string
		want string
	}{
		{
			name: "basic",
			data: []byte("hello"),
			key:  "secret",
			want: "85e76456e64bed8bf17e43ba99aa01d296e7d7ecc626ccc2d2132388c9f12159",
		},
		{
			name: "empty key",
			data: []byte("hello"),
			key:  "",
			want: "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
		},
		{
			name: "empty data",
			data: []byte{},
			key:  "secret",
			want: "2bb80d537b1da3e38bd30361aa855686bde0eacd7162fef6a25fe97bf527a25b",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hex.EncodeToString(GenSHA256(tt.data, tt.key))
			assert.Equal(t, tt.want, got)
		})
	}
}
