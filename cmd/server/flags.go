package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/caarlos0/env/v11"

	"github.com/LifeforDream/gometrics/internal/config"
)

// ServerOptions — конфигурация сервера, разбираемая parseOptions: сначала
// флаги командной строки, затем переменные окружения поверх них (env имеет
// приоритет).
type ServerOptions struct {
	// адрес и порт сервера; флаг -a, по умолчанию "localhost:8080"
	RunAddr string `env:"ADDRESS"`
	// уровень логирования; флаг -l, по умолчанию "info"
	LogLevel string `env:"LOG_LEVEL"`
	// StoreInterval — интервал сохранения метрик на диск в секундах; флаг -i,
	// по умолчанию 300. Если <= 0, запись синхронна при каждом обновлении.
	StoreInterval  int    `env:"STORE_INTERVAL"`
	FileStorePath  string `env:"FILE_STORAGE_PATH"` // путь к файлу хранения метрик; флаг -f, по умолчанию ""
	ToRestore      bool   `env:"RESTORE"`           // восстанавливать ли метрики из файла при старте; флаг -r, по умолчанию true
	DatabaseDsn    string `env:"DATABASE_DSN"`      // DSN подключения к PostgreSQL; флаг -d, по умолчанию "" (тогда используется файловое или memory-хранилище)
	HashKey        string `env:"KEY"`               // ключ HMAC-подписи тела запроса/ответа; флаг -k, по умолчанию ""
	AuditFilePath  string `env:"AUDIT_FILE"`        // путь к файлу для записи аудита; флаг -audit-file, по умолчанию ""
	AuditURL       string `env:"AUDIT_URL"`         // URL для отправки аудита по HTTP; флаг -audit-url, по умолчанию ""
	CryptoKeyPath  string `env:"CRYPTO_KEY"`        // путь к файлу с приватным ключом для расшифровки запросов
	ConfigFilename string `env:"CONFIG"`            // путь к файлу с конфигурацией в JSON
}

// fileConfig — конфигурация сервера из JSON-файла: те же поля, что и
// ServerOptions, но как указатели, чтобы отличить "ключ отсутствует в
// файле" от "ключ задан нулевым значением" (например, "restore": false) —
// без этого нельзя корректно решить, должен ли файл переопределять дефолт
// флага. ConfigFilename сюда не входит: файл конфигурации не может указывать сам на себя.
type fileConfig struct {
	RunAddr       *string `json:"address"`
	LogLevel      *string `json:"log_level"`
	StoreInterval *int    `json:"store_interval"`
	FileStorePath *string `json:"store_file"`
	ToRestore     *bool   `json:"restore"`
	DatabaseDsn   *string `json:"database_dsn"`
	HashKey       *string `json:"key"`
	AuditFilePath *string `json:"audit_file"`
	AuditURL      *string `json:"audit_url"`
	CryptoKeyPath *string `json:"crypto_key"`
}

func parseOptions(args ...string) (*ServerOptions, error) {
	var serverOptions ServerOptions

	if args == nil {
		args = os.Args[1:]
	}

	// Путь к файлу конфигурации нужно определить до объявления остальных
	// флагов: ADDRESS/RESTORE и т.д. ещё не зарегистрированы на этом этапе,
	// поэтому полноценный flag.FlagSet использовать нельзя.
	var cfg fileConfig
	if path := config.ResolveConfigPath(args); path != "" {
		loaded, err := config.ReadConfigFile[fileConfig](path)
		if err != nil {
			return nil, fmt.Errorf("error reading config from file: %w", err)
		}
		cfg = *loaded
	}

	fs := flag.NewFlagSet("server", flag.ContinueOnError)

	fs.StringVar(&serverOptions.RunAddr, "a", config.StrOr(cfg.RunAddr, "localhost:8080"), "address and port to run server")
	fs.StringVar(&serverOptions.LogLevel, "l", config.StrOr(cfg.LogLevel, "info"), "log level")
	fs.IntVar(&serverOptions.StoreInterval, "i", config.IntOr(cfg.StoreInterval, 300), "interval to store current values on disk")
	fs.StringVar(&serverOptions.FileStorePath, "f", config.StrOr(cfg.FileStorePath, ""), "path to metrics storage on disk")
	fs.BoolVar(&serverOptions.ToRestore, "r", config.BoolOr(cfg.ToRestore, true), "signal to restore metrics values from disk")
	fs.StringVar(&serverOptions.DatabaseDsn, "d", config.StrOr(cfg.DatabaseDsn, ""), "connection string to connect to database")
	fs.StringVar(&serverOptions.HashKey, "k", config.StrOr(cfg.HashKey, ""), "hash key")
	fs.StringVar(&serverOptions.AuditFilePath, "audit-file", config.StrOr(cfg.AuditFilePath, ""), "filepath to save audit logs to")
	fs.StringVar(&serverOptions.AuditURL, "audit-url", config.StrOr(cfg.AuditURL, ""), "url to send audit logs to")
	fs.StringVar(&serverOptions.CryptoKeyPath, "crypto-key", config.StrOr(cfg.CryptoKeyPath, ""), "filepath to a private key storage")
	fs.StringVar(&serverOptions.ConfigFilename, "c", "", "filepath to .json file with configuration options")
	fs.StringVar(&serverOptions.ConfigFilename, "config", "", "same as -c")

	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("error parsing args: %w", err)
	}

	err := env.Parse(&serverOptions)
	if err != nil {
		return nil, fmt.Errorf("error reading config from env: %w", err)
	}

	return &serverOptions, nil
}
