// Package router собирает chi.Router для сервера метрик поверх
// internal/handler.
package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/LifeforDream/gometrics/internal/handler"
)

// MetricsRouter регистрирует все маршруты сервера метрик поверх h и
// подключает переданные middlewares в порядке перечисления (первый
// оборачивает запрос первым). Сам роутер не зависит от конкретных
// мидлваров — их подключение остаётся на стороне вызывающего кода
// (cmd/server/main.go).
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
	r.Post("/updates", h.UpdateMetrics)

	return r
}
