package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/LifeforDream/gometrics/internal/logger"
	models "github.com/LifeforDream/gometrics/internal/model"
	myErrors "github.com/LifeforDream/gometrics/internal/model/errors"
	"github.com/LifeforDream/gometrics/internal/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

const pageHtml = `<html>
<body>
<h1> Known Metrics: </h1>
%s
</body>
</html>
`

type MetricService interface {
	GetMetrics() []string
	GetMetricValue(metricType string, name string) (string, error)
	GetMetric(metricType string, name string) (models.Metrics, error)
	UpdateGauge(models.Metrics) error
	UpdateCounter(models.Metrics) error
}

type Handler struct {
	service MetricService
}

func NewHandler(service *service.MetricService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	var metricslist []string
	metrics := h.service.GetMetrics()
	if len(metrics) > 0 {
		metricslist = append(metricslist, "<ul>")
		for _, metric := range metrics {
			metricslist = append(metricslist, fmt.Sprintf("<li>%s</li>", metric))
		}
		metricslist = append(metricslist, "</ul>")
	}
	page := fmt.Sprintf(pageHtml, strings.Join(metricslist, "\r\n"))

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(page))
}

func (h *Handler) GetMetricValue(w http.ResponseWriter, r *http.Request) {
	metricType := strings.ToLower(chi.URLParam(r, "type"))
	metricName := strings.ToLower(chi.URLParam(r, "name"))

	value, err := h.service.GetMetricValue(metricType, metricName)
	if err != nil {
		var invalidTypeErr myErrors.InvalidMetricType
		if errors.As(err, &invalidTypeErr) {
			logger.Log.Error("Invalid metric type", zap.String("newType", invalidTypeErr.NewType))
			w.WriteHeader(http.StatusBadRequest)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(value))
}

func (h *Handler) UpdateMetricValue(w http.ResponseWriter, r *http.Request) {
	var servErr error

	metricType := strings.ToLower(chi.URLParam(r, "type"))
	metricName := strings.ToLower(chi.URLParam(r, "name"))
	metricValue := strings.ToLower(chi.URLParam(r, "value"))

	switch metricType {
	case "gauge":
		floatVal, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		metric := models.Metrics{
			ID:    metricName,
			MType: models.Gauge,
			Value: &floatVal,
		}
		servErr = h.service.UpdateGauge(metric)
	case "counter":
		intVal, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		metric := models.Metrics{
			ID:    metricName,
			MType: models.Counter,
			Delta: &intVal,
		}
		servErr = h.service.UpdateCounter(metric)
	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if servErr != nil {
		var invalidTypeErr myErrors.InvalidMetricType
		if errors.As(servErr, &invalidTypeErr) {
			logger.Log.Error("Invalid metric type", zap.String("newType", invalidTypeErr.NewType))
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetMetric(w http.ResponseWriter, r *http.Request) {
	if ct := r.Header.Get("Content-Type"); ct != "application/json" {
		logger.Log.Debug("Invalid content-type", zap.String("content-type", ct))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var req models.Metrics

	logger.Log.Debug("decoding request")
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		logger.Log.Debug("cannot decode request JSON body", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if req.ID == "" {
		logger.Log.Debug("Metric name is empty, cannot retrieve")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if req.MType == "" {
		logger.Log.Debug("Metric type is empty, cannot retrieve")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	metric, err := h.service.GetMetric(req.MType, req.ID)

	if err != nil {
		var invalidTypeErr myErrors.InvalidMetricType
		if errors.As(err, &invalidTypeErr) {
			logger.Log.Error("Invalid metric type", zap.String("newType", invalidTypeErr.NewType))
			w.WriteHeader(http.StatusBadRequest)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	enc := json.NewEncoder(w)
	if err := enc.Encode(metric); err != nil {
		logger.Log.Debug("error encoding response", zap.Error(err))
		return
	}
	logger.Log.Debug("sending HTTP 200 response")
}

func (h *Handler) UpdateMetric(w http.ResponseWriter, r *http.Request) {
	if ct := r.Header.Get("Content-Type"); ct != "application/json" {
		logger.Log.Debug("Invalid content-type", zap.String("content-type", ct))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var servErr error
	var req models.Metrics

	logger.Log.Debug("decoding request")
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		logger.Log.Debug("cannot decode request JSON body", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	switch req.MType {
	case models.Counter:
		if req.Delta == nil {
			logger.Log.Debug("Empty Delta field for Counter")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		servErr = h.service.UpdateCounter(req)
	case models.Gauge:
		if req.Value == nil {
			logger.Log.Debug("Empty Value field for Gauge")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		servErr = h.service.UpdateGauge(req)
	default:
		logger.Log.Debug("Unexpected metric type", zap.String("type", req.MType))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if servErr != nil {
		var invalidTypeErr myErrors.InvalidMetricType
		if errors.As(servErr, &invalidTypeErr) {
			logger.Log.Error("Invalid metric type", zap.String("newType", invalidTypeErr.NewType))
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	logger.Log.Debug("sending HTTP 200 response")
}
