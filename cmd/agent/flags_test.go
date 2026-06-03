package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
	tests := []struct {
		name      string
		args      []string
		envParams map[string]string
		expected  AgentOptions
	}{
		{
			name:      "all envs overwrite flags",
			args:      []string{"-a", "localhost:8080", "--secure", "-p", "2", "-r", "10"},
			envParams: map[string]string{"ADDRESS": "localhost:8082", "POLL_INTERVAL": "3", "REPORT_INTERVAL": "11"},
			expected:  AgentOptions{"localhost:8082", true, 3, 11},
		},
		{
			name:      "envs overwrite some parameter",
			args:      []string{"-a", "localhost:8080", "--secure", "-p", "1", "-r", "2"},
			envParams: map[string]string{"ADDRESS": "localhost:8082"},
			expected:  AgentOptions{"localhost:8082", true, 1, 2},
		},
		{
			name:      "envs don't overwrite when empty",
			args:      []string{"-a", "localhost:8080", "--secure", "-p", "1", "-r", "2"},
			envParams: map[string]string{},
			expected:  AgentOptions{"localhost:8080", true, 1, 2},
		},
		{
			name:      "envs write when empty parameter",
			args:      []string{},
			envParams: map[string]string{"ADDRESS": "localhost:8082", "POLL_INTERVAL": "3", "REPORT_INTERVAL": "11"},
			expected:  AgentOptions{"localhost:8082", false, 3, 11},
		},
		{
			name:      "envs don't overwrite when empty parameter and env",
			args:      []string{"-a", "localhost:8080"},
			envParams: map[string]string{},
			expected:  AgentOptions{"localhost:8080", false, 2, 10},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for envName, envVal := range tt.envParams {
				t.Setenv(envName, envVal)
			}
			result, err := parseOptions(tt.args...)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected.Address, result.Address)
			assert.Equal(t, tt.expected.PollInterval, result.PollInterval)
			assert.Equal(t, tt.expected.ReportInterval, result.ReportInterval)
			assert.Equal(t, tt.expected.Secure, result.Secure)
		})
	}
}
