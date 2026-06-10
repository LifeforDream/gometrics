package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

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
	repo, err := repository.NewFileStorage(serverOptions.FileStorePath, serverOptions.StoreInterval, serverOptions.ToRestore)
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go repository.SaveMetricsJob(ctx, serverOptions.StoreInterval, repo)

	svc := service.NewMetricService(repo)
	h := handler.NewHandler(svc)
	srv := &http.Server{Addr: serverOptions.RunAddr, Handler: router.MetricsRouter(h, logger.WithLogging, middleware.StripSlashes, compress.Compress)}

	serverErr := make(chan error, 1)
	go func() {
		logger.Log.Info("Running server", zap.String("address", serverOptions.RunAddr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	select {
	case <-ctx.Done(): //SIGINT
	case err := <-serverErr:
		logger.Log.Fatal("Failed to start application", zap.Error(err))
	}

	// after SIGINT we give server 5 seconds to cleanup
	shutDownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = srv.Shutdown(shutDownCtx)
	if err != nil {
		logger.Log.Fatal("Failed to gracefully shutdown the server", zap.Error(err))
	}
}
