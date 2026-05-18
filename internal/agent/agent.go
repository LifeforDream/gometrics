package agent

import (
	"time"
)

type AgentMetric struct {
	Type  string
	Value float64
}

type Config struct {
	PollInterval   time.Duration
	ReportInterval time.Duration
	ServerAddr     string
}

type Agent struct {
	cfg Config
}

func New(cfg Config) *Agent {
	return &Agent{cfg: cfg}
}

func (a *Agent) Run() {
	c := make(chan map[string]AgentMetric, 1)
	go collect(a.cfg.PollInterval, c)
	go send(a.cfg.ReportInterval, c, a.cfg.ServerAddr)
}
