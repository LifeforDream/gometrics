package main

import (
	"net/http"

	"github.com/LifeforDream/gometrics/internal/handler"
	"github.com/LifeforDream/gometrics/internal/repository"
	"github.com/LifeforDream/gometrics/internal/service"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	mux := http.NewServeMux()
	h := handler.NewHandler(service.NewMetricService(&repository.MemStorage{}))
	mux.HandleFunc(`/update/{type}/{name}/{value}`, h.UpdateMetric)
	return http.ListenAndServe(`:8080`, mux)
}
