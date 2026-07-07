package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseOptions(t *testing.T) {
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
			},
			envParams: map[string]string{
				"ADDRESS":           "localhost:8082",
				"LOG_LEVEL":         "warn",
				"STORE_INTERVAL":    "150",
				"FILE_STORAGE_PATH": "file.txt",
				"RESTORE":           "t",
				"DATABASE_DSN":      "postgres://a:a@localhost/a",
				"KEY":               "secret",
			},
			expected: ServerOptions{
				RunAddr:       "localhost:8082",
				LogLevel:      "warn",
				StoreInterval: 150,
				FileStorePath: "file.txt",
				ToRestore:     true,
				DatabaseDsn:   "postgres://a:a@localhost/a",
				HashKey:       "secret",
			},
		},
		{
			name:      "envs don't overwrite when empty",
			args:      []string{"-a", "localhost:8085"},
			envParams: map[string]string{},
			expected: ServerOptions{
				RunAddr:       "localhost:8085",
				LogLevel:      "info",
				StoreInterval: 300,
				FileStorePath: "",
				ToRestore:     true,
				DatabaseDsn:   "",
				HashKey:       "",
			},
		},
		{
			name:      "envs write when empty parameter",
			args:      []string{},
			envParams: map[string]string{"ADDRESS": "localhost:8082"},
			expected: ServerOptions{
				RunAddr:       "localhost:8082",
				LogLevel:      "info",
				StoreInterval: 300,
				FileStorePath: "",
				ToRestore:     true,
				DatabaseDsn:   "",
				HashKey:       "",
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
		})
	}
}
