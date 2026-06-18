package main

import (
	"embed"
	"errors"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"go.uber.org/zap"
)

//go:embed migrations
var migrationsFS embed.FS

func performMigrate(dsn string, logger *zap.Logger) {
	d, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		logger.Fatal("Error reading FS", zap.Error(err))
	}
	m, err := migrate.NewWithSourceInstance("iofs", d, dsn)
	if err != nil {
		logger.Fatal("Error creating migrator", zap.Error(err))
	}
	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil {
			logger.Warn("failed to close migration source", zap.Error(srcErr))
		}
		if dbErr != nil {
			logger.Warn("failed to close migration db", zap.Error(dbErr))
		}
	}()
	err = m.Up()
	if err != nil {
		if !errors.Is(err, migrate.ErrNoChange) {
			logger.Fatal("Error performing migration", zap.Error(err))
		}
	}
}
