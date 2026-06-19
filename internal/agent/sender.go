package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	models "github.com/LifeforDream/gometrics/internal/model"
	"go.uber.org/zap"
)

func send(ctx context.Context, interval int, c chan map[string]AgentMetric, serverAddress string, logger *zap.Logger) {
	client := &http.Client{}
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

func compress(b *bytes.Buffer) error {
	od, err := io.ReadAll(b)
	b.Reset()

	if err != nil {
		return fmt.Errorf("failed to read from uncompressed data: %w", err)
	}
	w, err := gzip.NewWriterLevel(b, gzip.BestCompression)
	if err != nil {
		return fmt.Errorf("failed init compress writer: %w", err)
	}
	_, err = w.Write(od)
	if err != nil {
		return fmt.Errorf("failed write data to compress temporary buffer: %w", err)
	}
	err = w.Close()
	if err != nil {
		return fmt.Errorf("failed compress data: %w", err)
	}
	return nil
}

func sendMetric(metricType, metricName, serverAddress string, metricValue float64, client *http.Client) error {
	var buf bytes.Buffer
	var req models.Metrics

	switch metricType {
	case models.Counter:
		intVal := int64(metricValue)
		req = models.Metrics{
			ID:    metricName,
			MType: metricType,
			Delta: &intVal,
		}
	case models.Gauge:
		req = models.Metrics{
			ID:    metricName,
			MType: metricType,
			Value: &metricValue,
		}
	default:
		return fmt.Errorf("unsupported metric type: %s", metricType)
	}

	enc := json.NewEncoder(&buf)
	if err := enc.Encode(req); err != nil {
		return err
	}

	err := compress(&buf)
	if err != nil {
		return err
	}

	request, err := http.NewRequest(http.MethodPost, serverAddress+"/update", &buf)
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

	err := compress(&buf)
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
