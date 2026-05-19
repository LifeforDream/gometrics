package main

import (
	"net/http"

	"github.com/LifeforDream/gometrics/internal/handler"
	"github.com/LifeforDream/gometrics/internal/repository"
	"github.com/LifeforDream/gometrics/internal/service"
	"github.com/go-chi/chi/v5"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func MetricsRouter() chi.Router {
	r := chi.NewRouter()
	h := handler.NewHandler(service.NewMetricService(&repository.MemStorage{}))

	r.Route("/", func(r chi.Router) {
		r.Get("/", h.GetMetrics)                   // metrics list page
		r.Get("/value/{type}/{name}", h.GetMetric) // metric value page
		r.Post("/update/{type}/{name}/{value}", h.UpdateMetric)
	})
	return r
}

func run() error {
	return http.ListenAndServe(`:8080`, MetricsRouter())
}
