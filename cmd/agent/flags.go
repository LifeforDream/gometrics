package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/caarlos0/env/v11"

	"github.com/LifeforDream/gometrics/internal/config"
)

// AgentOptions — конфигурация агента, разбираемая parseOptions: сначала
// флаги командной строки, затем переменные окружения поверх них (env имеет
// приоритет).
type AgentOptions struct {
	Address string `env:"ADDRESS"` // адрес сервера; флаг -a, по умолчанию "localhost:8080"
	Secure  bool   // флаг -secure: использовать https вместо http, если в Address не указана схема
	// PollInterval — интервал опроса метрик в секундах; флаг -p, по умолчанию 2
	PollInterval int `env:"POLL_INTERVAL"`
	// ReportInterval — интервал отправки метрик на сервер в секундах; флаг -r, по умолчанию 10
	ReportInterval int `env:"REPORT_INTERVAL"`
	// HashKey — ключ HMAC-подписи тела запроса; флаг -k, по умолчанию ""
	HashKey string `env:"KEY"`
	// ConcurrentRequests — максимум одновременных запросов к серверу; флаг -l, по умолчанию 1
	ConcurrentRequests int    `env:"RATE_LIMIT"`
	CryptoKeyPath      string `env:"CRYPTO_KEY"` // путь к файлу с публичным ключом для шифрования запросов
	ConfigFilename     string `env:"CONFIG"`     // путь к файлу с конфигурацией в JSON
}

// fileConfig — конфигурация агента из JSON-файла: те же поля, что и
// AgentOptions, но как указатели, чтобы отличить "ключ отсутствует в
// файле" от "ключ задан нулевым значением" —
// без этого нельзя корректно решить, должен ли файл переопределять дефолт
// флага. ConfigFilename сюда не входит: файл конфигурации не может указывать сам на себя.
type fileConfig struct {
	Address            *string `json:"address"`
	Secure             *bool   `json:"secure"`
	PollInterval       *int    `json:"poll_interval"`
	ReportInterval     *int    `json:"report_interval"`
	HashKey            *string `json:"key"`
	ConcurrentRequests *int    `json:"concurrent_requests"`
	CryptoKeyPath      *string `json:"crypto_key"`
}

func parseOptions(args ...string) (*AgentOptions, error) {
	var agentOptions AgentOptions

	if args == nil {
		args = os.Args[1:]
	}

	fs := flag.NewFlagSet("agent", flag.ContinueOnError)

	fs.StringVar(&agentOptions.Address, "a", "localhost:8080", "server address")
	fs.BoolVar(&agentOptions.Secure, "secure", false, "flag to indicate usage of secure channel")
	fs.IntVar(&agentOptions.PollInterval, "p", 2, "poll interval in seconds")
	fs.IntVar(&agentOptions.ReportInterval, "r", 10, "report interval in seconds")
	fs.StringVar(&agentOptions.HashKey, "k", "", "hash key")
	fs.IntVar(&agentOptions.ConcurrentRequests, "l", 1, "max number of concurrent requests to server")
	fs.StringVar(&agentOptions.CryptoKeyPath, "crypto-key", "", "filepath to a public key storage")
	fs.StringVar(&agentOptions.ConfigFilename, "c", "", "filepath to .json file with configuration options")
	fs.StringVar(&agentOptions.ConfigFilename, "config", "", "same as -c")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	// CONFIG может быть в энве, и его нужно читать в первую очередь
	configPath := agentOptions.ConfigFilename
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

		applyFileConfig(&agentOptions, cfg, explicit)
	}

	if err := env.Parse(&agentOptions); err != nil {
		return nil, err
	}

	if agentOptions.ConcurrentRequests < 1 {
		return nil, fmt.Errorf("invalid ratelimit number, cannot send metrics %d", agentOptions.ConcurrentRequests)
	}

	return &agentOptions, nil
}

// applyFileConfig переносит значения из fileConfig в agentOptions только для
// флагов, не заданных явно. Заданные явно передаются в аргументе explicit.
func applyFileConfig(opts *AgentOptions, cfg *fileConfig, explicit map[string]bool) {
	if cfg.Address != nil && !explicit["a"] {
		opts.Address = *cfg.Address
	}
	if cfg.Secure != nil && !explicit["secure"] {
		opts.Secure = *cfg.Secure
	}
	if cfg.PollInterval != nil && !explicit["p"] {
		opts.PollInterval = *cfg.PollInterval
	}
	if cfg.ReportInterval != nil && !explicit["r"] {
		opts.ReportInterval = *cfg.ReportInterval
	}
	if cfg.HashKey != nil && !explicit["k"] {
		opts.HashKey = *cfg.HashKey
	}
	if cfg.ConcurrentRequests != nil && !explicit["l"] {
		opts.ConcurrentRequests = *cfg.ConcurrentRequests
	}
	if cfg.CryptoKeyPath != nil && !explicit["crypto-key"] {
		opts.CryptoKeyPath = *cfg.CryptoKeyPath
	}
}

func constructAddress(agentOptions *AgentOptions) string {
	var serverAddr string
	// since -a may be without scheme
	// we explicitly check for scheme
	// also url.Parse() does not detect missing scheme
	address := strings.ToLower(agentOptions.Address)
	if strings.HasPrefix(address, "http://") || strings.HasPrefix(address, "https://") {
		serverAddr = agentOptions.Address
	} else {
		if agentOptions.Secure {
			serverAddr = "https://" + agentOptions.Address
		} else {
			serverAddr = "http://" + agentOptions.Address
		}
	}
	return serverAddr
}
