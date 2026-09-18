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

	var cfg fileConfig
	if path := config.ResolveConfigPath(args); path != "" {
		loaded, err := config.ReadConfigFile[fileConfig](path)
		if err != nil {
			return nil, fmt.Errorf("error reading config from file: %w", err)
		}
		cfg = *loaded
	}

	fs := flag.NewFlagSet("agent", flag.ContinueOnError)

	fs.StringVar(&agentOptions.Address, "a", config.StrOr(cfg.Address, "localhost:8080"), "server address")
	fs.BoolVar(&agentOptions.Secure, "secure", config.BoolOr(cfg.Secure, false), "flag to indicate usage of secure channel")
	fs.IntVar(&agentOptions.PollInterval, "p", config.IntOr(cfg.PollInterval, 2), "poll interval in seconds")
	fs.IntVar(&agentOptions.ReportInterval, "r", config.IntOr(cfg.ReportInterval, 10), "report interval in seconds")
	fs.StringVar(&agentOptions.HashKey, "k", config.StrOr(cfg.HashKey, ""), "hash key")
	fs.IntVar(&agentOptions.ConcurrentRequests, "l", config.IntOr(cfg.ConcurrentRequests, 1), "max number of concurrent requests to server")
	fs.StringVar(&agentOptions.CryptoKeyPath, "crypto-key", config.StrOr(cfg.CryptoKeyPath, ""), "filepath to a public key storage")
	fs.StringVar(&agentOptions.ConfigFilename, "c", "", "filepath to .json file with configuration options")
	fs.StringVar(&agentOptions.ConfigFilename, "config", "", "same as -c")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	err := env.Parse(&agentOptions)
	if err != nil {
		return nil, err
	}

	if agentOptions.ConcurrentRequests < 1 {
		return nil, fmt.Errorf("invalid ratelimit number, cannot send metrics %d", agentOptions.ConcurrentRequests)
	}

	return &agentOptions, nil
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
