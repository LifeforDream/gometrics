package service

import (
	"testing"

	repository "github.com/LifeforDream/gometrics/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateGauge(t *testing.T) {

	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		mname   string
		values  []float64
		wantVal float64
	}{
		{
			name:    "valid gauge update",
			mname:   "Alloc",
			values:  []float64{23.5},
			wantVal: 23.5,
		},
		{
			name:    "double gauge update",
			mname:   "Alloc",
			values:  []float64{23.5, 30.0},
			wantVal: 30.0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &repository.MemStorage{}
			s := NewMetricService(repo)
			for _, val := range tt.values {
				s.UpdateGauge(tt.mname, val)
			}
			metric, ok := repo.GetMetric(tt.mname)
			require.True(t, ok, "metric not found in repository")
			require.Equal(t, tt.wantVal, *metric.Value, "unexpected metric value")
			require.Equal(t, "gauge", metric.MType, "unexpected metric type")
		})
	}
}

func TestUpdateCounter(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		mname   string
		values  []int64
		wantVal int64
	}{
		{
			name:    "valid counter update",
			mname:   "PollCount",
			values:  []int64{1},
			wantVal: 1,
		},
		{
			name:    "double counter update",
			mname:   "PollCount",
			values:  []int64{1, 2},
			wantVal: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &repository.MemStorage{}
			s := NewMetricService(repo)
			for _, val := range tt.values {
				s.UpdateCounter(tt.mname, val)
			}
			metric, ok := repo.GetMetric(tt.mname)
			require.True(t, ok, "metric not found in repository")
			require.Equal(t, float64(tt.wantVal), *metric.Value, "unexpected metric value")
			require.Equal(t, "counter", metric.MType, "unexpected metric type")
		})
	}
}

func TestMetricConflicts(t *testing.T) {
	repo := &repository.MemStorage{}
	s := NewMetricService(repo)

	// Create a gauge metric
	err := s.UpdateGauge("Alloc", 23.5)
	assert.NoError(t, err, "failed to create gauge metric: %v", err)

	// Attempt to update the same metric as a counter
	err = s.UpdateCounter("Alloc", 1)
	assert.Error(t, err, "expected error when updating gauge as counter")

	// Create a counter metric
	err = s.UpdateCounter("PollCount", 1)
	assert.NoError(t, err, "failed to create counter metric: %v", err)

	// Attempt to update the same metric as a gauge
	err = s.UpdateGauge("PollCount", 23.5)
	assert.Error(t, err, "expected error when updating counter as gauge")
}
