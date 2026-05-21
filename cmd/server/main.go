package main

import (
	"log"
	"net/http"

	"github.com/LifeforDream/gometrics/internal/handler"
	"github.com/LifeforDream/gometrics/internal/repository"
	"github.com/LifeforDream/gometrics/internal/router"
	"github.com/LifeforDream/gometrics/internal/service"
)

func main() {
	parseFlags()
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	log.Printf("Running server on %s", serverOptions.runAddr)
	svc := service.NewMetricService(&repository.MemStorage{})
	h := handler.NewHandler(svc)
	return http.ListenAndServe(serverOptions.runAddr, router.MetricsRouter(h))
}
