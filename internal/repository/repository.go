package repository

import models "github.com/LifeforDream/gometrics/internal/model"

type MetricRepo interface {
	GetMetric(name string) (models.Metrics, bool)
	SetMetric(metricVal models.Metrics)
}
