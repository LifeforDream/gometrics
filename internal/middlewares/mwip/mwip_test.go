package mwip

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

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
			name:       "X-Real-IP loses to RemoteAddr",
			remoteAddr: "10.0.0.1:54321",
			headers:    map[string]string{realIPHeader: "192.168.0.42"},
			want:       "10.0.0.1",
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

func TestClientIPAbsentFromContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	assert.Empty(t, utils.ClientIP(req.Context()))
}
