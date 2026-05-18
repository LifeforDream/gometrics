package agent

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_buildSnapshot(t *testing.T) {
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
	res = buildSnapshot(memStats, 0)
	assert.Equal(t, 0, int(res["PollCount"].Value))

	res = buildSnapshot(memStats, 1)
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
