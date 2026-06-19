package repository

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	models "github.com/LifeforDream/gometrics/internal/model"
	myErrors "github.com/LifeforDream/gometrics/internal/model/errors"
	"github.com/LifeforDream/gometrics/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveLoadMetrics(t *testing.T) {
	tests := []struct {
		name    string
		metrics map[string]models.Metrics
	}{
		{
			name: "gauge and counter round-trip",
			metrics: map[string]models.Metrics{
				"Alloc":     {ID: "Alloc", MType: models.Gauge, Value: utils.FloatPtr(t, 1.25)},
				"PollCount": {ID: "PollCount", MType: models.Counter, Delta: utils.IntPtr(t, 42)},
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
				"PollCount": {ID: "PollCount", MType: models.Counter, Delta: utils.IntPtr(t, 5)},
				"Alloc":     {ID: "Alloc", MType: models.Gauge, Value: utils.FloatPtr(t, 2.5)},
			},
			wantMetrics: map[string]models.Metrics{
				"PollCount": {ID: "PollCount", MType: models.Counter, Delta: utils.IntPtr(t, 5)},
				"Alloc":     {ID: "Alloc", MType: models.Gauge, Value: utils.FloatPtr(t, 2.5)},
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

func TestMemUpdateMetrics(t *testing.T) {
	tests := []struct {
		name    string
		setUp   []models.Metrics
		input   []models.Metrics
		wantErr bool
		verify  func(t *testing.T, repo *MemStorage)
	}{
		{
			name: "mixed gauge and counter applied",
			input: []models.Metrics{
				{ID: "alloc", MType: models.Gauge, Value: utils.FloatPtr(t, 1.25)},
				{ID: "pollcount", MType: models.Counter, Delta: utils.IntPtr(t, 3)},
			},
			verify: func(t *testing.T, repo *MemStorage) {
				g, err := repo.GetMetric(context.Background(), "alloc")
				require.NoError(t, err)
				assert.Equal(t, 1.25, *g.Value)

				c, err := repo.GetMetric(context.Background(), "pollcount")
				require.NoError(t, err)
				assert.Equal(t, int64(3), *c.Delta)
			},
		},
		{
			name:  "counter accumulates over existing value",
			setUp: []models.Metrics{{ID: "pollcount", MType: models.Counter, Delta: utils.IntPtr(t, 2)}},
			input: []models.Metrics{
				{ID: "pollcount", MType: models.Counter, Delta: utils.IntPtr(t, 3)},
			},
			verify: func(t *testing.T, repo *MemStorage) {
				c, err := repo.GetMetric(context.Background(), "pollcount")
				require.NoError(t, err)
				assert.Equal(t, int64(5), *c.Delta)
			},
		},
		{
			name:    "unknown metric type returns error",
			input:   []models.Metrics{{ID: "bad", MType: "invalid", Value: utils.FloatPtr(t, 1.0)}},
			wantErr: true,
		},
		{
			name: "error short-circuits: subsequent metrics not applied",
			input: []models.Metrics{
				{ID: "alloc", MType: models.Gauge, Value: utils.FloatPtr(t, 1.25)},
				{ID: "bad", MType: "invalid", Value: utils.FloatPtr(t, 1.0)},
				{ID: "pollcount", MType: models.Counter, Delta: utils.IntPtr(t, 5)},
			},
			wantErr: true,
			verify: func(t *testing.T, repo *MemStorage) {
				_, err := repo.GetMetric(context.Background(), "pollcount")
				assert.ErrorIs(t, err, myErrors.MetricNotFound)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMemStorage()
			for _, m := range tt.setUp {
				switch m.MType {
				case models.Counter:
					require.NoError(t, repo.UpdateCounter(context.Background(), m))
				case models.Gauge:
					require.NoError(t, repo.SetGauge(context.Background(), m))
				}
			}

			err := repo.UpdateMetrics(context.Background(), tt.input)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			if tt.verify != nil {
				tt.verify(t, repo)
			}
		})
	}
}
