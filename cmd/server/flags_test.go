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
			name:      "all envs overwrite flags",
			args:      []string{"-a", "localhost:8085"},
			envParams: map[string]string{"ADDRESS": "localhost:8082"},
			expected:  ServerOptions{"localhost:8082"},
		},
		{
			name:      "envs don't overwrite when empty",
			args:      []string{"-a", "localhost:8085"},
			envParams: map[string]string{},
			expected:  ServerOptions{"localhost:8085"},
		},
		{
			name:      "envs write when empty parameter",
			args:      []string{},
			envParams: map[string]string{"ADDRESS": "localhost:8082"},
			expected:  ServerOptions{"localhost:8082"},
		},
		{
			name:      "use defaults",
			args:      []string{},
			envParams: map[string]string{},
			expected:  ServerOptions{"localhost:8080"},
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
		})
	}
}
