package mwcrypto

import (
	"bytes"
	"crypto/rsa"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/LifeforDream/gometrics/internal/crypto"
	"github.com/LifeforDream/gometrics/pkg/certgen"
)

const (
	testKeyBits = 1024
	testCertID  = 1
)

// generateTestKeys генерирует пару ключей через certgen.GenerateKeyPair и
// загружает их обратно через internal/crypto, как это будет делать
// cmd/server/main.go при старте.
func generateTestKeys(t *testing.T, bits int) (*rsa.PublicKey, *rsa.PrivateKey) {
	t.Helper()
	certPEM, privPEM, err := certgen.GenerateKeyPair(testCertID, bits)
	require.NoError(t, err)

	dir := t.TempDir()
	certPath := filepath.Join(dir, "cert.pem")
	privPath := filepath.Join(dir, "private.pem")
	require.NoError(t, os.WriteFile(certPath, certPEM, 0o600))
	require.NoError(t, os.WriteFile(privPath, privPEM, 0o600))

	pub, err := crypto.LoadPublicKey(certPath)
	require.NoError(t, err)
	priv, err := crypto.LoadPrivateKey(privPath)
	require.NoError(t, err)
	return pub, priv
}

func TestWithCrypto(t *testing.T) {
	logger := zap.NewNop()
	pub, priv := generateTestKeys(t, testKeyBits)
	_, otherPriv := generateTestKeys(t, testKeyBits)

	const plaintext = `{"id":"alloc","type":"gauge","value":42.0}`

	tests := []struct {
		name            string
		privateKey      *rsa.PrivateKey
		body            func(t *testing.T) []byte
		wantStatus      int
		wantHandlerCall bool
	}{
		{
			name:       "no private key configured - passthrough",
			privateKey: nil,
			body: func(t *testing.T) []byte {
				return []byte(plaintext)
			},
			wantStatus:      http.StatusOK,
			wantHandlerCall: true,
		},
		{
			name:       "valid encrypted body - handler receives plaintext",
			privateKey: priv,
			body: func(t *testing.T) []byte {
				ciphertext, err := crypto.Encrypt(pub, []byte(plaintext))
				require.NoError(t, err)
				return ciphertext
			},
			wantStatus:      http.StatusOK,
			wantHandlerCall: true,
		},
		{
			name:       "garbage body with key configured - 400",
			privateKey: priv,
			body: func(t *testing.T) []byte {
				return []byte("not a valid ciphertext at all, just plain text")
			},
			wantStatus:      http.StatusBadRequest,
			wantHandlerCall: false,
		},
		{
			name:       "body encrypted with a different key - 400",
			privateKey: priv,
			body: func(t *testing.T) []byte {
				otherPub := &otherPriv.PublicKey
				ciphertext, err := crypto.Encrypt(otherPub, []byte(plaintext))
				require.NoError(t, err)
				return ciphertext
			},
			wantStatus:      http.StatusBadRequest,
			wantHandlerCall: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var called atomic.Bool
			var gotBody []byte
			handler := WithCrypto(tt.privateKey, logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called.Store(true)
				b, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				gotBody = b
				w.WriteHeader(http.StatusOK)
			}))
			srv := httptest.NewServer(handler)
			defer srv.Close()

			req, err := http.NewRequest(http.MethodPost, srv.URL, bytes.NewReader(tt.body(t)))
			require.NoError(t, err)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			_, _ = io.Copy(io.Discard, resp.Body)

			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			assert.Equal(t, tt.wantHandlerCall, called.Load())
			if tt.wantHandlerCall {
				assert.Equal(t, plaintext, string(gotBody))
			}
		})
	}
}
