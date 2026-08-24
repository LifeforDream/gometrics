// Package agent реализует агент сбора и отправки метрик: горутина collect
// снимает runtime- и системные метрики, горутина send отправляет их батчами
// на сервер.
package agent

import (
	"context"

	"go.uber.org/zap"
)

// AgentMetric — одна собранная метрика перед отправкой на сервер.
type AgentMetric struct {
	Type  string
	Value float64
}

// Config — настройки агента: интервалы опроса и отправки, адрес сервера,
// ключ подписи тела запроса и число одновременных запросов к серверу.
type Config struct {
	PollInterval       int
	ReportInterval     int
	ServerAddr         string
	HashKey            string
	ConcurrentRequests int
}

// Agent запускает сбор и отправку метрик согласно переданной Config.
type Agent struct {
	cfg Config
}

// New создаёт Agent с переданной конфигурацией.
func New(cfg Config) *Agent {
	return &Agent{cfg: cfg}
}

// Run запускает горутины collect и send, связанные каналом с буфером 1:
// collect пишет снятые метрики, send читает и отправляет их батчами.
// Обе горутины завершаются по ctx.Done().
func (a *Agent) Run(ctx context.Context, logger *zap.Logger) {
	c := make(chan map[string]AgentMetric, 1)
	go collect(ctx, a.cfg.PollInterval, c, logger)
	go send(ctx, logger, a.cfg.ReportInterval, c, a.cfg.ServerAddr, a.cfg.HashKey, a.cfg.ConcurrentRequests)
}
