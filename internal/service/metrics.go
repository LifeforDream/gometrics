package service

import (
	"context"
	"fmt"
	"strconv"

	models "github.com/LifeforDream/gometrics/internal/model"
	myErrors "github.com/LifeforDream/gometrics/internal/model/errors"
)

type MetricRepo interface {
	GetAllSlice(ctx context.Context) ([]models.Metrics, error)
	GetMetric(ctx context.Context, name string) (models.Metrics, error)
	SetGauge(ctx context.Context, metric models.Metrics) error
	UpdateCounter(ctx context.Context, metric models.Metrics) error
	UpdateMetrics(ctx context.Context, metrics []models.Metrics) error
	Ping(ctx context.Context) error
	Close() error
}

type MetricService struct {
	repo MetricRepo
}

func NewMetricService(repo MetricRepo) *MetricService {
	return &MetricService{repo: repo}
}

func (s *MetricService) Ping(ctx context.Context) error {
	return s.repo.Ping(ctx)
}

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

func (s *MetricService) UpdateGaugeByName(ctx context.Context, name string, value float64) error {
	metric := models.Metrics{
		ID:    name,
		MType: models.Gauge,
		Value: &value,
	}
	err := s.repo.SetGauge(ctx, metric)
	return err
}

func (s *MetricService) UpdateGauge(ctx context.Context, metric models.Metrics) error {
	return s.repo.SetGauge(ctx, metric)
}

func (s *MetricService) UpdateCounterByName(ctx context.Context, name string, delta int64) error {
	metric := models.Metrics{
		ID:    name,
		MType: models.Counter,
		Delta: &delta,
	}
	err := s.repo.UpdateCounter(ctx, metric)
	return err
}

func (s *MetricService) UpdateCounter(ctx context.Context, metric models.Metrics) error {
	return s.repo.UpdateCounter(ctx, metric)
}

func (s *MetricService) UpdateMetrics(ctx context.Context, metrics []models.Metrics) error {
	return s.repo.UpdateMetrics(ctx, metrics)
}

func (s *MetricService) ValidateMetric(metric models.Metrics) error {
	switch metric.MType {
	case models.Counter:
		if metric.Delta == nil {
			return myErrors.EmptyCounterDelta
		}
	case models.Gauge:
		if metric.Value == nil {
			return myErrors.EmptyGaugeValue
		}
	default:
		return myErrors.NonexistentMetricType
	}
	return nil
}
