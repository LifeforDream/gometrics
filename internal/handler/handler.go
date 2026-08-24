// Package handler реализует HTTP-слой: парсинг path-параметров и тела
// запроса, валидация типов и маппинг ошибок сервисного слоя в HTTP-статусы.
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	models "github.com/LifeforDream/gometrics/internal/model"
	myErrors "github.com/LifeforDream/gometrics/internal/model/errors"
)

const pageHtml = `<html>
<body>
<h1> Known Metrics: </h1>
%s
</body>
</html>
`

// MetricService — контракт сервисного слоя, ожидаемый Handler; реализуется
// service.MetricService.
type MetricService interface {
	GetMetrics(ctx context.Context) ([]string, error)
	GetMetricValue(ctx context.Context, metricType string, name string) (string, error)
	GetMetric(ctx context.Context, metricType string, name string) (models.Metrics, error)
	UpdateGauge(ctx context.Context, metric models.Metrics) error
	UpdateCounter(ctx context.Context, metric models.Metrics) error
	UpdateMetrics(ctx context.Context, metrics []models.Metrics) error
	ValidateMetric(metric models.Metrics) error
	Ping(ctx context.Context) error
}

// Handler объединяет HTTP-хендлеры сервера метрик поверх MetricService.
type Handler struct {
	service MetricService
	logger  *zap.Logger
}

