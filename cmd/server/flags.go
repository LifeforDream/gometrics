package main

import (
	"flag"
	"os"

	"github.com/caarlos0/env/v11"
)

// ServerOptions — конфигурация сервера, разбираемая parseOptions: сначала
// флаги командной строки, затем переменные окружения поверх них (env имеет
// приоритет).
type ServerOptions struct {
	RunAddr  string `env:"ADDRESS"`   // адрес и порт сервера; флаг -a, по умолчанию "localhost:8080"
	LogLevel string `env:"LOG_LEVEL"` // уровень логирования; флаг -l, по умолчанию "info"
	// StoreInterval — интервал сохранения метрик на диск в секундах; флаг -i,
	// по умолчанию 300. Если <= 0, запись синхронна при каждом обновлении.
	StoreInterval int    `env:"STORE_INTERVAL"`
	FileStorePath string `env:"FILE_STORAGE_PATH"` // путь к файлу хранения метрик; флаг -f, по умолчанию ""
	ToRestore     bool   `env:"RESTORE"`            // восстанавливать ли метрики из файла при старте; флаг -r, по умолчанию true
	DatabaseDsn   string `env:"DATABASE_DSN"`       // DSN подключения к PostgreSQL; флаг -d, по умолчанию "" (тогда используется файловое или memory-хранилище)
	HashKey       string `env:"KEY"`                // ключ HMAC-подписи тела запроса/ответа; флаг -k, по умолчанию ""
	AuditFilePath string `env:"AUDIT_FILE"`         // путь к файлу для записи аудита; флаг -audit-file, по умолчанию ""
	AuditURL      string `env:"AUDIT_URL"`          // URL для отправки аудита по HTTP; флаг -audit-url, по умолчанию ""
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
