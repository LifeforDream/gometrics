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
	PollInterval       int
	ReportInterval     int
	ServerAddr         string
	HashKey            string
	ConcurrentRequests int
}

type Agent struct {
	cfg Config
}

func New(cfg Config) *Agent {
	return &Agent{cfg: cfg}
}

func (a *Agent) Run(ctx context.Context, logger *zap.Logger) {
	c := make(chan map[string]AgentMetric, 1)
	go collect(ctx, a.cfg.PollInterval, c, logger)
	go send(ctx, logger, a.cfg.ReportInterval, c, a.cfg.ServerAddr, a.cfg.HashKey, a.cfg.ConcurrentRequests)
}
