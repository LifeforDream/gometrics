// Package main является точкой запуска для агента сбора и отправки метрик.
package main

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"google.golang.org/grpc/credentials"

	agent "github.com/LifeforDream/gometrics/internal/agent"
	"github.com/LifeforDream/gometrics/internal/buildinfo"
	"github.com/LifeforDream/gometrics/internal/crypto"
	"github.com/LifeforDream/gometrics/internal/logging"
)

var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)

func main() {
	buildinfo.Print(os.Stdout, buildVersion, buildDate, buildCommit)

	agentOptions, err := parseOptions()
	if err != nil {
		log.Fatal(err)
	}

	serverAddr := constructAddress(agentOptions)

	logger, err := logging.Initialize("info")
	if err != nil {
		log.Fatal(err)
	}

	var publicKey *rsa.PublicKey
	if agentOptions.CryptoKeyPath != "" {
		publicKey, err = crypto.LoadPublicKey(agentOptions.CryptoKeyPath)
		if err != nil {
			logger.Fatal("Error loading public key with configured path", zap.String("crypto-path", agentOptions.CryptoKeyPath), zap.Error(err))
		}
	}

	var grpcCreds credentials.TransportCredentials
	if agentOptions.GRPCAddr != "" && agentOptions.CryptoKeyPath != "" {
		certPEM, err := os.ReadFile(agentOptions.CryptoKeyPath)
		if err != nil {
			logger.Fatal("Error reading certificate for gRPC", zap.String("crypto-path", agentOptions.CryptoKeyPath), zap.Error(err))
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(certPEM) {
			logger.Fatal("Error appending certificate to pool for gRPC", zap.String("crypto-path", agentOptions.CryptoKeyPath))
		}
		grpcCreds = credentials.NewClientTLSFromCert(pool, "")
	}

	var hostIP net.IP
	if agentOptions.GRPCAddr != "" {
		hostIP, err = agent.OutboundIPHostPort(agentOptions.GRPCAddr)
	} else {
		hostIP, err = agent.OutboundIP(serverAddr)
	}
	if err != nil {
		logger.Fatal("error getting outbound IP address", zap.Error(err))
	}

	cfg := agent.Config{
		PollInterval:       agentOptions.PollInterval,
		ReportInterval:     agentOptions.ReportInterval,
		ServerAddr:         serverAddr,
		HashKey:            agentOptions.HashKey,
		ConcurrentRequests: agentOptions.ConcurrentRequests,
		PublicKey:          publicKey,
		HostIP:             hostIP.String(),
		GRPCAddr:           agentOptions.GRPCAddr,
		GRPCCreds:          grpcCreds,
	}

	a, err := agent.New(cfg)
	if err != nil {
		logger.Fatal("error starting agent", zap.Error(err))
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	a.Run(ctx, logger)

	<-ctx.Done()

	a.Wait()

	if err := a.Close(); err != nil {
		logger.Error("error closing agent", zap.Error(err))
	}
}
