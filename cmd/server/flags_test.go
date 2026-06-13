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
			},
			envParams: map[string]string{
				"ADDRESS":           "localhost:8082",
				"LOG_LEVEL":         "warn",
				"STORE_INTERVAL":    "150",
				"FILE_STORAGE_PATH": "file.txt",
				"RESTORE":           "t",
			},
			expected: ServerOptions{
				RunAddr:       "localhost:8082",
				LogLevel:      "warn",
				StoreInterval: 150,
				FileStorePath: "file.txt",
				ToRestore:     true,
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
				FileStorePath: "metrics.json",
				ToRestore:     true,
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
				FileStorePath: "metrics.json",
				ToRestore:     true,
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
				FileStorePath: "metrics.json",
				ToRestore:     true,
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
		})
	}
}
