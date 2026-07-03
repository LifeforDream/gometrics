package agent

import (
	"context"
	"fmt"
	"maps"
	"math/rand"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"go.uber.org/zap"

	models "github.com/LifeforDream/gometrics/internal/model"
)

func buildMemStatsSnapshot(memStats runtime.MemStats, pollCount int) map[string]AgentMetric {
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

func buildPsUtilSnapshot(m *mem.VirtualMemoryStat, cpudata []float64) map[string]AgentMetric {
	ret := map[string]AgentMetric{
		"TotalMemory": {Type: models.Gauge, Value: float64(m.Total)},
		"FreeMemory":  {Type: models.Gauge, Value: float64(m.Free)},
	}
	for core, v := range cpudata {
		k := fmt.Sprintf("CPUutilization%d", core+1)
		ret[k] = AgentMetric{Type: models.Gauge, Value: v}
	}
	return ret
}

func collectMemStats(ctx context.Context, interval int, c chan map[string]AgentMetric) {
	var memStats runtime.MemStats
	pollCount := 0
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			runtime.ReadMemStats(&memStats)
			pollCount++
			c <- buildMemStatsSnapshot(memStats, pollCount)
		case <-ctx.Done():
			return
		}
	}
}

func collectPsUtil(ctx context.Context, interval int, c chan map[string]AgentMetric, logger *zap.Logger) {
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m, err := mem.VirtualMemoryWithContext(ctx)
			if err != nil {
				logger.Error("error collecting memory metrics", zap.Error(err))
				continue
			}

			cpud, err := cpu.PercentWithContext(ctx, 0, true)
			if err != nil {
				logger.Error("error collecting cpu metrics", zap.Error(err))
				continue
			}

			c <- buildPsUtilSnapshot(m, cpud)
		case <-ctx.Done():
			return
		}
	}
}

func collect(ctx context.Context, interval int, c chan map[string]AgentMetric, logger *zap.Logger) {
	colChan := make(chan map[string]AgentMetric, 2) // 2 goroutines = 2 slots
	metricMap := make(map[string]AgentMetric)

	go collectMemStats(ctx, interval, colChan)
	go collectPsUtil(ctx, interval, colChan, logger)

	for {
		select {
		case tempMap := <-colChan:
			maps.Copy(metricMap, tempMap)
			// drain channel to avoid waiting
			select {
			case <-c:
			default:
			}
			c <- maps.Clone(metricMap)
		case <-ctx.Done():
			return
		}
	}
}
