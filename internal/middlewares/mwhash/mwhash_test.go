package mwhash

import (
	"bytes"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/LifeforDream/gometrics/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestWithHash(t *testing.T) {
	const (
		key          = "secretkey"
		requestBody  = `{"id":"alloc","type":"gauge","value":42.0}`
		responseBody = `{"status":"ok"}`
	)
	logger := zap.NewNop()

	tests := []struct {
		name            string
		hashKey         string
		requestHashHdr  string
		wantStatus      int
		wantHandlerCall bool
		wantRespHash    bool
	}{
		{
			name:            "no key, no hash header - passes through, no hash in response",
			hashKey:         "",
			requestHashHdr:  "",
			wantStatus:      http.StatusOK,
			wantHandlerCall: true,
			wantRespHash:    false,
		},
		{
			name:            "no key, request has hash header - still passes through",
			hashKey:         "",
			requestHashHdr:  "anyhashvalue",
			wantStatus:      http.StatusOK,
			wantHandlerCall: true,
			wantRespHash:    false,
		},
		{
			name:            "key configured, no hash header - passes through, response carries hash",
			hashKey:         key,
			requestHashHdr:  "",
			wantStatus:      http.StatusOK,
			wantHandlerCall: true,
			wantRespHash:    true,
		},
		{
			name:            "key configured, correct hash - 200 and response carries hash",
			hashKey:         key,
			requestHashHdr:  hex.EncodeToString(utils.GenSHA256([]byte(requestBody), key)),
			wantStatus:      http.StatusOK,
			wantHandlerCall: true,
			wantRespHash:    true,
		},
		{
			name:            "key configured, wrong hash - 400 and handler not called",
			hashKey:         key,
			requestHashHdr:  "wronghashvalue",
			wantStatus:      http.StatusBadRequest,
			wantHandlerCall: false,
			wantRespHash:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var called atomic.Bool
			handler := WithHash(tt.hashKey, logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called.Store(true)
				_, _ = w.Write([]byte(responseBody))
			}))
			srv := httptest.NewServer(handler)
			defer srv.Close()

			req, err := http.NewRequest(http.MethodPost, srv.URL, bytes.NewBufferString(requestBody))
			require.NoError(t, err)
			if tt.requestHashHdr != "" {
				req.Header.Set(utils.HashHeaderName, tt.requestHashHdr)
			}

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			_, _ = io.Copy(io.Discard, resp.Body)

			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			assert.Equal(t, tt.wantHandlerCall, called.Load())

			if tt.wantRespHash {
				expected := hex.EncodeToString(utils.GenSHA256([]byte(responseBody), tt.hashKey))
				assert.Equal(t, expected, resp.Header.Get(utils.HashHeaderName))
			} else {
				assert.Empty(t, resp.Header.Get(utils.HashHeaderName))
			}
		})
	}
}
