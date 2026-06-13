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
				for name, metric := range metrics {
					err := sendMetric(metric.Type, name, serverAddress, metric.Value, client)
					if err != nil {
						logger.Error("Error sending metric", zap.String("metricName", name), zap.Error(err))
					}
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
	// compresses data in place, modifies buffer
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
		return fmt.Errorf("failed write data to compress temporary buffer: %W", err)
	}
	err = w.Close()
	if err != nil {
		return fmt.Errorf("failed compress data: %W", err)
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

	// convert to json
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(req); err != nil {
		return err
	}

	// compress request
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
