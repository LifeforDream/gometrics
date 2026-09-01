package agent

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"go.uber.org/zap"

	"github.com/LifeforDream/gometrics/internal/compress"
	models "github.com/LifeforDream/gometrics/internal/model"
	"github.com/LifeforDream/gometrics/internal/utils"
)

func send(ctx context.Context, logger *zap.Logger, interval int, c chan map[string]AgentMetric, serverAddress, hashKey string, concreqs int) {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 3
	retryClient.Backoff = func(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration {
		return time.Duration(2*attemptNum+1) * time.Second
	}
	retryClient.Logger = nil
	client := retryClient.StandardClient()

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	workChan := make(chan map[string]AgentMetric, concreqs)

	for range concreqs {
		go worker(ctx, logger, workChan, serverAddress, hashKey, client)
	}

	for {
		select {
		case <-ticker.C:
			select {
			case metrics := <-c:
				workChan <- metrics
			case <-ctx.Done():
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func worker(ctx context.Context, logger *zap.Logger, c chan map[string]AgentMetric, serverAddress, hashKey string, client *http.Client) {
	for {
		select {
		case metrics := <-c:
			err := sendMetricBatch(metrics, serverAddress, hashKey, client)
			if err != nil {
				logger.Error("Error sending metrics batch", zap.Error(err))
			}
		case <-ctx.Done():
			return
		}
	}
}

func sendMetricBatch(metrics map[string]AgentMetric, serverAddress, hashKey string, client *http.Client) error {
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

	d, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	zw, err := compress.NewWriter(&buf)
	if err != nil {
		return err
	}

	_, err = zw.Write(d)
	if err != nil {
		return err
	}

	err = zw.Close()
	if err != nil {
		return err
	}

	request, err := http.NewRequest(http.MethodPost, serverAddress+"/updates", &buf)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Encoding", "gzip")
	request.Header.Set("Content-Type", "application/json")

	if hashKey != "" {
		hash := utils.GenSHA256(buf.Bytes(), hashKey)
		request.Header.Set(utils.HashHeaderName, hex.EncodeToString(hash))
	}

	resp, err := client.Do(request)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return nil
}
