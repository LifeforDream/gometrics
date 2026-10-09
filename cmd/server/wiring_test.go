package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/hex"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/LifeforDream/gometrics/internal/audit"
	"github.com/LifeforDream/gometrics/internal/crypto"
	"github.com/LifeforDream/gometrics/internal/handler"
	models "github.com/LifeforDream/gometrics/internal/model"
	pb "github.com/LifeforDream/gometrics/internal/proto"
	"github.com/LifeforDream/gometrics/internal/repository"
	"github.com/LifeforDream/gometrics/internal/router"
	"github.com/LifeforDream/gometrics/internal/service"
	"github.com/LifeforDream/gometrics/internal/utils"
	"github.com/LifeforDream/gometrics/pkg/certgen"
)

type fakeMetricsServer struct {
	pb.UnimplementedMetricsServer
	c chan struct{}
}

func newFakeMetricsServer(c chan struct{}) *fakeMetricsServer {
	return &fakeMetricsServer{
		c: c,
	}
}

func (s *fakeMetricsServer) UpdateMetrics(ctx context.Context, req *pb.UpdateMetricsRequest) (*pb.UpdateMetricsResponse, error) {
	s.c <- struct{}{}
	// block until done
	<-ctx.Done()
	return &pb.UpdateMetricsResponse{}, nil
}

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

func TestShutdownServersNilGrpc(t *testing.T) {
	logger := zap.NewNop()
	repo := repository.NewMemStorage()
	svc := service.NewMetricService(repo, audit.NewAuditor(logger))
	h := handler.NewHandler(svc, logger)
	r := router.MetricsRouter(h,
		router.NewChain(),
		router.NewChain(),
		router.NewChain(),
	)

	httpSrv := &http.Server{
		Addr:    ":0",
		Handler: r,
	}

	err := shutdownServers(context.Background(), httpSrv, nil)
	require.NoError(t, err)

}

func TestShutdownServersBoth(t *testing.T) {
	logger := zap.NewNop()
	repo := repository.NewMemStorage()
	svc := service.NewMetricService(repo, audit.NewAuditor(logger))
	h := handler.NewHandler(svc, logger)
	r := router.MetricsRouter(h,
		router.NewChain(),
		router.NewChain(),
		router.NewChain(),
	)

	httpSrv := &http.Server{
		Addr:    ":0",
		Handler: r,
	}

	lis, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	grpcSrv := grpc.NewServer()
	pb.RegisterMetricsServer(grpcSrv, pb.UnimplementedMetricsServer{})

	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- grpcSrv.Serve(lis)
	}()

	// вызываем сервер, чтобы убедиться, что он стартанул,
	// в противном случае shutdownServers может завершиться раньше, чем grpcSrv.Serve стартовал.
	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	client := pb.NewMetricsClient(conn)
	_, err = client.UpdateMetrics(context.Background(), &pb.UpdateMetricsRequest{})
	require.Error(t, err) // code = Unimplemented

	err = shutdownServers(context.Background(), httpSrv, grpcSrv)
	require.NoError(t, err)

	select {
	case serveErr := <-serveErrCh:
		require.NoError(t, serveErr) // Serve возвращает nil после Stop/GracefulStop
	case <-time.After(20 * time.Millisecond):
		t.Fatal("grpcSrv.Serve did not return after shutdownServers completed")
	}
}

func TestShutdownServersBlockingGrpc(t *testing.T) {
	logger := zap.NewNop()
	repo := repository.NewMemStorage()
	svc := service.NewMetricService(repo, audit.NewAuditor(logger))
	h := handler.NewHandler(svc, logger)
	r := router.MetricsRouter(h,
		router.NewChain(),
		router.NewChain(),
		router.NewChain(),
	)

	httpSrv := &http.Server{
		Addr:    ":0",
		Handler: r,
	}

	lis, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	grpcSrv := grpc.NewServer()
	callbackChan := make(chan struct{})
	pb.RegisterMetricsServer(grpcSrv, newFakeMetricsServer(callbackChan))

	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- grpcSrv.Serve(lis)
	}()

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	client := pb.NewMetricsClient(conn)

	updateMetricsErrCh := make(chan error, 1)
	go func() {
		_, err = client.UpdateMetrics(context.Background(), &pb.UpdateMetricsRequest{})
		updateMetricsErrCh <- err
	}()

	<-callbackChan

	timedCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	resultChan := make(chan error, 1)
	go func() {
		resultChan <- shutdownServers(timedCtx, httpSrv, grpcSrv)
	}()

	select {
	case err := <-resultChan:
		require.NoError(t, err)
	case <-time.After(20 * time.Millisecond):
		t.Fatal("shutdownServers did not return before context deadline")
	}

	select {
	case serveErr := <-serveErrCh:
		require.NoError(t, serveErr) // Serve возвращает nil после Stop/GracefulStop
	case <-time.After(20 * time.Millisecond):
		t.Fatal("grpcSrv.Serve did not return after shutdownServers completed")
	}

	select {
	case updateMetricsErr := <-updateMetricsErrCh:
		require.Error(t, updateMetricsErr)
	case <-time.After(20 * time.Millisecond):
		t.Fatal("client.UpdateMetrics did not return after shutdownServers completed")
	}
}
