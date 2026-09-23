package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConstructAddress(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		address string
		secure  bool
		want    string
	}{
		{
			name:    "bare host, insecure",
			address: "localhost:8080",
			secure:  false,
			want:    "http://localhost:8080",
		},
		{
			name:    "bare host, secure",
			address: "localhost:8080",
			secure:  true,
			want:    "https://localhost:8080",
		},
		{
			name:    "already has http scheme",
			address: "http://localhost:8080",
			secure:  false,
			want:    "http://localhost:8080",
		},
		{
			name:    "already has https scheme",
			address: "https://localhost:8080",
			secure:  false,
			want:    "https://localhost:8080",
		},
		{
			name:    "uppercase scheme is preserved as-is",
			address: "HTTP://localhost:8080",
			secure:  false,
			want:    "HTTP://localhost:8080",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var agentOptions AgentOptions

			agentOptions.Address = tt.address
			agentOptions.Secure = tt.secure
			got := constructAddress(&agentOptions)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEnvFlagOrder(t *testing.T) {
	// -c/-config/CONFIG реально загружают файл, поэтому в кейсах
	// нужен существующий (пусть и пустой) JSON-файл, а не произвольная строка.
	configPath := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(configPath, []byte("{}"), 0o600))

	tests := []struct {
		name      string
		args      []string
		envParams map[string]string
		expected  AgentOptions
		wantErr   bool
	}{
		{
			name:      "all envs overwrite flags",
			args:      []string{"-a", "localhost:8080", "--secure", "-p", "2", "-r", "10", "-k", "sec", "-l", "1", "-crypto-key", "/path/from/flag.pem", "-c", configPath},
			envParams: map[string]string{"ADDRESS": "localhost:8082", "POLL_INTERVAL": "3", "REPORT_INTERVAL": "11", "KEY": "secret", "RATE_LIMIT": "2", "CRYPTO_KEY": "/path/from/env.pem", "CONFIG": configPath},
			expected: AgentOptions{
				Address:            "localhost:8082",
				Secure:             true,
				PollInterval:       3,
				ReportInterval:     11,
				HashKey:            "secret",
				ConcurrentRequests: 2,
				CryptoKeyPath:      "/path/from/env.pem",
				ConfigFilename:     configPath,
			},
			wantErr: false,
		},
		{
			name:      "envs overwrite some parameter",
			args:      []string{"-a", "localhost:8080", "--secure", "-p", "1", "-r", "2"},
			envParams: map[string]string{"ADDRESS": "localhost:8082"},
			expected: AgentOptions{
				Address:            "localhost:8082",
				Secure:             true,
				PollInterval:       1,
				ReportInterval:     2,
				HashKey:            "",
				ConcurrentRequests: 1,
			},
			wantErr: false,
		},
		{
			name:      "envs don't overwrite when empty",
			args:      []string{"-a", "localhost:8080", "--secure", "-p", "1", "-r", "2", "-crypto-key", "/path/to/cert.pem", "-config", configPath},
			envParams: map[string]string{},
			expected: AgentOptions{
				Address:            "localhost:8080",
				Secure:             true,
				PollInterval:       1,
				ReportInterval:     2,
				HashKey:            "",
				ConcurrentRequests: 1,
				CryptoKeyPath:      "/path/to/cert.pem",
				ConfigFilename:     configPath,
			},
			wantErr: false,
		},
		{
			name:      "envs write when empty parameter",
			args:      []string{},
			envParams: map[string]string{"ADDRESS": "localhost:8082", "POLL_INTERVAL": "3", "REPORT_INTERVAL": "11", "CONFIG": configPath},
			expected: AgentOptions{
				Address:            "localhost:8082",
				Secure:             false,
				PollInterval:       3,
				ReportInterval:     11,
				HashKey:            "",
				ConcurrentRequests: 1,
				ConfigFilename:     configPath,
			},
			wantErr: false,
		},
		{
			name:      "envs don't overwrite when empty parameter and env",
			args:      []string{"-a", "localhost:8080"},
			envParams: map[string]string{},
			expected: AgentOptions{
				Address:            "localhost:8080",
				Secure:             false,
				PollInterval:       2,
				ReportInterval:     10,
				HashKey:            "",
				ConcurrentRequests: 1,
			},
			wantErr: false,
		},
		{
			name: "invalid ratelimit number",
			args: []string{"-l", "0"},
			envParams: map[string]string{
				"RATE_LIMIT": "0",
			},
			expected: AgentOptions{},
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for envName, envVal := range tt.envParams {
				t.Setenv(envName, envVal)
			}
			result, err := parseOptions(tt.args...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected.Address, result.Address)
			assert.Equal(t, tt.expected.PollInterval, result.PollInterval)
			assert.Equal(t, tt.expected.ReportInterval, result.ReportInterval)
			assert.Equal(t, tt.expected.Secure, result.Secure)
			assert.Equal(t, tt.expected.HashKey, result.HashKey)
			assert.Equal(t, tt.expected.ConcurrentRequests, result.ConcurrentRequests)
			assert.Equal(t, tt.expected.CryptoKeyPath, result.CryptoKeyPath)
			assert.Equal(t, tt.expected.ConfigFilename, result.ConfigFilename)

		})
	}
}

