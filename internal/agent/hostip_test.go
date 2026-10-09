package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutboundIP(t *testing.T) {
	tests := []struct {
		name       string
		serverAddr string
		want       string // пустая строка — проверяется только IsLoopback
		wantErr    bool
	}{
		{
			name:       "IPv4 loopback with port",
			serverAddr: "http://127.0.0.1:8080",
			want:       "127.0.0.1",
		},
		{
			name:       "https scheme",
			serverAddr: "https://127.0.0.1:8443",
			want:       "127.0.0.1",
		},
		{
			name:       "no port - default for scheme is used",
			serverAddr: "http://127.0.0.1",
			want:       "127.0.0.1",
		},
		{
			name:       "hostname is resolved",
			serverAddr: "http://localhost:8080",
		},
		{
			name:       "empty host",
			serverAddr: "http://",
			wantErr:    true,
		},
		{
			name:       "unparseable URL",
			serverAddr: "://127.0.0.1:8080",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := OutboundIP(tt.serverAddr)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.True(t, got.IsLoopback(), "expected loopback address, got %s", got)
			if tt.want != "" {
				assert.Equal(t, tt.want, got.String())
			}
		})
	}
}

func TestOutboundIPHostPort(t *testing.T) {
	tests := []struct {
		name     string
		hostport string
		want     string // пустая строка — проверяется только IsLoopback
		wantErr  bool
	}{
		{
			name:     "IPv4 loopback with port",
			hostport: "127.0.0.1:8080",
			want:     "127.0.0.1",
		},
		{
			name:     "hostname is resolved",
			hostport: "localhost:8080",
		},
		{
			name:     "missing port is an error",
			hostport: "127.0.0.1",
			wantErr:  true,
		},
		{
			name:     "empty hostport is an error",
			hostport: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := OutboundIPHostPort(tt.hostport)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.True(t, got.IsLoopback(), "expected loopback address, got %s", got)
			if tt.want != "" {
				assert.Equal(t, tt.want, got.String())
			}
		})
	}
}
