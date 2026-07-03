package agent

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/shirou/gopsutil/v4/mem"
	"github.com/stretchr/testify/assert"

	models "github.com/LifeforDream/gometrics/internal/model"
)

func TestBuildPsUtilSnapshot(t *testing.T) {
	tests := []struct {
		name        string
		memStat     *mem.VirtualMemoryStat
		cpuData     []float64
		wantMetrics map[string]AgentMetric
	}{
		{
			name:    "memory only no CPUs",
			memStat: &mem.VirtualMemoryStat{Total: 8192, Free: 2048},
			cpuData: []float64{},
			wantMetrics: map[string]AgentMetric{
				"TotalMemory": {Type: models.Gauge, Value: 8192},
				"FreeMemory":  {Type: models.Gauge, Value: 2048},
			},
		},
		{
			name:    "single CPU core",
			memStat: &mem.VirtualMemoryStat{Total: 1000, Free: 500},
			cpuData: []float64{42.5},
			wantMetrics: map[string]AgentMetric{
				"TotalMemory":     {Type: models.Gauge, Value: 1000},
				"FreeMemory":      {Type: models.Gauge, Value: 500},
				"CPUutilization1": {Type: models.Gauge, Value: 42.5},
			},
		},
		{
			name:    "multiple CPU cores indexed correctly",
			memStat: &mem.VirtualMemoryStat{Total: 4096, Free: 1024},
			cpuData: []float64{10.0, 20.0, 30.0, 40.0},
			wantMetrics: map[string]AgentMetric{
				"TotalMemory":     {Type: models.Gauge, Value: 4096},
				"FreeMemory":      {Type: models.Gauge, Value: 1024},
				"CPUutilization1": {Type: models.Gauge, Value: 10.0},
				"CPUutilization2": {Type: models.Gauge, Value: 20.0},
				"CPUutilization3": {Type: models.Gauge, Value: 30.0},
				"CPUutilization4": {Type: models.Gauge, Value: 40.0},
			},
		},
		{
			name:    "zero memory values",
			memStat: &mem.VirtualMemoryStat{},
			cpuData: []float64{},
			wantMetrics: map[string]AgentMetric{
				"TotalMemory": {Type: models.Gauge, Value: 0},
				"FreeMemory":  {Type: models.Gauge, Value: 0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildPsUtilSnapshot(tt.memStat, tt.cpuData)
			assert.Equal(t, tt.wantMetrics, got)

			// verify CPU keys follow the expected naming pattern
			for i := range tt.cpuData {
				key := fmt.Sprintf("CPUutilization%d", i+1)
				assert.Contains(t, got, key, "expected key %s in result", key)
				assert.Equal(t, models.Gauge, got[key].Type)
			}
		})
	}
}

func TestBuildMemStatsSnapshot(t *testing.T) {
	metricNames := []string{
		"Alloc",
		"BuckHashSys",
		"Frees",
		"GCCPUFraction",
		"GCSys",
		"HeapAlloc",
		"HeapIdle",
		"HeapInuse",
		"HeapObjects",
		"HeapReleased",
		"HeapSys",
		"LastGC",
		"Lookups",
		"MCacheInuse",
		"MCacheSys",
		"MSpanInuse",
		"MSpanSys",
		"Mallocs",
		"NextGC",
		"NumForcedGC",
		"NumGC",
		"OtherSys",
		"PauseTotalNs",
		"StackInuse",
		"StackSys",
		"Sys",
		"TotalAlloc",
		"PollCount",
		"RandomValue",
	}
	var memStats runtime.MemStats
	var res map[string]AgentMetric

	runtime.ReadMemStats(&memStats)
	res = buildMemStatsSnapshot(memStats, 0)
	assert.Equal(t, 0, int(res["PollCount"].Value))

	res = buildMemStatsSnapshot(memStats, 1)
	assert.Equal(t, 1, int(res["PollCount"].Value))

	keys := make([]string, 0, len(res))
	for k := range res {
		keys = append(keys, k)
	}
	assert.ElementsMatch(t, metricNames, keys)
	for k, v := range res {
		assert.NotEmpty(t, v.Type, "metric type should not be empty for %s", k)
		if k == "PollCount" {
			assert.Equal(t, "counter", v.Type, "PollCount should be of type counter")
		} else {
			assert.Equal(t, "gauge", v.Type, "metric %s should be of type gauge", k)
		}
	}
}
