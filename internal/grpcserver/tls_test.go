package grpcserver

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/LifeforDream/gometrics/internal/proto"
	"github.com/LifeforDream/gometrics/pkg/certgen"
)

// startTestServer поднимает настоящий grpc.Server на случайном порту
// localhost с переданными creds и останавливает его по завершении теста.
func startTestServer(t *testing.T, creds credentials.TransportCredentials) string {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	srv := New(&fakeMetricUpdater{}, nil, zap.NewNop(), creds)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	return lis.Addr().String()
}

// dialAndUpdateMetrics подключается к addr с переданными creds и вызывает
// UpdateMetrics с пустым запросом, возвращая итоговую ошибку (транспортную
// или от самого RPC).
func dialAndUpdateMetrics(t *testing.T, addr string, creds credentials.TransportCredentials) error {
	t.Helper()

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(creds))
	require.NoError(t, err)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = pb.NewMetricsClient(conn).UpdateMetrics(ctx, pb.UpdateMetricsRequest_builder{}.Build())
	return err
}

// certPool строит x509.CertPool из PEM-байтов одного сертификата.
func certPool(t *testing.T, certPEM []byte) *x509.CertPool {
	t.Helper()
	pool := x509.NewCertPool()
	require.True(t, pool.AppendCertsFromPEM(certPEM))
	return pool
}

func TestGRPCServerTLS(t *testing.T) {
	certPEM, keyPEM, err := certgen.GenerateKeyPair(1, 2048)
	require.NoError(t, err)
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	require.NoError(t, err)

	serverCreds := credentials.NewTLS(&tls.Config{Certificates: []tls.Certificate{cert}})

	t.Run("client trusting the server certificate succeeds", func(t *testing.T) {
		addr := startTestServer(t, serverCreds)

		clientCreds := credentials.NewTLS(&tls.Config{RootCAs: certPool(t, certPEM)})
		err := dialAndUpdateMetrics(t, addr, clientCreds)

		assert.NoError(t, err)
	})

	t.Run("client with an unrelated trust root is rejected", func(t *testing.T) {
		addr := startTestServer(t, serverCreds)

		otherCertPEM, _, err := certgen.GenerateKeyPair(2, 2048)
		require.NoError(t, err)
		clientCreds := credentials.NewTLS(&tls.Config{RootCAs: certPool(t, otherCertPEM)})

		err = dialAndUpdateMetrics(t, addr, clientCreds)

		require.Error(t, err)
	})

	t.Run("plaintext client against a TLS-only server is rejected", func(t *testing.T) {
		addr := startTestServer(t, serverCreds)

		err := dialAndUpdateMetrics(t, addr, insecure.NewCredentials())

		require.Error(t, err)
	})

	t.Run("no creds configured keeps the server plaintext", func(t *testing.T) {
		addr := startTestServer(t, nil)

		err := dialAndUpdateMetrics(t, addr, insecure.NewCredentials())

		assert.NoError(t, err, "New(..., nil) must default to plaintext")
	})
}
