package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	agent "github.com/LifeforDream/gometrics/internal/agent"
)

func main() {
	agentOptions, err := parseOptions()
	if err != nil {
		log.Fatal(err)
	}

	serverAddr := constructAddress(agentOptions)

	cfg := agent.Config{
		PollInterval:   agentOptions.PollInterval,
		ReportInterval: agentOptions.ReportInterval,
		ServerAddr:     serverAddr,
	}
	a := agent.New(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	a.Run(ctx)

	<-ctx.Done()
}
