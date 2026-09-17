// Package agent реализует агент сбора и отправки метрик: горутина collect
// снимает runtime- и системные метрики, горутина send отправляет их батчами
// на сервер.
package agent

import (
	"context"
	"crypto/rsa"

	"go.uber.org/zap"
)

// agentMetric — одна собранная метрика перед отправкой на сервер.
type agentMetric struct {
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
	PublicKey          *rsa.PublicKey
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
	c := make(chan map[string]agentMetric, 1)
	go collect(ctx, a.cfg.PollInterval, c, logger)
	go send(ctx, SendParams{
		logger:        logger,
		interval:      a.cfg.ReportInterval,
		c:             c,
		serverAddress: a.cfg.ServerAddr,
		hashKey:       a.cfg.HashKey,
		concreqs:      a.cfg.ConcurrentRequests,
		publicKey:     a.cfg.PublicKey,
	})
}
