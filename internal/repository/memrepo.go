package repository

import (
	"maps"
	"slices"
	"sync"

	models "github.com/LifeforDream/gometrics/internal/model"
	myErrors "github.com/LifeforDream/gometrics/internal/model/errors"
)

type MemStorage struct {
	mu    sync.RWMutex
	store map[string]models.Metrics
}

func NewMemStorage() *MemStorage {
	return &MemStorage{store: make(map[string]models.Metrics)}
}

func (m *MemStorage) GetAll() []models.Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return slices.Collect(maps.Values(m.store))
}

func (m *MemStorage) GetMetric(name string) (models.Metrics, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	metric, ok := m.store[name]
	return metric, ok
}

func (m *MemStorage) SetGauge(metric models.Metrics) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	exist, ok := m.store[metric.ID]
	// check for type
	if ok {
		if exist.MType != models.Gauge {
			return myErrors.InvalidMetricType{
				ExistingType: exist.MType,
				NewType:      models.Gauge,
				MetricName:   metric.ID,
			}
		}
	}
	m.store[metric.ID] = metric
	return nil
}

func (m *MemStorage) UpdateCounter(metric models.Metrics) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var startVal int64

	exist, ok := m.store[metric.ID]

	if ok {
		if exist.MType != models.Counter {
			return myErrors.InvalidMetricType{
				ExistingType: exist.MType,
				NewType:      models.Counter,
				MetricName:   metric.ID,
			}
		}
		startVal = int64(*exist.Value)
	}
	newVal := float64(startVal + *metric.Delta)
	// we could clean the Delta here
	// but seems to be no reason right now
	metric.Value = &newVal
	metric.Delta = nil
	m.store[metric.ID] = metric
	return nil
}
