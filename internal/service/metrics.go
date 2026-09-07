// Package service содержит бизнес-логику слоя между хендлером и
// репозиторием: gauge заменяет значение, counter накапливает.
package service

import (
	"context"
	"fmt"
	"strconv"

	models "github.com/LifeforDream/gometrics/internal/model"
	myErrors "github.com/LifeforDream/gometrics/internal/model/errors"
	"github.com/LifeforDream/gometrics/internal/utils"
)

// MetricRepo — контракт хранилища метрик, реализуемый MemStorage,
// FileBackedStorage и DBStorage; MetricService работает с любым из них
// через этот интерфейс.
type MetricRepo interface {
	GetAllSlice(ctx context.Context) ([]models.Metrics, error)
	GetMetric(ctx context.Context, name string) (models.Metrics, error)
	SetGauge(ctx context.Context, metric models.Metrics) error
	UpdateCounter(ctx context.Context, metric models.Metrics) error
	UpdateMetrics(ctx context.Context, metrics []models.Metrics) error
	Ping(ctx context.Context) error
	Close() error
}

// Auditor — получатель уведомлений об успешных обновлениях метрик
// (Observer-паттерн); реализуется audit.Auditor.
type Auditor interface {
	Update([]models.Metrics, string)
}

// MetricService реализует бизнес-правила поверх MetricRepo: gauge заменяет
// хранимое значение, counter накапливает его с предыдущим; после успешного
// изменения репозитория уведомляет auditor.
type MetricService struct {
	repo    MetricRepo
	auditor Auditor
}

// NewMetricService создаёт MetricService поверх переданных репозитория
// и аудитора.
func NewMetricService(repo MetricRepo, auditor Auditor) *MetricService {
	return &MetricService{repo: repo, auditor: auditor}
}

func (s *MetricService) withAudit(ctx context.Context, f func(context.Context) ([]models.Metrics, error)) error {
	metrics, err := f(ctx)
	if err == nil {
		s.auditor.Update(metrics, utils.ClientIP(ctx))
	}
	return err
}

// Ping проверяет доступность репозитория.
func (s *MetricService) Ping(ctx context.Context) error {
	return s.repo.Ping(ctx)
}

// GetMetrics возвращает все метрики в виде строк "имя значение" для
// отображения на HTML-странице.
func (s *MetricService) GetMetrics(ctx context.Context) ([]string, error) {
	metrics, err := s.repo.GetAllSlice(ctx)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, v := range metrics {
		switch v.MType {
		case models.Gauge:
			out = append(out, fmt.Sprintf("%s %f", v.ID, *v.Value))
		case models.Counter:
			out = append(out, fmt.Sprintf("%s %d", v.ID, *v.Delta))
		}
	}
	return out, nil
}

// GetMetricValue возвращает значение метрики в виде строки, если её
// фактический тип совпадает с metricType; иначе возвращает
// myErrors.InvalidMetricType.
func (s *MetricService) GetMetricValue(ctx context.Context, metricType, name string) (string, error) {
	metric, err := s.repo.GetMetric(ctx, name)
	if err != nil {
		return "", err
	}
	if metric.MType != metricType {
		return "", myErrors.InvalidMetricType{
			ExistingType: metric.MType,
			NewType:      metricType,
			MetricName:   name,
		}
	}
	switch metric.MType {
	case models.Gauge:
		return strconv.FormatFloat(*metric.Value, 'f', -1, 64), nil
	case models.Counter:
		return strconv.FormatInt(*metric.Delta, 10), nil
	}
	return "", myErrors.InvalidMetricType{
		ExistingType: metric.MType,
		NewType:      metricType,
		MetricName:   name,
	}
}

// GetMetric возвращает метрику целиком, если её фактический тип совпадает
// с metricType; иначе возвращает myErrors.InvalidMetricType.
func (s *MetricService) GetMetric(ctx context.Context, metricType, name string) (models.Metrics, error) {
	metric, err := s.repo.GetMetric(ctx, name)
	if err != nil {
		return models.Metrics{}, err
	}
	if metric.MType != metricType {
		return models.Metrics{}, myErrors.InvalidMetricType{
			ExistingType: metric.MType,
			NewType:      metricType,
			MetricName:   name,
		}
	}
	return metric, nil
}

// UpdateGaugeByName собирает метрику типа gauge из имени и значения и
// сохраняет её, заменяя предыдущее значение и отправляя в auditor.
func (s *MetricService) UpdateGaugeByName(ctx context.Context, name string, value float64) error {
	return s.withAudit(ctx, func(context.Context) ([]models.Metrics, error) {
		metric := models.Metrics{
			ID:    name,
			MType: models.Gauge,
			Value: &value,
		}
		err := s.repo.SetGauge(ctx, metric)
		return []models.Metrics{metric}, err
	})

}

// UpdateGauge сохраняет метрику типа gauge, заменяя предыдущее значение
// и отправляя уведомление в auditor.
func (s *MetricService) UpdateGauge(ctx context.Context, metric models.Metrics) error {
	return s.withAudit(ctx, func(context.Context) ([]models.Metrics, error) {
		err := s.repo.SetGauge(ctx, metric)
		return []models.Metrics{metric}, err
	})

}

// UpdateCounterByName собирает метрику типа counter из имени и дельты и
// накапливает её поверх ранее сохранённого значения.
func (s *MetricService) UpdateCounterByName(ctx context.Context, name string, delta int64) error {
	return s.withAudit(ctx, func(context.Context) ([]models.Metrics, error) {
		metric := models.Metrics{
			ID:    name,
			MType: models.Counter,
			Delta: &delta,
		}
		err := s.repo.UpdateCounter(ctx, metric)
		return []models.Metrics{metric}, err
	})
}

// UpdateCounter накапливает метрику типа counter поверх ранее сохранённого
// значения, после чего отправляет в auditor.
func (s *MetricService) UpdateCounter(ctx context.Context, metric models.Metrics) error {
	return s.withAudit(ctx, func(context.Context) ([]models.Metrics, error) {
		err := s.repo.UpdateCounter(ctx, metric)
		return []models.Metrics{metric}, err
	})
}

// UpdateMetrics атомарно применяет батч метрик (gauge заменяют значение,
// counter накапливают его).
func (s *MetricService) UpdateMetrics(ctx context.Context, metrics []models.Metrics) error {
	return s.withAudit(ctx, func(context.Context) ([]models.Metrics, error) {
		err := s.repo.UpdateMetrics(ctx, metrics)
		return metrics, err
	})
}

// ValidateMetric проверяет, что тип метрики известен и соответствующее
// ему поле (Delta для counter, Value для gauge) заполнено.
func (s *MetricService) ValidateMetric(metric models.Metrics) error {
	switch metric.MType {
	case models.Counter:
		if metric.Delta == nil {
			return myErrors.ErrEmptyCounterDelta
		}
	case models.Gauge:
		if metric.Value == nil {
			return myErrors.ErrEmptyGaugeValue
		}
	default:
		return myErrors.ErrNonexistentMetricType
	}
	return nil
}
