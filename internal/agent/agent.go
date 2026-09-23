// Package agent реализует агент сбора и отправки метрик: горутина collect
// снимает runtime- и системные метрики, горутина send отправляет их батчами
// на сервер.
package agent

import (
	"context"
	"crypto/rsa"
	"net/http"
	"sync"

	"go.uber.org/zap"
)

// agentMetric — одна собранная метрика перед отправкой на сервер.
type agentMetric struct {
	Type  string
	Value float64
}

// интерфейс для клиента отправки запросов
type httpSender interface {
	Do(req *http.Request) (*http.Response, error)
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
	Client             httpSender // клиент для отправки запросов, может быть подменён в тестах.
}

// Agent запускает сбор и отправку метрик согласно переданному Config.
type Agent struct {
	cfg Config
	wg  sync.WaitGroup
}

// New создаёт Agent с переданной конфигурацией.
func New(cfg Config) *Agent {
	if cfg.Client == nil {
		cfg.Client = newRetryableClient(3, 5)
	}
	return &Agent{cfg: cfg}
}

// Run запускает горутины collect и send, связанные каналом с буфером 1:
//
// collect пишет снятые метрики,
//
// send читает и отправляет их батчами.
//
// Обе горутины завершаются по ctx.Done().
func (a *Agent) Run(ctx context.Context, logger *zap.Logger) {
	c := make(chan map[string]agentMetric, 1)
	a.wg.Add(2)
	go func() { defer a.wg.Done(); collect(ctx, a.cfg.PollInterval, c, logger) }()
	go func() {
		defer a.wg.Done()
		send(ctx, SendParams{
			logger:             logger,
			interval:           a.cfg.ReportInterval,
			metricsChannel:     c,
			serverAddress:      a.cfg.ServerAddr,
			hashKey:            a.cfg.HashKey,
			concurrentRequests: a.cfg.ConcurrentRequests,
			publicKey:          a.cfg.PublicKey,
			client:             a.cfg.Client,
		})
	}()
}

// Wait ожидает завершения дочерних горутин агента.
// Позволяет гарантированно дождаться завершения всех процессов при
// graceful shutdown.
func (a *Agent) Wait() {
	a.wg.Wait()
}
