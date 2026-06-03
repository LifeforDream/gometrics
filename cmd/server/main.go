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
	serverOptions, err := parseOptions()
	if err != nil {
		log.Fatal(err)
	}
	if err := run(serverOptions); err != nil {
		log.Fatal(err)
	}
}

func run(serverOptions *ServerOptions) error {
	log.Printf("Running server on %s", serverOptions.RunAddr)
	svc := service.NewMetricService(repository.NewMemStorage())
	h := handler.NewHandler(svc)
	return http.ListenAndServe(serverOptions.RunAddr, router.MetricsRouter(h))
}
