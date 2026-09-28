package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCertGenOptions(t *testing.T) {
	curDir, err := os.Getwd()
	if err != nil {
		t.Fatal("cannot get current workdir", err)
	}

	tests := []struct {
		name      string
		args      []string
		envParams map[string]string
		expected  CertgenOptions
	}{
		{
			name:      "default values",
			args:      []string{},
			envParams: map[string]string{},
			expected:  CertgenOptions{Bits: 4096, OutDir: curDir},
		},
		{
			name:      "explicit args",
			args:      []string{"-b", "1024", "-o", "/tmp"},
			envParams: map[string]string{},
			expected:  CertgenOptions{Bits: 1024, OutDir: "/tmp"},
		},
		{
			name:      "only envs",
			args:      []string{},
			envParams: map[string]string{"OUT_DIR": "/tmp", "BITS": "1024"},
			expected:  CertgenOptions{Bits: 1024, OutDir: "/tmp"},
		},
		{
			name:      "envs overwrite args",
			args:      []string{"-b", "1024", "-o", "/tmp"},
			envParams: map[string]string{"OUT_DIR": "/envdir", "BITS": "2048"},
			expected:  CertgenOptions{Bits: 2048, OutDir: "/envdir"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for envName, envVal := range tt.envParams {
				t.Setenv(envName, envVal)
			}
			result, err := parseOptions(tt.args...)

			assert.NoError(t, err)
			assert.Equal(t, tt.expected.Bits, result.Bits)
			assert.Equal(t, tt.expected.OutDir, result.OutDir)

		})
	}
}
