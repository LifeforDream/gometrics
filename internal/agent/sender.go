package agent

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	models "github.com/LifeforDream/gometrics/internal/model"
)

func send(interval int, c chan map[string]AgentMetric, serverAddress string) {
	client := &http.Client{}
	for {
		time.Sleep(time.Duration(interval) * time.Second)
		metrics := <-c
		for name, metric := range metrics {
			// send to goroutine?
			err := sendMetric(metric.Type, name, serverAddress, metric.Value, client)
			if err != nil {
				log.Printf("Error sending metric %s: %v\n", name, err)
			}
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
