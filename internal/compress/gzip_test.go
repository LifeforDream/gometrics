package compress

import (
	"bytes"
	"compress/gzip"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriter(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
	}{
		{
			name:  "plain text round-trip",
			input: []byte("hello, gzip"),
		},
		{
			name:  "json payload round-trip",
			input: []byte(`[{"id":"alloc","type":"gauge","value":1.25}]`),
		},
		{
			name:  "empty input",
			input: []byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			w, err := NewWriter(&buf)
			require.NoError(t, err)

			_, err = w.Write(tt.input)
			require.NoError(t, err)

			require.NoError(t, w.Close())

			gr, err := gzip.NewReader(&buf)
			require.NoError(t, err)
			defer gr.Close()

			got, err := io.ReadAll(gr)
			require.NoError(t, err)

			assert.Equal(t, tt.input, got)
		})
	}
}
