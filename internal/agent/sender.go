package agent

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/LifeforDream/gometrics/internal/compress"
	"github.com/LifeforDream/gometrics/internal/crypto"
	models "github.com/LifeforDream/gometrics/internal/model"
	"github.com/LifeforDream/gometrics/internal/utils"
)

// SendParams хранит параметры для запуска функции send() агента
type SendParams struct {
	logger        *zap.Logger
	interval      int
	c             chan map[string]agentMetric
	serverAddress string
	hashKey       string
	concreqs      int
	publicKey     *rsa.PublicKey
	client        httpSender
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

	workChan := make(chan map[string]agentMetric, params.concreqs)

	var workersWg sync.WaitGroup
	workersWg.Add(params.concreqs)
	for range params.concreqs {
		go func() { defer workersWg.Done(); worker(workChan, params) }()
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

	for m := range params.c {
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

func worker(c chan map[string]agentMetric, params SendParams) {
	for metrics := range c {
		err := sendMetricBatch(metrics, params)
		if err != nil {
			params.logger.Error("Error sending metrics batch", zap.Error(err))
		}
	}
}

func sendMetricBatch(metrics map[string]agentMetric, params SendParams) error {
	var buf bytes.Buffer
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
			return fmt.Errorf("unsupported metric type: %s", v.Type)
		}
		payload = append(payload, metric)
	}

	if len(payload) == 0 {
		return nil
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshalling data: %w", err)
	}

	zw, err := compress.NewWriter(&buf)
	if err != nil {
		return fmt.Errorf("error creating compressor: %w", err)
	}

	_, err = zw.Write(jsonData)
	if err != nil {
		return fmt.Errorf("error writing compressed data: %w", err)
	}

	err = zw.Close()
	if err != nil {
		return fmt.Errorf("error closing compress writer: %w", err)
	}

	var encData []byte
	if params.publicKey != nil {
		rawData, err := io.ReadAll(&buf)
		if err != nil {
			return fmt.Errorf("error reading data for signing: %w", err)
		}
		encData, err = crypto.Encrypt(params.publicKey, rawData)
		if err != nil {
			return fmt.Errorf("error encrypting data: %w", err)
		}
		buf = *bytes.NewBuffer(encData)
	}

	request, err := http.NewRequest(http.MethodPost, params.serverAddress+"/updates", &buf)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Encoding", "gzip")
	request.Header.Set("Content-Type", "application/json")

	if params.hashKey != "" {
		hash := utils.GenSHA256(buf.Bytes(), params.hashKey)
		request.Header.Set(utils.HashHeaderName, hex.EncodeToString(hash))
	}

	resp, err := params.client.Do(request)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return nil
}
