package repository

import (
	"maps"
	"slices"

	models "github.com/LifeforDream/gometrics/internal/model"
)

type MemStorage struct {
	store map[string]models.Metrics
}

func (m *MemStorage) GetAll() []models.Metrics {
	return slices.Collect(maps.Values(m.store))
}

func (m *MemStorage) GetMetric(name string) (models.Metrics, bool) {
	metric, ok := m.store[name]
	return metric, ok
}

func (m *MemStorage) SetMetric(metricVal models.Metrics) {
	if m.store == nil {
		m.store = make(map[string]models.Metrics)
	}
	m.store[metricVal.ID] = metricVal
}
