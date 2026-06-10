package repository

import (
	"os"
	"path/filepath"
	"testing"

	models "github.com/LifeforDream/gometrics/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var floatPtr = func(f float64) *float64 { return &f }
var intPtr = func(i int64) *int64 { return &i }

func TestSaveLoadMetrics(t *testing.T) {
	tests := []struct {
		name    string
		metrics map[string]models.Metrics
	}{
		{
			name: "gauge and counter round-trip",
			metrics: map[string]models.Metrics{
				"Alloc":     {ID: "Alloc", MType: models.Gauge, Value: floatPtr(1.25)},
				"PollCount": {ID: "PollCount", MType: models.Counter, Delta: intPtr(42)},
			},
		},
		{
			name:    "empty metrics",
			metrics: map[string]models.Metrics{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fname := filepath.Join(t.TempDir(), "metrics.json")

			require.NoError(t, SaveMetrics(fname, tt.metrics))

			repo, err := NewFileStorage(fname, 0, true)
			require.NoError(t, err)
			assert.Equal(t, tt.metrics, repo.GetAll())
		})
	}
}

func TestLoadMetricsEmptyFile(t *testing.T) {
	tests := []struct {
		name       string
		createFile bool
	}{
		{
			name:       "nonexistent file",
			createFile: false,
		},
		{
			name:       "empty file",
			createFile: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fname := filepath.Join(t.TempDir(), "metrics.json")
			if tt.createFile {
				file, err := os.Create(fname)
				require.NoError(t, err)
				defer file.Close()
			}
			repo, err := NewFileStorage(fname, 0, true)
			require.NoError(t, err)
			assert.Empty(t, repo.GetAll())
		})
	}
}

func TestNewFileStorage(t *testing.T) {
	tests := []struct {
		name           string
		restore        bool
		preloadMetrics map[string]models.Metrics
		wantMetrics    map[string]models.Metrics
	}{
		{
			name:        "restore=false starts empty",
			restore:     false,
			wantMetrics: map[string]models.Metrics{},
		},
		{
			name:    "restore=true loads existing file",
			restore: true,
			preloadMetrics: map[string]models.Metrics{
				"PollCount": {ID: "PollCount", MType: models.Counter, Delta: intPtr(5)},
				"Alloc":     {ID: "Alloc", MType: models.Gauge, Value: floatPtr(2.5)},
			},
			wantMetrics: map[string]models.Metrics{
				"PollCount": {ID: "PollCount", MType: models.Counter, Delta: intPtr(5)},
				"Alloc":     {ID: "Alloc", MType: models.Gauge, Value: floatPtr(2.5)},
			},
		},
		{
			name:        "restore=true on nonexistent file starts empty",
			restore:     true,
			wantMetrics: map[string]models.Metrics{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fname := filepath.Join(t.TempDir(), "metrics.json")
			if tt.preloadMetrics != nil {
				require.NoError(t, SaveMetrics(fname, tt.preloadMetrics))
			}
			repo, err := NewFileStorage(fname, 0, tt.restore)
			require.NoError(t, err)
			require.NotNil(t, repo)
			assert.Equal(t, tt.wantMetrics, repo.GetAll())
		})
	}
}
