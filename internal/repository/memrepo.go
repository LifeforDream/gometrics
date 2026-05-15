package repository

import models "github.com/LifeforDream/gometrics/internal/model"

type MemStorage struct {
	store map[string]models.Metrics
}

var GlobalStorage = MemStorage{
	store: make(map[string]models.Metrics),
}

func (m *MemStorage) GetMetric(name string) (models.Metrics, bool) {
	metric, ok := m.store[name]
	return metric, ok
}

func (m *MemStorage) SetMetric(metricVal models.Metrics) {
	m.store[metricVal.ID] = metricVal
}

func GetMetric(name string) (models.Metrics, bool) {
	return GlobalStorage.GetMetric(name)
}

func SetMetric(metricVal models.Metrics) {
	GlobalStorage.SetMetric(metricVal)
}
