package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/LifeforDream/gometrics/internal/handler"
	"github.com/LifeforDream/gometrics/internal/logging"
	"github.com/LifeforDream/gometrics/internal/middlewares/compress"
	"github.com/LifeforDream/gometrics/internal/middlewares/logs"
	"github.com/LifeforDream/gometrics/internal/repository"
	"github.com/LifeforDream/gometrics/internal/router"
	"github.com/LifeforDream/gometrics/internal/service"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
)

func main() {
	serverOptions, err := parseOptions()
	if err != nil {
		log.Fatal(err)
	}
	logger, err := logging.Initialize(serverOptions.LogLevel)
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	var (
		repo service.MetricRepo
		db   *sql.DB
	)

	// choose storage
	if serverOptions.DatabaseDsn != "" {
		db, err = sql.Open("pgx", serverOptions.DatabaseDsn)
		if err != nil {
			log.Fatal(err)
		}

		performMigrate(serverOptions.DatabaseDsn, logger)

		repo = repository.NewDbStorage(db)

	} else if serverOptions.FileStorePath != "" {
		frepo, err := repository.NewFileStorage(serverOptions.FileStorePath, serverOptions.StoreInterval, serverOptions.ToRestore)
		if err != nil {
			log.Fatal(err)
		}
		repo = frepo
		go repository.SaveMetricsJob(ctx, serverOptions.StoreInterval, frepo, logger)
	} else {
		repo = repository.NewMemStorage()
	}

	svc := service.NewMetricService(repo)
	h := handler.NewHandler(svc, logger)
	srv := &http.Server{
		Addr:    serverOptions.RunAddr,
		Handler: router.MetricsRouter(h, logs.WithLogging(logger), middleware.StripSlashes, compress.Compress(logger)),
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("Running server", zap.String("address", serverOptions.RunAddr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	select {
	case <-ctx.Done(): //SIGINT
	case err := <-serverErr:
		logger.Fatal("Failed to start application", zap.Error(err))
	}

	// after SIGINT we give server 5 seconds to cleanup
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = srv.Shutdown(shutdownCtx)
	if err != nil {
		logger.Fatal("Failed to gracefully shutdown the server", zap.Error(err))
	}

	err = repo.Close()
	if err != nil {
		logger.Error("Error closing repo", zap.Error(err))
	}
}
