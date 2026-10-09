// Package router собирает chi.Router для сервера метрик поверх
// internal/handler.
package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/LifeforDream/gometrics/internal/handler"
)

// Chain служит способом организовать разные группы мидлваров.
// Мидлвары применятся в порядке размещения в списке middlewares.
// Пустой Chain{} - валидная цепь.
type Chain struct {
	middlewares []func(http.Handler) http.Handler
}

// NewChain создаёт и возвращает Chain для группировки мидлваров.
func NewChain(middlewares ...func(http.Handler) http.Handler) Chain {
	return Chain{middlewares: middlewares}
}

// MetricsRouter регистрирует все маршруты сервера метрик поверх h и
// подключает переданные middlewares:
//   - core - для всех хэндлеров.
//   - readMws - для хэндлеров, возвращающих данные.
//   - writeMws - только для хэндлеров, изменяющих данные.
//
// Сам роутер не зависит от конкретных мидлваров — их подключение остаётся на стороне вызывающего кода.
func MetricsRouter(h *handler.Handler, core, readMws, writeMws Chain) chi.Router {
	r := chi.NewRouter()
	r.Use(core.middlewares...)

	r.Group(func(r chi.Router) {
		r.Use(readMws.middlewares...)
		r.Get("/ping", h.Ping)
		r.Get("/", h.GetMetrics)
		r.Get("/value/{type}/{name}", h.GetMetricValue)
		r.Post("/value", h.GetMetric)
	})

	r.Group(func(r chi.Router) {
		r.Use(writeMws.middlewares...)
		r.Post("/update/{type}/{name}/{value}", h.UpdateMetricValue)
		r.Post("/update", h.UpdateMetric)
		r.Post("/updates", h.UpdateMetrics)
	})

	return r
}
