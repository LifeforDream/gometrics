package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/LifeforDream/gometrics/internal/logger"
	models "github.com/LifeforDream/gometrics/internal/model"
	"go.uber.org/zap"
)

func send(ctx context.Context, interval int, c chan map[string]AgentMetric, serverAddress string) {
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
						logger.Log.Error("Error sending metric", zap.String("metricName", name), zap.Error(err))
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
		logger.Log.Debug("error encoding response", zap.Error(err))
		return err
	}
	resp, err := client.Post(serverAddress+"/update", "application/json", &buf)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return nil
}
