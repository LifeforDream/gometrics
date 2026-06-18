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

	r.Get("/ping", h.Ping)
	r.Get("/", h.GetMetrics)
	r.Get("/value/{type}/{name}", h.GetMetricValue)
	r.Post("/value", h.GetMetric)
	r.Post("/update/{type}/{name}/{value}", h.UpdateMetricValue)
	r.Post("/update", h.UpdateMetric)

	return r
}
