package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	agent "github.com/LifeforDream/gometrics/internal/agent"
	"github.com/LifeforDream/gometrics/internal/logging"
)

func main() {
	agentOptions, err := parseOptions()
	if err != nil {
		log.Fatal(err)
	}

	serverAddr := constructAddress(agentOptions)

	logger, err := logging.Initialize("info")
	if err != nil {
		log.Fatal(err)
	}

	cfg := agent.Config{
		PollInterval:       agentOptions.PollInterval,
		ReportInterval:     agentOptions.ReportInterval,
		ServerAddr:         serverAddr,
		HashKey:            agentOptions.HashKey,
		ConcurrentRequests: agentOptions.ConcurrentRequests,
	}

	a := agent.New(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	a.Run(ctx, logger)

	<-ctx.Done()
}
