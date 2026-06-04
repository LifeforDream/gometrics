package logger

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestWithLogging(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		method     string
		path       string
		wantStatus int
		wantSize   int
	}{
		{
			name:       "explicit status is logged",
			handler:    func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusCreated) },
			method:     http.MethodPost,
			path:       "/update/gauge/Alloc/1.5",
			wantStatus: http.StatusCreated,
			wantSize:   0,
		},
		{
			name: "implicit 200 and body size are logged",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("1.25"))
			},
			method:     http.MethodGet,
			path:       "/value/gauge/Alloc",
			wantStatus: http.StatusOK,
			wantSize:   4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// arrange: replace global logger with an observer
			core, logs := observer.New(zapcore.InfoLevel)
			original := Log
			Log = zap.New(core)
			t.Cleanup(func() { Log = original })

			ts := httptest.NewServer(WithLogging(tt.handler))
			defer ts.Close()

			req, err := http.NewRequest(tt.method, ts.URL+tt.path, nil)
			require.NoError(t, err)
			resp, err := ts.Client().Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, 1, logs.Len())
			assert.Equal(t, 1, logs.FilterField(zap.Int("status", tt.wantStatus)).Len())
			assert.Equal(t, 1, logs.FilterField(zap.Int("size", tt.wantSize)).Len())
			assert.Equal(t, 1, logs.FilterField(zap.String("uri", tt.path)).Len())
			assert.Equal(t, 1, logs.FilterField(zap.String("method", tt.method)).Len())
		})
	}
}
