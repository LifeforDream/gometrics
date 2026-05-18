package main

import (
	"flag"
	"time"

	agent "github.com/LifeforDream/gometrics/internal/agent"
)

func main() {
	pollInterval := flag.Duration("p", 2*time.Second, "poll interval")
	reportInterval := flag.Duration("r", 10*time.Second, "report interval")
	serverAddr := flag.String("a", "http://localhost:8080", "server address")

	flag.Parse()

	cfg := agent.Config{
		PollInterval:   *pollInterval,
		ReportInterval: *reportInterval,
		ServerAddr:     *serverAddr,
	}
	a := agent.New(cfg)
	a.Run()
	select {}
}
