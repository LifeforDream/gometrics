package router

import (
	"net/http"

	"github.com/LifeforDream/gometrics/internal/handler"
	"github.com/go-chi/chi/v5"
)

func MetricsRouter(h *handler.Handler, middlewares ...func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	for _, middleware := range middlewares {
		r.Use(middleware)
	}

	r.Get("/", h.GetMetrics)
	r.Post("/value", h.GetMetricJson)
	r.Get("/value/{type}/{name}", h.GetMetric)
	r.Post("/update", h.UpdateMetricJson)
	r.Post("/update/{type}/{name}/{value}", h.UpdateMetric)

	return r
}
