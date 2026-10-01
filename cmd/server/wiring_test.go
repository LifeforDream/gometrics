package main

import (
	"bytes"
	"compress/gzip"
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
	"go.uber.org/zap"

	"github.com/LifeforDream/gometrics/internal/audit"
	"github.com/LifeforDream/gometrics/internal/crypto"
	"github.com/LifeforDream/gometrics/internal/handler"
	models "github.com/LifeforDream/gometrics/internal/model"
	"github.com/LifeforDream/gometrics/internal/repository"
	"github.com/LifeforDream/gometrics/internal/router"
	"github.com/LifeforDream/gometrics/internal/service"
	"github.com/LifeforDream/gometrics/internal/utils"
	"github.com/LifeforDream/gometrics/pkg/certgen"
)

// TestServerChainsHashCryptoCompress прогоняет запрос через ту же сборку
// мидлваров, что и main (newMiddlewares + buildChains): хэш проверяется на
// байтах, полученных с провода (до расшифровки), расшифровка происходит до
// разжатия gzip.
func TestServerChainsHashCryptoCompress(t *testing.T) {
	logger := zap.NewNop()
	const hashKey = "secretkey"

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

	repo := repository.NewMemStorage()
	svc := service.NewMetricService(repo, audit.NewAuditor(logger))
	h := handler.NewHandler(svc, logger)

	core, readMws, writeMws := buildChains(newMiddlewares(hashKey, priv, nil, logger))
	r := router.MetricsRouter(h,
		router.NewChain(core...),
		router.NewChain(readMws...),
		router.NewChain(writeMws...),
	)
	srv := httptest.NewServer(r)
	defer srv.Close()

	payload := []models.Metrics{
		{ID: "alloc", MType: models.Gauge, Value: new(42.5)},
		{ID: "pollcount", MType: models.Counter, Delta: new(int64(7))},
	}

	gzipJSON := func(t *testing.T, payload []models.Metrics) []byte {
		t.Helper()
		body, err := json.Marshal(payload)
		require.NoError(t, err)

		var gz bytes.Buffer
		zw := gzip.NewWriter(&gz)
		_, err = zw.Write(body)
		require.NoError(t, err)
		require.NoError(t, zw.Close())
		return gz.Bytes()
	}

	postUpdates := func(t *testing.T, body []byte) *http.Response {
		t.Helper()
		req, err := http.NewRequest(http.MethodPost, srv.URL+"/updates", bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(utils.HashHeaderName, hex.EncodeToString(utils.GenSHA256(body, hashKey)))

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		t.Cleanup(func() {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		})
		return resp
	}

	getMetricValue := func(t *testing.T, mtype, name string) (int, string) {
		t.Helper()
		resp, err := http.Get(srv.URL + "/value/" + mtype + "/" + name)
		require.NoError(t, err)
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		return resp.StatusCode, string(b)
	}

	t.Run("encrypted, gzipped, hashed body round-trips end to end", func(t *testing.T) {
		gz := gzipJSON(t, payload)
		ciphertext, err := crypto.Encrypt(pub, gz)
		require.NoError(t, err)

		resp := postUpdates(t, ciphertext)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		status, body := getMetricValue(t, "gauge", "alloc")
		assert.Equal(t, http.StatusOK, status)
		assert.Equal(t, "42.5", body)

		status, body = getMetricValue(t, "counter", "pollcount")
		assert.Equal(t, http.StatusOK, status)
		assert.Equal(t, "7", body)
	})

	t.Run("plaintext gzip body sent when server expects encryption - rejected", func(t *testing.T) {
		gz := gzipJSON(t, payload)

		resp := postUpdates(t, gz)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
