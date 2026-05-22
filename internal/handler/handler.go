package handler

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	myErrors "github.com/LifeforDream/gometrics/internal/model/errors"
	"github.com/LifeforDream/gometrics/internal/service"
	"github.com/go-chi/chi/v5"
)

const pageHtml = `<html>
<body>
<h1> Known Metrics: </h1>
%s
</body>
</html>
`

type Handler struct {
	service *service.MetricService
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

func (h *Handler) GetMetric(w http.ResponseWriter, r *http.Request) {
	metricType := strings.ToLower(chi.URLParam(r, "type"))
	metricName := strings.ToLower(chi.URLParam(r, "name"))

	value, err := h.service.GetMetric(metricType, metricName)
	if err != nil {
		var invalidTypeErr myErrors.InvalidMetricType
		if errors.As(err, &invalidTypeErr) {
			log.Printf("Invalid metric type: %s", invalidTypeErr.NewType)
			w.WriteHeader(http.StatusBadRequest)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(value))
}

func (h *Handler) UpdateMetric(w http.ResponseWriter, r *http.Request) {
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
		var invalidTypeErr myErrors.InvalidMetricType
		if errors.As(servErr, &invalidTypeErr) {
			log.Printf("Invalid metric type: %s", invalidTypeErr.NewType)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
