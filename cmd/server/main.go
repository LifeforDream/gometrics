package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/LifeforDream/gometrics/internal/audit"
	"github.com/LifeforDream/gometrics/internal/handler"
	"github.com/LifeforDream/gometrics/internal/logging"
	"github.com/LifeforDream/gometrics/internal/middlewares/logs"
	"github.com/LifeforDream/gometrics/internal/middlewares/mwcompress"
	"github.com/LifeforDream/gometrics/internal/middlewares/mwhash"
	"github.com/LifeforDream/gometrics/internal/middlewares/mwip"
	"github.com/LifeforDream/gometrics/internal/repository"
	"github.com/LifeforDream/gometrics/internal/router"
	"github.com/LifeforDream/gometrics/internal/service"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
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
	)

	// choose storage
	if serverOptions.DatabaseDsn != "" {
		pool, err := pgxpool.New(ctx, serverOptions.DatabaseDsn)
		if err != nil {
			logger.Fatal("Error opening connection to db", zap.Error(err))
		}

		repo, err = repository.NewDbStorage(ctx, pool, logger)
		if err != nil {
			logger.Fatal("Error initializing db storage", zap.Error(err))
		}

	} else if serverOptions.FileStorePath != "" {
		frepo, err := repository.NewFileStorage(serverOptions.FileStorePath, serverOptions.StoreInterval, serverOptions.ToRestore)
		if err != nil {
			logger.Fatal("Error creating file storage", zap.Error(err))
		}
		repo = frepo
		go repository.SaveMetricsJob(ctx, serverOptions.StoreInterval, frepo, logger)
	} else {
		repo = repository.NewMemStorage()
	}

	auditor := audit.NewAuditor(logger)
	if serverOptions.AuditFilePath != "" {
		fasender, err := audit.NewFileAuditSender(serverOptions.AuditFilePath, logger)
		if err != nil {
			logger.Warn("error opening file for audit writing", zap.Error(err))
		} else {
			auditor.RegisterSub(fasender)
		}
	}
	if serverOptions.AuditURL != "" {
		httpsender, err := audit.NewHTTPAuditSender(serverOptions.AuditURL, logger)
		if err != nil {
			logger.Warn("error creating sender for audit http sender", zap.Error(err))
		} else {
			auditor.RegisterSub(httpsender)
		}
	}

	svc := service.NewMetricService(repo, auditor)
	h := handler.NewHandler(svc, logger)
	srv := &http.Server{
		Addr:    serverOptions.RunAddr,
		Handler: router.MetricsRouter(h, logs.WithLogging(logger), mwip.WithClientIP, mwhash.WithHash(serverOptions.HashKey, logger), middleware.StripSlashes, mwcompress.Compress(logger)),
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
	auditor.Close()
}
