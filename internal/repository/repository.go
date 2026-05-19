package repository

import models "github.com/LifeforDream/gometrics/internal/model"

type MetricRepo interface {
	GetAll() []models.Metrics
	GetMetric(name string) (models.Metrics, bool)
	SetMetric(metricVal models.Metrics)
}
