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
	fs.StringVar(&serverOptions.CryptoKeyPath, "crypto-key", "", "filepath to a private key storage")
	fs.StringVar(&serverOptions.ConfigFilename, "c", "", "filepath to .json file with configuration options")
	fs.StringVar(&serverOptions.ConfigFilename, "config", "", "same as -c")

	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("error parsing args: %w", err)
	}

	// CONFIG может быть в энве, и его нужно читать в первую очередь
	configPath := serverOptions.ConfigFilename
	if v, ok := os.LookupEnv("CONFIG"); ok && v != "" {
		configPath = v
	}

	if configPath != "" {
		cfg, err := config.ReadConfigFile[fileConfig](configPath)
		if err != nil {
			return nil, fmt.Errorf("error reading config from file: %w", err)
		}

		explicit := make(map[string]bool)
		fs.Visit(func(f *flag.Flag) { explicit[f.Name] = true })

		applyFileConfig(&serverOptions, cfg, explicit)
	}

	err := env.Parse(&serverOptions)
	if err != nil {
		return nil, fmt.Errorf("error reading config from env: %w", err)
	}

	return &serverOptions, nil
}

// applyFileConfig переносит значения из fileConfig в serverOptions только для
// флагов, не заданных явно. Заданные явно передаются в аргументе explicit.
func applyFileConfig(opts *ServerOptions, cfg *fileConfig, explicit map[string]bool) {
	if cfg.RunAddr != nil && !explicit["a"] {
		opts.RunAddr = *cfg.RunAddr
	}
	if cfg.LogLevel != nil && !explicit["l"] {
		opts.LogLevel = *cfg.LogLevel
	}
	if cfg.StoreInterval != nil && !explicit["i"] {
		opts.StoreInterval = *cfg.StoreInterval
	}
	if cfg.FileStorePath != nil && !explicit["f"] {
		opts.FileStorePath = *cfg.FileStorePath
	}
	if cfg.ToRestore != nil && !explicit["r"] {
		opts.ToRestore = *cfg.ToRestore
	}
	if cfg.DatabaseDsn != nil && !explicit["d"] {
		opts.DatabaseDsn = *cfg.DatabaseDsn
	}
	if cfg.HashKey != nil && !explicit["k"] {
		opts.HashKey = *cfg.HashKey
	}
	if cfg.AuditFilePath != nil && !explicit["audit-file"] {
		opts.AuditFilePath = *cfg.AuditFilePath
	}
	if cfg.AuditURL != nil && !explicit["audit-url"] {
		opts.AuditURL = *cfg.AuditURL
	}
	if cfg.CryptoKeyPath != nil && !explicit["crypto-key"] {
		opts.CryptoKeyPath = *cfg.CryptoKeyPath
	}
}
