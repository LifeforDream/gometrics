package mwip

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/LifeforDream/gometrics/internal/utils"
)

const (
	realIPHeader    = "X-Real-IP"
	forwardedHeader = "X-Forwarded-For"
)

func TestWithClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		want       string
	}{
		{
			name:       "RemoteAddr with port",
			remoteAddr: "192.168.0.42:54321",
			want:       "192.168.0.42",
		},
		{
			name:       "RemoteAddr without port",
			remoteAddr: "192.168.0.42",
			want:       "192.168.0.42",
		},
		{
			name:       "IPv6 RemoteAddr",
			remoteAddr: "[::1]:54321",
			want:       "::1",
		},
		{
			name:       "X-Real-IP wins over RemoteAddr",
			remoteAddr: "10.0.0.1:54321",
			headers:    map[string]string{realIPHeader: "192.168.0.42"},
			want:       "192.168.0.42",
		},
		{
			name:       "X-Forwarded-For wins over RemoteAddr when X-Real-IP absent",
			remoteAddr: "10.0.0.1:54321",
			headers:    map[string]string{forwardedHeader: "203.0.113.9, 10.0.0.1"},
			want:       "203.0.113.9",
		},
		{
			name:       "X-Real-IP wins over X-Forwarded-For",
			remoteAddr: "10.0.0.1:54321",
			headers: map[string]string{
				realIPHeader:    "192.168.0.42",
				forwardedHeader: "203.0.113.9, 10.0.0.1",
			},
			want: "192.168.0.42",
		},
		{
			name:       "empty headers fall back to RemoteAddr",
			remoteAddr: "10.0.0.1:54321",
			headers: map[string]string{
				realIPHeader:    "  ",
				forwardedHeader: "",
			},
			want: "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			h := WithClientIP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got = utils.ClientIP(r.Context())
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodPost, "/update", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWithTrustedSubnet(t *testing.T) {
	mustCIDR := func(t *testing.T, cidr string) *net.IPNet {
		t.Helper()
		_, n, err := net.ParseCIDR(cidr)
		require.NoError(t, err)
		return n
	}

	tests := []struct {
		name       string
		subnet     string // пустая строка — доверенная подсеть не задана
		remoteAddr string
		headers    map[string]string
		wantStatus int
		wantCalled bool
	}{
		{
			name:       "no subnet - any IP accepted",
			remoteAddr: "10.0.0.1:54321",
			headers:    map[string]string{realIPHeader: "203.0.113.9"},
			wantStatus: http.StatusOK,
			wantCalled: true,
		},
		{
			name:       "no subnet - unparseable IP still accepted",
			remoteAddr: "10.0.0.1:54321",
			headers:    map[string]string{realIPHeader: "not-an-ip"},
			wantStatus: http.StatusOK,
			wantCalled: true,
		},
		{
			name:       "X-Real-IP inside subnet - accepted",
			subnet:     "192.168.0.0/24",
			remoteAddr: "10.0.0.1:54321",
			headers:    map[string]string{realIPHeader: "192.168.0.42"},
			wantStatus: http.StatusOK,
			wantCalled: true,
		},
		{
			name:       "X-Real-IP outside subnet - forbidden",
			subnet:     "192.168.0.0/24",
			remoteAddr: "192.168.0.1:54321",
			headers:    map[string]string{realIPHeader: "192.168.1.42"},
			wantStatus: http.StatusForbidden,
			wantCalled: false,
		},
		{
			name:       "subnet boundary: network address is inside",
			subnet:     "192.168.0.0/24",
			remoteAddr: "10.0.0.1:54321",
			headers:    map[string]string{realIPHeader: "192.168.0.0"},
			wantStatus: http.StatusOK,
			wantCalled: true,
		},
		{
			name:       "subnet boundary: broadcast address is inside",
			subnet:     "192.168.0.0/24",
			remoteAddr: "10.0.0.1:54321",
			headers:    map[string]string{realIPHeader: "192.168.0.255"},
			wantStatus: http.StatusOK,
			wantCalled: true,
		},
		{
			name:       "no headers - RemoteAddr checked against subnet",
			subnet:     "127.0.0.0/8",
			remoteAddr: "127.0.0.1:54321",
			wantStatus: http.StatusOK,
			wantCalled: true,
		},
		{
			name:       "no headers - RemoteAddr outside subnet forbidden",
			subnet:     "192.168.0.0/24",
			remoteAddr: "10.0.0.1:54321",
			wantStatus: http.StatusForbidden,
			wantCalled: false,
		},
		{
			name:       "X-Forwarded-For ignored - RemoteAddr outside subnet forbidden",
			subnet:     "203.0.113.0/24",
			remoteAddr: "10.0.0.1:54321",
			headers:    map[string]string{forwardedHeader: "203.0.113.9, 10.0.0.1"},
			wantStatus: http.StatusForbidden,
			wantCalled: false,
		},
		{
			name:       "unparseable IP with subnet - forbidden",
			subnet:     "192.168.0.0/24",
			remoteAddr: "192.168.0.1:54321",
			headers:    map[string]string{realIPHeader: "not-an-ip"},
			wantStatus: http.StatusForbidden,
			wantCalled: false,
		},
		{
			name:       "IPv6 inside subnet - accepted",
			subnet:     "2001:db8::/32",
			remoteAddr: "10.0.0.1:54321",
			headers:    map[string]string{realIPHeader: "2001:db8::1"},
			wantStatus: http.StatusOK,
			wantCalled: true,
		},
		{
			name:       "IPv4 client against IPv6 subnet - forbidden",
			subnet:     "2001:db8::/32",
			remoteAddr: "10.0.0.1:54321",
			headers:    map[string]string{realIPHeader: "192.168.0.42"},
			wantStatus: http.StatusForbidden,
			wantCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var network *net.IPNet
			if tt.subnet != "" {
				network = mustCIDR(t, tt.subnet)
			}

			called := false
			h := WithTrustedSubnet(network, zap.NewNop())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodPost, "/updates", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)
			assert.Equal(t, tt.wantCalled, called)
		})
	}
}

func TestClientIPAbsentFromContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	assert.Empty(t, utils.ClientIP(req.Context()))
}
