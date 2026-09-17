package agent

import (
	"bytes"
	"compress/gzip"
	"crypto/rsa"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LifeforDream/gometrics/internal/crypto"
	models "github.com/LifeforDream/gometrics/internal/model"
	"github.com/LifeforDream/gometrics/internal/utils"
	"github.com/LifeforDream/gometrics/pkg/certgen"
)

func TestSendMetricBatch(t *testing.T) {
	tests := []struct {
		name        string
		metrics     map[string]agentMetric
		wantPayload []models.Metrics
		hashKey     string
		cryptoPub   *rsa.PublicKey
		wantErr     bool
		hitsServer  bool
	}{
		{
			name: "success with gauge and counter",
			metrics: map[string]agentMetric{
				"alloc":     {Type: models.Gauge, Value: 1.25},
				"pollcount": {Type: models.Counter, Value: 3},
			},
			wantPayload: []models.Metrics{
				{ID: "alloc", MType: models.Gauge, Value: new(1.25)},
				{ID: "pollcount", MType: models.Counter, Delta: new(int64(3))},
			},
			hashKey:    "",
			hitsServer: true,
		},
		{
			name:       "empty batch makes no HTTP call",
			metrics:    map[string]agentMetric{},
			hitsServer: false,
		},
		{
			name:       "unknown metric type returns error",
			metrics:    map[string]agentMetric{"bad": {Type: "invalid", Value: 1}},
			wantErr:    true,
			hitsServer: false,
		},
		{
			name: "setting hash key sets header",
			metrics: map[string]agentMetric{
				"alloc":     {Type: models.Gauge, Value: 1.25},
				"pollcount": {Type: models.Counter, Value: 3},
			},
			wantPayload: []models.Metrics{
				{ID: "alloc", MType: models.Gauge, Value: new(1.25)},
				{ID: "pollcount", MType: models.Counter, Delta: new(int64(3))},
			},
			hashKey:    "somekey",
			hitsServer: true,
		},
	}
	client := &http.Client{}
	bodyCh := make(chan []byte, 1)
	hashHeaderCh := make(chan string, 1)
	rawBodyCh := make(chan []byte, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "gzip", r.Header.Get("Content-Encoding"))
		hashHeaderCh <- r.Header.Get(utils.HashHeaderName)
		rawBody, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		rawBodyCh <- rawBody

		gr, err := gzip.NewReader(bytes.NewReader(rawBody))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer gr.Close()
		body, err := io.ReadAll(gr)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		bodyCh <- body
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := sendMetricBatch(tt.metrics, client, SendParams{
				serverAddress: server.URL,
				hashKey:       tt.hashKey,
				publicKey:     tt.cryptoPub,
			})
			if tt.wantErr {
				assert.Error(t, gotErr)
			} else {
				assert.NoError(t, gotErr)
			}
			if tt.hitsServer {
				var got []models.Metrics
				body := <-bodyCh
				require.NoError(t, json.Unmarshal(body, &got))
				assert.ElementsMatch(t, tt.wantPayload, got)

				hashHeader := <-hashHeaderCh
				rawBody := <-rawBodyCh
				if tt.hashKey != "" {
					assert.Equal(t, hex.EncodeToString(utils.GenSHA256(rawBody, tt.hashKey)), hashHeader)
				} else {
					assert.Empty(t, hashHeader)
				}
			}
		})
	}
}

// TestSendMetricBatchEncryptsWhenCryptoKeyConfigured проверяет, что при
// заданном публичном ключе sendMetricBatch шифрует уже сжатое gzip'ом тело
// перед отправкой, и что зашифрованные байты действительно расшифровываются
// обратно в исходный JSON парным приватным ключом.
func TestSendMetricBatchEncryptsWhenCryptoKeyConfigured(t *testing.T) {
	certPEM, privPEM, err := certgen.GenerateKeyPair(1, 1024)
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

	metrics := map[string]agentMetric{
		"alloc":     {Type: models.Gauge, Value: 1.25},
		"pollcount": {Type: models.Counter, Value: 3},
	}
	wantPayload := []models.Metrics{
		{ID: "alloc", MType: models.Gauge, Value: new(1.25)},
		{ID: "pollcount", MType: models.Counter, Delta: new(int64(3))},
	}

	rawBodyCh := make(chan []byte, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawBody, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		rawBodyCh <- rawBody
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &http.Client{}
	err = sendMetricBatch(metrics, client, SendParams{
		serverAddress: server.URL,
		publicKey:     pub,
	})
	require.NoError(t, err)

	rawBody := <-rawBodyCh

	gzipBytes, err := crypto.Decrypt(priv, rawBody)
	require.NoError(t, err)
	assert.NotEqual(t, rawBody, gzipBytes, "raw body on the wire must be ciphertext, not the plain gzip bytes")

	gr, err := gzip.NewReader(bytes.NewReader(gzipBytes))
	require.NoError(t, err)
	defer gr.Close()
	body, err := io.ReadAll(gr)
	require.NoError(t, err)

	var got []models.Metrics
	require.NoError(t, json.Unmarshal(body, &got))
	assert.ElementsMatch(t, wantPayload, got)
}

// TestSendMetricBatchWithoutCryptoKeyIsUnchanged проверяет, что при пустом
// публичном ключе тело остаётся обычным gzip'ом JSON без шифрования.
func TestSendMetricBatchWithoutCryptoKeyIsUnchanged(t *testing.T) {
	metrics := map[string]agentMetric{
		"alloc": {Type: models.Gauge, Value: 1.25},
	}

	rawBodyCh := make(chan []byte, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawBody, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		rawBodyCh <- rawBody
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &http.Client{}
	err := sendMetricBatch(metrics, client, SendParams{
		serverAddress: server.URL,
	})
	require.NoError(t, err)

	rawBody := <-rawBodyCh

	gr, err := gzip.NewReader(bytes.NewReader(rawBody))
	require.NoError(t, err, "body must still be plain gzip when no crypto key is configured")
	defer gr.Close()
	body, err := io.ReadAll(gr)
	require.NoError(t, err)

	var got []models.Metrics
	require.NoError(t, json.Unmarshal(body, &got))
	assert.ElementsMatch(t, []models.Metrics{{ID: "alloc", MType: models.Gauge, Value: new(1.25)}}, got)
}
