package main

import (
	"flag"
	"os"

	"github.com/caarlos0/env/v11"
)

type ServerOptions struct {
	RunAddr       string `env:"ADDRESS"`
	LogLevel      string `env:"LOG_LEVEL"`
	StoreInterval int    `env:"STORE_INTERVAL"`
	FileStorePath string `env:"FILE_STORAGE_PATH"`
	ToRestore     bool   `env:"RESTORE"`
	DatabaseDsn   string `env:"DATABASE_DSN"`
	HashKey       string `env:"KEY"`
	AuditFilePath string `env:"AUDIT_FILE"`
	AuditURL      string `env:"AUDIT_URL"`
}

func parseOptions(args ...string) (*ServerOptions, error) {
	var serverOptions ServerOptions
	fs := flag.NewFlagSet("server", flag.ContinueOnError)

	fs.StringVar(&serverOptions.RunAddr, "a", "localhost:8080", "address and port to run server")
	fs.StringVar(&serverOptions.LogLevel, "l", "info", "log level")
	fs.IntVar(&serverOptions.StoreInterval, "i", 300, "interval to store current values on disk")
	fs.StringVar(&serverOptions.FileStorePath, "f", "", "path to metrics storage on disk")
	fs.BoolVar(&serverOptions.ToRestore, "r", true, "signal to restore metrics values from disk")
	fs.StringVar(&serverOptions.DatabaseDsn, "d", "", "connection string to connect to database")
	fs.StringVar(&serverOptions.HashKey, "k", "", "hash key")
	fs.StringVar(&serverOptions.AuditFilePath, "audit-file", "", "filepath to save audit logs to")
	fs.StringVar(&serverOptions.AuditURL, "audit-url", "", "url to send audit logs to")

	if args == nil {
		args = os.Args[1:]
	}
	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	err := env.Parse(&serverOptions)
	if err != nil {
		return nil, err
	}
	return &serverOptions, nil
}
