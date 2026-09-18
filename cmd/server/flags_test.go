package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseOptions(t *testing.T) {
	// -c/-config/CONFIG реально загружают файл, поэтому в кейсах
	// нужен существующий (пусть и пустой) JSON-файл, а не произвольная строка.
	configPath := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(configPath, []byte("{}"), 0o600))

	tests := []struct {
		name      string
		args      []string
		envParams map[string]string
		expected  ServerOptions
	}{
		{
			name: "all envs overwrite flags",
			args: []string{
				"-a", "localhost:8085",
				"-l", "debug",
				"-i", "350",
				"-f", "m.json",
				"-r", "f",
				"-d", "postgres://u:u@localhost/db",
				"-k", "a",
				"-audit-file", "audit_a.log",
				"-audit-url", "http://a.example/audit",
				"-crypto-key", "/path/from/flag.pem",
				"-c", configPath,
			},
			envParams: map[string]string{
				"ADDRESS":           "localhost:8082",
				"LOG_LEVEL":         "warn",
				"STORE_INTERVAL":    "150",
				"FILE_STORAGE_PATH": "file.txt",
				"RESTORE":           "t",
				"DATABASE_DSN":      "postgres://a:a@localhost/a",
				"KEY":               "secret",
				"AUDIT_FILE":        "audit_b.log",
				"AUDIT_URL":         "http://b.example/audit",
				"CRYPTO_KEY":        "/path/from/env.pem",
				"CONFIG":            configPath,
			},
			expected: ServerOptions{
				RunAddr:        "localhost:8082",
				LogLevel:       "warn",
				StoreInterval:  150,
				FileStorePath:  "file.txt",
				ToRestore:      true,
				DatabaseDsn:    "postgres://a:a@localhost/a",
				HashKey:        "secret",
				AuditFilePath:  "audit_b.log",
				AuditURL:       "http://b.example/audit",
				CryptoKeyPath:  "/path/from/env.pem",
				ConfigFilename: configPath,
			},
		},
		{
			name:      "envs don't overwrite when empty",
			args:      []string{"-a", "localhost:8085", "-crypto-key", "/path/to/private.pem", "-config", configPath},
			envParams: map[string]string{},
			expected: ServerOptions{
				RunAddr:        "localhost:8085",
				LogLevel:       "info",
				StoreInterval:  300,
				FileStorePath:  "",
				ToRestore:      true,
				DatabaseDsn:    "",
				HashKey:        "",
				CryptoKeyPath:  "/path/to/private.pem",
				ConfigFilename: configPath,
			},
		},
		{
			name:      "envs write when empty parameter",
			args:      []string{},
			envParams: map[string]string{"ADDRESS": "localhost:8082", "CONFIG": configPath},
			expected: ServerOptions{
				RunAddr:        "localhost:8082",
				LogLevel:       "info",
				StoreInterval:  300,
				FileStorePath:  "",
				ToRestore:      true,
				DatabaseDsn:    "",
				HashKey:        "",
				ConfigFilename: configPath,
			},
		},
		{
			name:      "db flag only",
			args:      []string{"-d", "postgres://u:u@localhost/db"},
			envParams: map[string]string{},
			expected: ServerOptions{
				RunAddr:       "localhost:8080",
				LogLevel:      "info",
				StoreInterval: 300,
				FileStorePath: "",
				ToRestore:     true,
				DatabaseDsn:   "postgres://u:u@localhost/db",
				HashKey:       "",
			},
		},
		{
			name:      "use defaults",
			args:      []string{},
			envParams: map[string]string{},
			expected: ServerOptions{
				RunAddr:       "localhost:8080",
				LogLevel:      "info",
				StoreInterval: 300,
				FileStorePath: "",
				ToRestore:     true,
				DatabaseDsn:   "",
				HashKey:       "",
			},
		},
		{
			name: "audit flags only, no env overrides",
			args: []string{
				"-audit-file", "audit.log",
				"-audit-url", "http://localhost:9000/audit",
			},
			envParams: map[string]string{},
			expected: ServerOptions{
				RunAddr:       "localhost:8080",
				LogLevel:      "info",
				StoreInterval: 300,
				ToRestore:     true,
				AuditFilePath: "audit.log",
				AuditURL:      "http://localhost:9000/audit",
			},
		},
		{
			name:      "audit disabled by default",
			args:      []string{},
			envParams: map[string]string{},
			expected: ServerOptions{
				RunAddr:       "localhost:8080",
				LogLevel:      "info",
				StoreInterval: 300,
				ToRestore:     true,
				AuditFilePath: "",
				AuditURL:      "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for envName, envVal := range tt.envParams {
				t.Setenv(envName, envVal)
			}
			result, err := parseOptions(tt.args...)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected.RunAddr, result.RunAddr)
			assert.Equal(t, tt.expected.FileStorePath, result.FileStorePath)
			assert.Equal(t, tt.expected.LogLevel, result.LogLevel)
			assert.Equal(t, tt.expected.StoreInterval, result.StoreInterval)
			assert.Equal(t, tt.expected.ToRestore, result.ToRestore)
			assert.Equal(t, tt.expected.DatabaseDsn, result.DatabaseDsn)
			assert.Equal(t, tt.expected.HashKey, result.HashKey)
			assert.Equal(t, tt.expected.AuditFilePath, result.AuditFilePath)
			assert.Equal(t, tt.expected.AuditURL, result.AuditURL)
			assert.Equal(t, tt.expected.CryptoKeyPath, result.CryptoKeyPath)
			assert.Equal(t, tt.expected.ConfigFilename, result.ConfigFilename)
		})
	}
}

func writeServerConfigFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestParseOptionsAppliesConfigFile(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		args      []string
		envParams map[string]string
		expected  ServerOptions
	}{
		{
			name: "config values fill in when no flags or envs set",
			content: `{
				"address": "localhost:9090",
				"restore": false,
				"store_interval": 5,
				"store_file": "/tmp/config-file.db",
				"database_dsn": "postgres://cfg",
				"crypto_key": "/path/from/config.pem"
			}`,
			expected: ServerOptions{
				RunAddr:       "localhost:9090",
				LogLevel:      "info",
				StoreInterval: 5,
				FileStorePath: "/tmp/config-file.db",
				ToRestore:     false,
				DatabaseDsn:   "postgres://cfg",
				CryptoKeyPath: "/path/from/config.pem",
			},
		},
		{
			name:    "explicit flag overrides config for that field only",
			content: `{"address": "localhost:9090", "database_dsn": "postgres://cfg"}`,
			args:    []string{"-a", "localhost:7000"},
			expected: ServerOptions{
				RunAddr:       "localhost:7000",
				LogLevel:      "info",
				DatabaseDsn:   "postgres://cfg",
				ToRestore:     true,
				StoreInterval: 300,
			},
		},
		{
			name:      "env overrides config for that field only",
			content:   `{"address": "localhost:9090", "database_dsn": "postgres://cfg"}`,
			envParams: map[string]string{"ADDRESS": "localhost:7001"},
			expected: ServerOptions{
				RunAddr:       "localhost:7001",
				LogLevel:      "info",
				DatabaseDsn:   "postgres://cfg",
				ToRestore:     true,
				StoreInterval: 300,
			},
		},
		{
			name:    "explicit flag equal to default still wins over config",
			content: `{"restore": false}`,
			args:    []string{"-r=true"},
			expected: ServerOptions{
				RunAddr:       "localhost:8080",
				LogLevel:      "info",
				StoreInterval: 300,
				ToRestore:     true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeServerConfigFile(t, tt.content)
			for envName, envVal := range tt.envParams {
				t.Setenv(envName, envVal)
			}
			args := append(append([]string{}, tt.args...), "-c", path)
			result, err := parseOptions(args...)
			require.NoError(t, err)
			assert.Equal(t, tt.expected.RunAddr, result.RunAddr)
			assert.Equal(t, tt.expected.LogLevel, result.LogLevel)
			assert.Equal(t, tt.expected.StoreInterval, result.StoreInterval)
			assert.Equal(t, tt.expected.FileStorePath, result.FileStorePath)
			assert.Equal(t, tt.expected.ToRestore, result.ToRestore)
			assert.Equal(t, tt.expected.DatabaseDsn, result.DatabaseDsn)
			assert.Equal(t, tt.expected.CryptoKeyPath, result.CryptoKeyPath)
		})
	}
}

func TestParseOptionsConfigFileViaEnv(t *testing.T) {
	path := writeServerConfigFile(t, `{"address": "localhost:9095"}`)
	t.Setenv("CONFIG", path)
	result, err := parseOptions([]string{}...)
	require.NoError(t, err)
	assert.Equal(t, "localhost:9095", result.RunAddr)
}

func TestParseOptionsConfigFileNotFound(t *testing.T) {
	_, err := parseOptions("-c", filepath.Join(t.TempDir(), "missing.json"))
	require.Error(t, err)
}

func TestParseOptionsNoConfigFlagSkipsLoading(t *testing.T) {
	result, err := parseOptions([]string{}...)
	require.NoError(t, err)
	assert.Equal(t, "localhost:8080", result.RunAddr)
}
