package router

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

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/LifeforDream/gometrics/internal/audit"
	"github.com/LifeforDream/gometrics/internal/crypto"
	"github.com/LifeforDream/gometrics/internal/handler"
	"github.com/LifeforDream/gometrics/internal/middlewares/logs"
	"github.com/LifeforDream/gometrics/internal/middlewares/mwcompress"
	"github.com/LifeforDream/gometrics/internal/middlewares/mwcrypto"
	"github.com/LifeforDream/gometrics/internal/middlewares/mwhash"
	"github.com/LifeforDream/gometrics/internal/middlewares/mwip"
	models "github.com/LifeforDream/gometrics/internal/model"
	"github.com/LifeforDream/gometrics/internal/repository"
	"github.com/LifeforDream/gometrics/internal/service"
	"github.com/LifeforDream/gometrics/internal/utils"
	"github.com/LifeforDream/gometrics/pkg/certgen"
)

// TestUpdatesEndToEndWithHashAndCrypto закрепляет порядок подключения
// мидлваров, который должен использоваться в cmd/server/main.go:
// logs -> mwip -> mwhash -> mwcrypto -> StripSlashes -> mwcompress -> handler.
// Хэш проверяется на байтах, полученных с провода (до расшифровки),
// расшифровка происходит до разжатия gzip.
func TestUpdatesEndToEndWithHashAndCrypto(t *testing.T) {
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

	r := MetricsRouter(h,
		logs.WithLogging(logger),
		mwip.WithClientIP,
		mwhash.WithHash(hashKey, logger),
		mwcrypto.WithCrypto(priv, logger),
		chimiddleware.StripSlashes,
		mwcompress.Compress(logger),
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