func writeAgentConfigFile(t *testing.T, content string) string {
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
		expected  AgentOptions
	}{
		{
			name: "config values fill in when no flags or envs set",
			content: `{
				"address": "localhost:9090",
				"report_interval": 3,
				"poll_interval": 1,
				"crypto_key": "/path/from/config.pem"
			}`,
			expected: AgentOptions{
				Address:            "localhost:9090",
				PollInterval:       1,
				ReportInterval:     3,
				ConcurrentRequests: 1,
				CryptoKeyPath:      "/path/from/config.pem",
			},
		},
		{
			name:    "explicit flag overrides config for that field only",
			content: `{"address": "localhost:9090", "crypto_key": "/path/from/config.pem"}`,
			args:    []string{"-a", "localhost:7000"},
			expected: AgentOptions{
				Address:            "localhost:7000",
				PollInterval:       2,
				ReportInterval:     10,
				ConcurrentRequests: 1,
				CryptoKeyPath:      "/path/from/config.pem",
			},
		},
		{
			name:      "env overrides config for that field only",
			content:   `{"address": "localhost:9090", "crypto_key": "/path/from/config.pem"}`,
			envParams: map[string]string{"ADDRESS": "localhost:7001"},
			expected: AgentOptions{
				Address:            "localhost:7001",
				PollInterval:       2,
				ReportInterval:     10,
				ConcurrentRequests: 1,
				CryptoKeyPath:      "/path/from/config.pem",
			},
		},
		{
			name:    "explicit flag equal to default still wins over config",
			content: `{"poll_interval": 9}`,
			args:    []string{"-p", "2"},
			expected: AgentOptions{
				Address:            "localhost:8080",
				PollInterval:       2,
				ReportInterval:     10,
				ConcurrentRequests: 1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeAgentConfigFile(t, tt.content)
			for envName, envVal := range tt.envParams {
				t.Setenv(envName, envVal)
			}
			args := append(append([]string{}, tt.args...), "-c", path)
			result, err := parseOptions(args...)
			require.NoError(t, err)
			assert.Equal(t, tt.expected.Address, result.Address)
			assert.Equal(t, tt.expected.PollInterval, result.PollInterval)
			assert.Equal(t, tt.expected.ReportInterval, result.ReportInterval)
			assert.Equal(t, tt.expected.ConcurrentRequests, result.ConcurrentRequests)
			assert.Equal(t, tt.expected.CryptoKeyPath, result.CryptoKeyPath)
		})
	}
}

func TestParseOptionsConfigFileViaEnv(t *testing.T) {
	path := writeAgentConfigFile(t, `{"address": "localhost:9095"}`)
	t.Setenv("CONFIG", path)
	result, err := parseOptions([]string{}...)
	require.NoError(t, err)
	assert.Equal(t, "localhost:9095", result.Address)
}

func TestParseOptionsConfigFileNotFound(t *testing.T) {
	_, err := parseOptions("-c", filepath.Join(t.TempDir(), "missing.json"))
	require.Error(t, err)
}

func TestParseOptionsNoConfigFlagSkipsLoading(t *testing.T) {
	result, err := parseOptions([]string{}...)
	require.NoError(t, err)
	assert.Equal(t, "localhost:8080", result.Address)
}
