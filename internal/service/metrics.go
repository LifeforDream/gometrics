package service

import (
	"fmt"
	"strconv"

	models "github.com/LifeforDream/gometrics/internal/model"
	myErrors "github.com/LifeforDream/gometrics/internal/model/errors"
)

type MetricRepo interface {
	GetAll() []models.Metrics
	GetMetric(name string) (models.Metrics, bool)
	SetGauge(metric models.Metrics) error
	UpdateCounter(metric models.Metrics) error
}

type MetricService struct {
	repo MetricRepo
}

func NewMetricService(repo MetricRepo) *MetricService {
	return &MetricService{repo: repo}
}

func (s *MetricService) GetMetrics() []string {
	metrics := s.repo.GetAll()
	var out []string
	for _, v := range metrics {
		switch v.MType {
		case models.Gauge:
			out = append(out, fmt.Sprintf("%s %f", v.ID, *v.Value))
		case models.Counter:
			out = append(out, fmt.Sprintf("%s %d", v.ID, int64(*v.Value)))
		}
	}
	return out
}

func (s *MetricService) GetMetric(metricType, name string) (string, error) {
	metric, ok := s.repo.GetMetric(name)
	if !ok {
		return "", myErrors.MetricNotFound
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
		return strconv.FormatInt(int64(*metric.Value), 10), nil
	}
	return "", myErrors.InvalidMetricType{
		ExistingType: metric.MType,
		NewType:      metricType,
		MetricName:   name,
	}
}

func (s *MetricService) UpdateGauge(name string, value float64) error {
	metric := models.Metrics{
		ID:    name,
		MType: models.Gauge,
		Value: &value,
	}
	err := s.repo.SetGauge(metric)
	return err
}

func (s *MetricService) UpdateCounter(name string, value int64) error {
	metric := models.Metrics{
		ID:    name,
		MType: models.Counter,
		Delta: &value,
	}
	err := s.repo.UpdateCounter(metric)
	return err
}
