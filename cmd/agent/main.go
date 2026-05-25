package main

import (
	"context"
	"os"
	"os/signal"

	agent "github.com/LifeforDream/gometrics/internal/agent"
)

func main() {
	parseFlags()

	serverAddr := constructAddress()

	cfg := agent.Config{
		PollInterval:   agentOptions.pollInterval,
		ReportInterval: agentOptions.reportInterval,
		ServerAddr:     serverAddr,
	}
	a := agent.New(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	a.Run(ctx)

	<-ctx.Done()
}
