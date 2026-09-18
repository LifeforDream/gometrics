// Package main является точкой запуска для агента сбора и отправки метрик.
package main

import (
	"context"
	"crypto/rsa"
	"log"
	"os"
	"os/signal"

	"go.uber.org/zap"

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

	cfg := agent.Config{
		PollInterval:       agentOptions.PollInterval,
		ReportInterval:     agentOptions.ReportInterval,
		ServerAddr:         serverAddr,
		HashKey:            agentOptions.HashKey,
		ConcurrentRequests: agentOptions.ConcurrentRequests,
		PublicKey:          publicKey,
	}

	a := agent.New(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	a.Run(ctx, logger)

	<-ctx.Done()
}
