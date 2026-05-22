package main

import (
	"log"
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

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	a.Run()

	// Block until a signal is received.
	s := <-c
	log.Printf("Stopping metrics agent: %s", s)
}
