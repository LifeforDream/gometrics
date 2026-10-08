// Package agent реализует агент сбора и отправки метрик: горутина collect
// снимает runtime- и системные метрики, горутина send отправляет их батчами
// на сервер.
package agent

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net/http"
	"sync"

	"go.uber.org/zap"
	"google.golang.org/grpc/credentials"
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

// Config — настройки агента.
type Config struct {
	PollInterval       int                              // интервал опроса метрик.
	ReportInterval     int                              // интервал отправки метрик.
	ServerAddr         string                           // адрес сервера для отправки метрик.
	HashKey            string                           // ключ для хэширования тела запроса.
	ConcurrentRequests int                              // максимальное количество одновременно отправляемых запросов.
	PublicKey          *rsa.PublicKey                   // публичный ключ для шифрования тела запросов.
	Client             httpSender                       // клиент для отправки запросов, может быть подменён в тестах.
	HostIP             string                           // исходящий IP-адрес хоста, на котором запущен агент.
	GRPCAddr           string                           // адрес для отправки метрик по gRPC.
	GRPCCreds          credentials.TransportCredentials // креды для отправки метрик по gRPC.
}

// Agent запускает сбор и отправку метрик согласно переданному Config.
type Agent struct {
	cfg    Config
	wg     sync.WaitGroup
	sender batchSender
}

// New создаёт Agent с переданной конфигурацией.
func New(cfg Config) (*Agent, error) {
	if cfg.Client == nil {
		cfg.Client = newRetryableClient(3, 5)
	}
	var sender batchSender
	if cfg.GRPCAddr != "" {
		grpcsender, err := newGRPCBatchSender(cfg.GRPCAddr, cfg.HostIP, cfg.GRPCCreds)
		if err != nil {
			return nil, fmt.Errorf("error creating agent with grpc: %w", err)
		}
		sender = grpcsender
	} else {
		sender = &httpBatchSender{
			serverAddress: cfg.ServerAddr,
			hashKey:       cfg.HashKey,
			publicKey:     cfg.PublicKey,
			client:        cfg.Client,
			hostIP:        cfg.HostIP,
		}
	}
	return &Agent{cfg: cfg, sender: sender}, nil
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
			concurrentRequests: a.cfg.ConcurrentRequests,
			sender:             a.sender,
		})
	}()
}

// Wait ожидает завершения дочерних горутин агента.
// Позволяет гарантированно дождаться завершения всех процессов при
// graceful shutdown.
func (a *Agent) Wait() {
	a.wg.Wait()
}

func (a *Agent) Close() error {
	return a.sender.Close()
}
