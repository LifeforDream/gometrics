package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/LifeforDream/gometrics/internal/compress"
	models "github.com/LifeforDream/gometrics/internal/model"
	"github.com/hashicorp/go-retryablehttp"
	"go.uber.org/zap"
)

func send(ctx context.Context, interval int, c chan map[string]AgentMetric, serverAddress string, logger *zap.Logger) {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 3
	retryClient.Backoff = func(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration {
		return time.Duration(2*attemptNum+1) * time.Second
	}
	retryClient.Logger = nil

	client := retryClient.StandardClient()
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			select {
			case metrics := <-c:
				err := sendMetricBatch(metrics, serverAddress, client)
				if err != nil {
					logger.Error("Error sending metrics batch", zap.Error(err))
				}
			case <-ctx.Done():
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func sendMetricBatch(metrics map[string]AgentMetric, serverAddress string, client *http.Client) error {
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
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(payload); err != nil {
		return err
	}

	err := compress.Buffer(&buf)
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodPost, serverAddress+"/updates", &buf)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Encoding", "gzip")
	request.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(request)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return nil
}
