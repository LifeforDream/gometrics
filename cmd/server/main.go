package main

import (
	"log"
	"net/http"

	"github.com/LifeforDream/gometrics/internal/handler"
	"github.com/LifeforDream/gometrics/internal/middlewares/compress"
	"github.com/LifeforDream/gometrics/internal/middlewares/logger"
	"github.com/LifeforDream/gometrics/internal/repository"
	"github.com/LifeforDream/gometrics/internal/router"
	"github.com/LifeforDream/gometrics/internal/service"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

func main() {
	serverOptions, err := parseOptions()
	if err != nil {
		log.Fatal(err)
	}
	if err := logger.Initialize(serverOptions.LogLevel); err != nil {
		log.Fatal(err)
	}
	if err := run(serverOptions); err != nil {
		logger.Log.Fatal("Failed to start application", zap.Error(err))
	}
}

func run(serverOptions *ServerOptions) error {
	logger.Log.Info("Running server", zap.String("address", serverOptions.RunAddr))
	svc := service.NewMetricService(repository.NewMemStorage())
	h := handler.NewHandler(svc)
	return http.ListenAndServe(serverOptions.RunAddr, router.MetricsRouter(h, logger.WithLogging, middleware.StripSlashes, compress.Compress))
}