// NewHandler создаёт Handler поверх переданных сервиса и логгера.
func NewHandler(service MetricService, logger *zap.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

// Ping обрабатывает GET /ping и проверяет доступность хранилища через
// MetricService.Ping.
func (h *Handler) Ping(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Ping(r.Context()); err != nil {
		h.logger.Error("Connection to database can't be established", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// GetMetrics обрабатывает GET / и отдаёт HTML-страницу со списком всех
// известных метрик.
func (h *Handler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	var metricslist []string
	metrics, err := h.service.GetMetrics(r.Context())
	if err != nil {
		h.logger.Error("Error while retrieving metrics", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
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

// GetMetricValue обрабатывает GET /value/{type}/{name} и возвращает
// значение метрики текстом. Отвечает 404, если метрика не найдена, и 400,
// если её фактический тип не совпадает с {type}.
func (h *Handler) GetMetricValue(w http.ResponseWriter, r *http.Request) {
	metricType := strings.ToLower(chi.URLParam(r, "type"))
	metricName := strings.ToLower(chi.URLParam(r, "name"))

	value, err := h.service.GetMetricValue(r.Context(), metricType, metricName)
	if err != nil {
		var invalidTypeErr myErrors.InvalidMetricType
		if errors.As(err, &invalidTypeErr) {
			h.logger.Debug("Invalid metric type", zap.String("newType", invalidTypeErr.NewType))
			w.WriteHeader(http.StatusBadRequest)
		} else if errors.Is(err, myErrors.MetricNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(value))
}

// UpdateMetricValue обрабатывает POST /update/{type}/{name}/{value}:
// парсит значение из path-параметра согласно {type} (gauge — float64,
// counter — int64) и обновляет метрику через сервисный слой.
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
		servErr = h.service.UpdateGauge(r.Context(), metric)
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
		servErr = h.service.UpdateCounter(r.Context(), metric)
	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if servErr != nil {
		var invalidTypeErr myErrors.InvalidMetricType
		if errors.As(servErr, &invalidTypeErr) {
			h.logger.Debug("Invalid metric type", zap.String("newType", invalidTypeErr.NewType))
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		h.logger.Error("Unexpected error while updating metrics value", zap.Error(servErr))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// GetMetric обрабатывает POST /value: принимает JSON-тело models.Metrics
// с заполненными ID и MType и возвращает метрику целиком в JSON.
// Отвечает 404, если метрика не найдена, и 400 при несовпадении типа
// или некорректном/пустом теле запроса.
func (h *Handler) GetMetric(w http.ResponseWriter, r *http.Request) {
	if ct := r.Header.Get("Content-Type"); ct != "application/json" {
		h.logger.Debug("Invalid content-type", zap.String("content-type", ct))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var req models.Metrics

	h.logger.Debug("decoding request")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Debug("cannot read request body", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal(body, &req); err != nil {
		h.logger.Debug("cannot decode request JSON body", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if req.ID == "" {
		h.logger.Debug("Metric name is empty, cannot retrieve")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if req.MType == "" {
		h.logger.Debug("Metric type is empty, cannot retrieve")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	metric, err := h.service.GetMetric(r.Context(), req.MType, req.ID)

	if err != nil {
		var invalidTypeErr myErrors.InvalidMetricType
		if errors.As(err, &invalidTypeErr) {
			h.logger.Debug("Invalid metric type", zap.String("newType", invalidTypeErr.NewType))
			w.WriteHeader(http.StatusBadRequest)
		} else if errors.Is(err, myErrors.MetricNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	enc := json.NewEncoder(w)
	if err := enc.Encode(metric); err != nil {
		h.logger.Debug("error encoding response", zap.Error(err))
		return
	}
	h.logger.Debug("sending HTTP 200 response")
}

// UpdateMetric обрабатывает POST /update: принимает JSON-тело models.Metrics
// и обновляет одну метрику через сервисный слой. Отвечает 400 при
// невалидном теле/метрике или несовпадении типа с уже сохранённым.
func (h *Handler) UpdateMetric(w http.ResponseWriter, r *http.Request) {
	if ct := r.Header.Get("Content-Type"); ct != "application/json" {
		h.logger.Debug("Invalid content-type", zap.String("content-type", ct))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var servErr error
	var req models.Metrics

	h.logger.Debug("decoding request")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Debug("cannot read request body", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal(body, &req); err != nil {
		h.logger.Debug("cannot decode request JSON body", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	valErr := h.service.ValidateMetric(req)
	if valErr != nil {
		h.logger.Debug("Invalid metric in Update", zap.String("metricName", req.ID), zap.Error(valErr))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	switch req.MType {
	case models.Counter:
		servErr = h.service.UpdateCounter(r.Context(), req)
	case models.Gauge:
		servErr = h.service.UpdateGauge(r.Context(), req)
	}

	if servErr != nil {
		var invalidTypeErr myErrors.InvalidMetricType
		if errors.As(servErr, &invalidTypeErr) {
			h.logger.Debug("Invalid metric type", zap.String("newType", invalidTypeErr.NewType))
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		h.logger.Error("Unexpected error while updating metric", zap.Error(servErr))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	h.logger.Debug("sending HTTP 200 response")
}

// UpdateMetrics обрабатывает POST /updates: принимает JSON-массив
// models.Metrics и обновляет их одним батчем через сервисный слой.
// Валидирует каждую метрику до отправки батча в сервис.
func (h *Handler) UpdateMetrics(w http.ResponseWriter, r *http.Request) {
	if ct := r.Header.Get("Content-Type"); ct != "application/json" {
		h.logger.Debug("Invalid content-type", zap.String("content-type", ct))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var req []models.Metrics
	var servErr error

	h.logger.Debug("decoding request")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Debug("cannot read request body", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal(body, &req); err != nil {
		h.logger.Debug("cannot decode request JSON body", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	for _, m := range req {
		valErr := h.service.ValidateMetric(m)
		if valErr != nil {
			h.logger.Debug("Validation failed metric in UpdateMetrics", zap.String("metricName", m.ID), zap.Error(valErr))
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	servErr = h.service.UpdateMetrics(r.Context(), req)
	if servErr != nil {
		var invalidTypeErr myErrors.InvalidMetricType
		if errors.As(servErr, &invalidTypeErr) {
			h.logger.Debug("Invalid metric type on batch update", zap.String("newType", invalidTypeErr.NewType))
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		h.logger.Error("Unexpected error while updating metrics", zap.Error(servErr))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	h.logger.Debug("sending HTTP 200 response")

}
