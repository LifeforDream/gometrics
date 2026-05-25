package agent

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	models "github.com/LifeforDream/gometrics/internal/model"
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
						log.Printf("Error sending metric %s: %v\n", name, err)
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
	var url string
	switch metricType {
	case models.Counter:
		url = fmt.Sprintf("%s/update/%s/%s/%d", serverAddress, metricType, metricName, int64(metricValue))
	case models.Gauge:
		url = fmt.Sprintf("%s/update/%s/%s/%f", serverAddress, metricType, metricName, metricValue)
	default:
		return fmt.Errorf("unsupported metric type: %s", metricType)
	}
	resp, err := client.Post(url, "text/plain", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return nil
}
