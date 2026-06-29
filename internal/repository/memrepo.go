package repository

import (
	"context"
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

func (m *MemStorage) Ping(ctx context.Context) error {
	return nil
}

func (m *MemStorage) GetAllSlice(ctx context.Context) ([]models.Metrics, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return slices.Collect(maps.Values(m.store)), nil
}

func (m *MemStorage) GetAll() map[string]models.Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return maps.Clone(m.store)
}

func (m *MemStorage) SetAll(data map[string]models.Metrics) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store = data
}

func (m *MemStorage) GetMetric(ctx context.Context, name string) (models.Metrics, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	metric, ok := m.store[name]
	if !ok {
		return models.Metrics{}, myErrors.MetricNotFound
	}
	return metric, nil
}

func (m *MemStorage) setGauge(metric models.Metrics) error {
	exist, ok := m.store[metric.ID]
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

func (m *MemStorage) SetGauge(ctx context.Context, metric models.Metrics) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.setGauge(metric)
}

func (m *MemStorage) updateCounter(metric models.Metrics) error {
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
		startVal = *exist.Delta
	}
	newVal := startVal + *metric.Delta
	metric.Delta = &newVal
	m.store[metric.ID] = metric
	return nil
}

func (m *MemStorage) UpdateCounter(ctx context.Context, metric models.Metrics) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.updateCounter(metric)
}

func (m *MemStorage) UpdateMetrics(ctx context.Context, metrics []models.Metrics) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var err error

	for _, metric := range metrics {
		switch metric.MType {
		case models.Counter:
			err = m.updateCounter(metric)
		case models.Gauge:
			err = m.setGauge(metric)
		default:
			err = myErrors.NonexistentMetricType
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *MemStorage) Close() error {
	return nil
}
