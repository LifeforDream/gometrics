package agent

import (
	"math/rand"
	"runtime"
	"time"

	models "github.com/LifeforDream/gometrics/internal/model"
)

func buildSnapshot(memStats runtime.MemStats, pollCount int) map[string]AgentMetric {
	return map[string]AgentMetric{
		"Alloc":         {Type: models.Gauge, Value: float64(memStats.Alloc)},
		"BuckHashSys":   {Type: models.Gauge, Value: float64(memStats.BuckHashSys)},
		"Frees":         {Type: models.Gauge, Value: float64(memStats.Frees)},
		"GCCPUFraction": {Type: models.Gauge, Value: memStats.GCCPUFraction},
		"GCSys":         {Type: models.Gauge, Value: float64(memStats.GCSys)},
		"HeapAlloc":     {Type: models.Gauge, Value: float64(memStats.HeapAlloc)},
		"HeapIdle":      {Type: models.Gauge, Value: float64(memStats.HeapIdle)},
		"HeapInuse":     {Type: models.Gauge, Value: float64(memStats.HeapInuse)},
		"HeapObjects":   {Type: models.Gauge, Value: float64(memStats.HeapObjects)},
		"HeapReleased":  {Type: models.Gauge, Value: float64(memStats.HeapReleased)},
		"HeapSys":       {Type: models.Gauge, Value: float64(memStats.HeapSys)},
		"LastGC":        {Type: models.Gauge, Value: float64(memStats.LastGC)},
		"Lookups":       {Type: models.Gauge, Value: float64(memStats.Lookups)},
		"MCacheInuse":   {Type: models.Gauge, Value: float64(memStats.MCacheInuse)},
		"MCacheSys":     {Type: models.Gauge, Value: float64(memStats.MCacheSys)},
		"MSpanInuse":    {Type: models.Gauge, Value: float64(memStats.MSpanInuse)},
		"MSpanSys":      {Type: models.Gauge, Value: float64(memStats.MSpanSys)},
		"Mallocs":       {Type: models.Gauge, Value: float64(memStats.Mallocs)},
		"NextGC":        {Type: models.Gauge, Value: float64(memStats.NextGC)},
		"NumForcedGC":   {Type: models.Gauge, Value: float64(memStats.NumForcedGC)},
		"NumGC":         {Type: models.Gauge, Value: float64(memStats.NumGC)},
		"OtherSys":      {Type: models.Gauge, Value: float64(memStats.OtherSys)},
		"PauseTotalNs":  {Type: models.Gauge, Value: float64(memStats.PauseTotalNs)},
		"StackInuse":    {Type: models.Gauge, Value: float64(memStats.StackInuse)},
		"StackSys":      {Type: models.Gauge, Value: float64(memStats.StackSys)},
		"Sys":           {Type: models.Gauge, Value: float64(memStats.Sys)},
		"TotalAlloc":    {Type: models.Gauge, Value: float64(memStats.TotalAlloc)},
		"PollCount":     {Type: models.Counter, Value: float64(pollCount)},
		"RandomValue":   {Type: models.Gauge, Value: rand.Float64()},
	}
}

func collect(interval time.Duration, c chan map[string]AgentMetric) {
	var memStats runtime.MemStats
	pollCount := 0
	for {
		runtime.ReadMemStats(&memStats)
		pollCount++
		//drain channel
		select {
		case <-c:
		default:
		}
		c <- buildSnapshot(memStats, pollCount)
		time.Sleep(interval)
	}
}
