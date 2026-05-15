package handler

import (
	"net/http"
	"strconv"

	merrors "github.com/LifeforDream/gometrics/internal/model/errors"
	"github.com/LifeforDream/gometrics/internal/service"
)

type Handler struct {
	service *service.MetricService
}

func NewHandler(service *service.MetricService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) UpdateMetric(w http.ResponseWriter, r *http.Request) {
	var servErr error

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	metricType := r.PathValue("type")
	metricName := r.PathValue("name")
	metricValue := r.PathValue("value")

	switch metricType {
	case "gauge":
		floatVal, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		servErr = h.service.UpdateGauge(metricName, floatVal)
	case "counter":
		intVal, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		servErr = h.service.UpdateCounter(metricName, intVal)
	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if servErr != nil {
		_, ok := servErr.(merrors.InvalidMetricType)
		if ok {
			w.WriteHeader(http.StatusBadRequest)
			return
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}
