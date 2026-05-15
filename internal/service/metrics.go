package service

import (
	models "github.com/LifeforDream/gometrics/internal/model"
	merrors "github.com/LifeforDream/gometrics/internal/model/errors"
	repository "github.com/LifeforDream/gometrics/internal/repository"
)

type MetricService struct {
	repo repository.MetricRepo
}

func NewMetricService(repo repository.MetricRepo) *MetricService {
	return &MetricService{repo: repo}
}

func (s *MetricService) UpdateGauge(name string, value float64) error {
	curVal, ok := s.repo.GetMetric(name)
	if ok {
		if curVal.MType != models.Gauge {
			return merrors.InvalidMetricType{
				ExistingType: curVal.MType,
				NewType:      models.Gauge,
				MetricName:   name,
			}
		}
	}
	metric := models.Metrics{
		ID:    name,
		MType: models.Gauge,
		Value: &value,
	}
	s.repo.SetMetric(metric)
	return nil
}

func (s *MetricService) UpdateCounter(name string, value int64) error {
	var startVal int64
	curVal, ok := s.repo.GetMetric(name)
	if !ok {
		startVal = int64(0)
	} else {
		if curVal.MType != models.Counter {
			return merrors.InvalidMetricType{
				ExistingType: curVal.MType,
				NewType:      models.Counter,
				MetricName:   name,
			}
		}
		startVal = int64(*curVal.Value)
	}
	newVal := float64(startVal + value)
	metric := models.Metrics{
		ID:    name,
		MType: models.Counter,
		Value: &newVal,
	}
	s.repo.SetMetric(metric)
	return nil
}
