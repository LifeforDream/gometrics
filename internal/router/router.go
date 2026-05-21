package router

import (
	"github.com/LifeforDream/gometrics/internal/handler"
	"github.com/go-chi/chi/v5"
)

func MetricsRouter(h *handler.Handler) chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.GetMetrics)
	r.Get("/value/{type}/{name}", h.GetMetric)
	r.Post("/update/{type}/{name}/{value}", h.UpdateMetric)

	return r
}
