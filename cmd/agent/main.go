package main

import (
	agent "github.com/LifeforDream/gometrics/internal/agent"
)

func main() {
	parseFlags()
	cfg := agent.Config{
		PollInterval:   agentOptions.pollInterval,
		ReportInterval: agentOptions.reportInterval,
		ServerAddr:     agentOptions.schema + agentOptions.address,
	}
	a := agent.New(cfg)
	a.Run()
	select {}
}
