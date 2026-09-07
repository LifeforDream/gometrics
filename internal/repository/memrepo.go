// Package repository содержит реализации интерфейса service.MetricRepo:
// MemStorage (в памяти), FileBackedStorage (с сохранением на диск) и
// DBStorage (PostgreSQL).
package repository

import (
	"context"
	"maps"
	"slices"
	"sync"

	models "github.com/LifeforDream/gometrics/internal/model"
	myErrors "github.com/LifeforDream/gometrics/internal/model/errors"
)

// MemStorage — потокобезопасное хранилище метрик в памяти на базе
// map[string]models.Metrics с защитой через sync.RWMutex. Batch-метод
// UpdateMetrics держит блокировку на всю итерацию, что гарантирует
// атомарность батча.
type MemStorage struct {
	mu    sync.RWMutex
	store map[string]models.Metrics
}

// NewMemStorage создаёт пустое MemStorage.
func NewMemStorage() *MemStorage {
	return &MemStorage{store: make(map[string]models.Metrics)}
}

// Ping всегда возвращает nil — хранилище в памяти не имеет внешнего
// соединения, которое можно было бы проверить.
func (m *MemStorage) Ping(ctx context.Context) error {
	return nil
}

// GetAllSlice возвращает копию всех метрик в виде слайса.
func (m *MemStorage) GetAllSlice(ctx context.Context) ([]models.Metrics, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return slices.Collect(maps.Values(m.store)), nil
}

// GetAll возвращает поверхностную копию внутренней карты метрик.
func (m *MemStorage) GetAll() map[string]models.Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return maps.Clone(m.store)
}

// SetAll полностью заменяет внутреннее хранилище переданной картой,
// не сливая её с уже имеющимися данными.
func (m *MemStorage) SetAll(data map[string]models.Metrics) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store = data
}

// GetMetric возвращает метрику по имени или myErrors.ErrMetricNotFound,
// если такой метрики нет.
func (m *MemStorage) GetMetric(ctx context.Context, name string) (models.Metrics, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	metric, ok := m.store[name]
	if !ok {
		return models.Metrics{}, myErrors.ErrMetricNotFound
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

// SetGauge заменяет хранимое значение метрики типа gauge. Возвращает
// myErrors.InvalidMetricType, если метрика с таким именем уже существует
// с другим типом.
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

// UpdateCounter прибавляет Delta метрики к ранее сохранённому значению
// метрики типа counter. Возвращает myErrors.InvalidMetricType, если
// метрика с таким именем уже существует с другим типом.
func (m *MemStorage) UpdateCounter(ctx context.Context, metric models.Metrics) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.updateCounter(metric)
}

// UpdateMetrics атомарно применяет батч метрик: удерживает блокировку на
// всю итерацию, останавливаясь на первой ошибке.
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
			err = myErrors.ErrNonexistentMetricType
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// Close всегда возвращает nil — хранилищу в памяти нечего сбрасывать
// на завершении работы.
func (m *MemStorage) Close() error {
	return nil
}
