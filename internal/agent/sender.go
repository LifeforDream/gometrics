package agent

import (
	"context"
	"fmt"
	"maps"
	"sync"
	"time"

	"go.uber.org/zap"

	models "github.com/LifeforDream/gometrics/internal/model"
)

// sendTimeout ограничивает отправку одного батча метрик.
const sendTimeout = 30 * time.Second

// SendParams хранит параметры для запуска функции send() агента
type SendParams struct {
	logger             *zap.Logger
	interval           int
	metricsChannel     chan map[string]agentMetric
	concurrentRequests int
	sender             batchSender
}

type batchSender interface {
	Send(context.Context, []models.Metrics) error
	Close() error
}

type metricHolder struct {
	m  map[string]agentMetric
	mu sync.RWMutex
}

func newMetricHolder() *metricHolder {
	return &metricHolder{
		m: make(map[string]agentMetric),
	}
}

func (mm *metricHolder) Store(nm map[string]agentMetric) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	mm.m = nm
}

func (mm *metricHolder) Load() map[string]agentMetric {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	return mm.m
}

func send(ctx context.Context, params SendParams) {
	metricsHolder := newMetricHolder()
	lastSentMetrics := make(map[string]agentMetric)

	ticker := time.NewTicker(time.Duration(params.interval) * time.Second)
	defer ticker.Stop()

	workChan := make(chan map[string]agentMetric, params.concurrentRequests)

	var workersWg sync.WaitGroup
	workersWg.Add(params.concurrentRequests)
	for range params.concurrentRequests {
		go func() { defer workersWg.Done(); worker(ctx, workChan, params) }()
	}

	var twg sync.WaitGroup
	twg.Go(func() {
		for {
			select {
			case <-ticker.C:
				lastSentMetrics = metricsHolder.Load()
				workChan <- lastSentMetrics
			case <-ctx.Done():
				return
			}
		}
	})

	for m := range params.metricsChannel {
		metricsHolder.Store(m)
	}
	twg.Wait()

	lastBatch := metricsHolder.Load()
	if !maps.Equal(lastBatch, lastSentMetrics) {
		workChan <- lastBatch
	}
	close(workChan)
	workersWg.Wait()
}

func worker(ctx context.Context, c chan map[string]agentMetric, params SendParams) {
	// WithoutCancel, чтобы отправить последний батч метрик
	// после отмены контекста в рамках gracefulShutdown.
	base := context.WithoutCancel(ctx)
	for metrics := range c {
		sendCtx, cancel := context.WithTimeout(base, sendTimeout)
		err := sendMetricBatch(sendCtx, metrics, params)
		cancel()
		if err != nil {
			params.logger.Error("Error sending metrics batch", zap.Error(err))
		}
	}
}

func toModelMetrics(metrics map[string]agentMetric) ([]models.Metrics, error) {
	var payload []models.Metrics
	for k, v := range metrics {
		metric := models.Metrics{}
		metric.ID = k
		metric.MType = v.Type
		switch v.Type {
		case models.Counter:
			intVal := int64(v.Value)
			metric.Delta = &intVal
		case models.Gauge:
			metric.Value = &v.Value
		default:
			return nil, fmt.Errorf("unsupported metric type: %s", v.Type)
		}
		payload = append(payload, metric)
	}
	return payload, nil
}

func sendMetricBatch(ctx context.Context, metrics map[string]agentMetric, params SendParams) error {
	payload, err := toModelMetrics(metrics)
	if err != nil {
		return fmt.Errorf("error converting metrics: %w", err)
	}

	if len(payload) == 0 {
		return nil
	}
	err = params.sender.Send(ctx, payload)
	if err != nil {
		return fmt.Errorf("error sending metrics: %w", err)
	}
	return nil
}
