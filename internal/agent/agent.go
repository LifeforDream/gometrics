package agent

import (
	"context"

	"go.uber.org/zap"
)

type AgentMetric struct {
	Type  string
	Value float64
}

type Config struct {
	PollInterval   int
	ReportInterval int
	ServerAddr     string
}

type Agent struct {
	cfg Config
}

func New(cfg Config) *Agent {
	return &Agent{cfg: cfg}
}

func (a *Agent) Run(ctx context.Context, logger *zap.Logger) {
	c := make(chan map[string]AgentMetric, 1)
	go collect(ctx, a.cfg.PollInterval, c)
	go send(ctx, a.cfg.ReportInterval, c, a.cfg.ServerAddr, logger)
}
