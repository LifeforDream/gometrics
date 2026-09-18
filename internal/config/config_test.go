package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testConfig — вспомогательная структура для проверки ReadConfigFile:
// указатели позволяют отличить отсутствующий в файле ключ от нулевого
// значения ("count": 0 или "flag": false).
type testConfig struct {
	Name  *string `json:"name"`
	Count *int    `json:"count"`
	Flag  *bool   `json:"flag"`
}

func writeConfigFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestReadConfigFile(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
		check   func(t *testing.T, cfg *testConfig)
	}{
		{
			name:    "all fields present",
			content: `{"name": "svc", "count": 5, "flag": true}`,
			check: func(t *testing.T, cfg *testConfig) {
				require.NotNil(t, cfg.Name)
				assert.Equal(t, "svc", *cfg.Name)
				require.NotNil(t, cfg.Count)
				assert.Equal(t, 5, *cfg.Count)
				require.NotNil(t, cfg.Flag)
				assert.True(t, *cfg.Flag)
			},
		},
		{
			name:    "zero values are distinguishable from absent keys",
			content: `{"count": 0, "flag": false}`,
			check: func(t *testing.T, cfg *testConfig) {
				assert.Nil(t, cfg.Name)
				require.NotNil(t, cfg.Count)
				assert.Equal(t, 0, *cfg.Count)
				require.NotNil(t, cfg.Flag)
				assert.False(t, *cfg.Flag)
			},
		},
		{
			name:    "partial config leaves other fields nil",
			content: `{"name": "svc"}`,
			check: func(t *testing.T, cfg *testConfig) {
				require.NotNil(t, cfg.Name)
				assert.Equal(t, "svc", *cfg.Name)
				assert.Nil(t, cfg.Count)
				assert.Nil(t, cfg.Flag)
			},
		},
		{
			name:    "empty object leaves everything nil",
			content: `{}`,
			check: func(t *testing.T, cfg *testConfig) {
				assert.Nil(t, cfg.Name)
				assert.Nil(t, cfg.Count)
				assert.Nil(t, cfg.Flag)
			},
		},
		{
			name:    "invalid json returns error",
			content: `{"name": `,
			wantErr: true,
		},
		{
			name:    "wrong json type for field returns error",
			content: `{"count": "not-a-number"}`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeConfigFile(t, tt.content)
			cfg, err := ReadConfigFile[testConfig](path)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			tt.check(t, cfg)
		})
	}
}

func TestReadConfigFileMissingFile(t *testing.T) {
	_, err := ReadConfigFile[testConfig](filepath.Join(t.TempDir(), "does-not-exist.json"))
	require.Error(t, err)
}

func TestResolveConfigPath(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		setEnv   bool
		envValue string
		want     string
	}{
		{
			name: "no flag, no env",
			args: []string{"-a", "localhost:8080"},
			want: "",
		},
		{
			name: "-c with separate value",
			args: []string{"-c", "path/from/c.json"},
			want: "path/from/c.json",
		},
		{
			name: "-c=value form",
			args: []string{"-c=path/from/c-eq.json"},
			want: "path/from/c-eq.json",
		},
		{
			name: "-config with separate value",
			args: []string{"-config", "path/from/config.json"},
			want: "path/from/config.json",
		},
		{
			name: "-config=value form",
			args: []string{"-config=path/from/config-eq.json"},
			want: "path/from/config-eq.json",
		},
		{
			name: "flag found among unrelated flags",
			args: []string{"-a", "localhost:8080", "-c", "path/from/c.json", "-k", "secret"},
			want: "path/from/c.json",
		},
		{
			name:     "env overrides flag",
			args:     []string{"-c", "path/from/c.json"},
			setEnv:   true,
			envValue: "path/from/env.json",
			want:     "path/from/env.json",
		},
		{
			name:     "env used when no flag given",
			args:     []string{},
			setEnv:   true,
			envValue: "path/from/env-only.json",
			want:     "path/from/env-only.json",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				t.Setenv("CONFIG", tt.envValue)
			}
			assert.Equal(t, tt.want, ResolveConfigPath(tt.args))
		})
	}
}

func TestStrOr(t *testing.T) {
	val := "value"
	assert.Equal(t, "value", StrOr(&val, "default"))
	assert.Equal(t, "default", StrOr(nil, "default"))
}

func TestBoolOr(t *testing.T) {
	val := false
	assert.Equal(t, false, BoolOr(&val, true))
	assert.Equal(t, true, BoolOr(nil, true))
}

func TestIntOr(t *testing.T) {
	val := 0
	assert.Equal(t, 0, IntOr(&val, 42))
	assert.Equal(t, 42, IntOr(nil, 42))
}
