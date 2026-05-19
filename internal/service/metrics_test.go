package service

import (
	"testing"

	models "github.com/LifeforDream/gometrics/internal/model"
	repository "github.com/LifeforDream/gometrics/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMetrics(t *testing.T) {
	type metric struct {
		mtype string
		name  string
		value float64
	}
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		repo  repository.MetricRepo
		input []metric
		want  []string
	}{
		{
			name: "get gauge and a counter",
			repo: &repository.MemStorage{},
			input: []metric{
				{"counter", "pollcount", float64(2)},
				{"gauge", "alloc", 1.25},
			},
			want: []string{
				"pollcount 2",
				"alloc 1.250000",
			},
		},
		{
			name: "get no metrics",
			repo: &repository.MemStorage{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewMetricService(tt.repo)
			for _, metric := range tt.input {
				tt.repo.SetMetric(models.Metrics{ID: metric.name, MType: metric.mtype, Value: &metric.value})
			}
			got := s.GetMetrics()
			assert.ElementsMatch(t, tt.want, got)
		})
	}
}

func TestGetMetric(t *testing.T) {
	floatPtr := func(f float64) *float64 { return &f }
	type input struct {
		name       string
		metricType string
	}
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		repo  repository.MetricRepo
		setUp []models.Metrics
		// Named input parameters for target function.
		input   input
		want    string
		wantErr bool
	}{
		{
			name: "get valid counter",
			repo: &repository.MemStorage{},
			setUp: []models.Metrics{
				{ID: "pollcount", MType: "counter", Value: floatPtr(2)},
			},
			input: input{
				name:       "pollcount",
				metricType: "counter",
			},
			want:    "2",
			wantErr: false,
		},
		{
			name: "get valid gauge",
			repo: &repository.MemStorage{},
			setUp: []models.Metrics{
				{ID: "alloc", MType: "gauge", Value: floatPtr(1.25)},
			},
			input: input{
				name:       "alloc",
				metricType: "gauge",
			},
			want:    "1.25",
			wantErr: false,
		},
		{
			name: "metric not found",
			repo: &repository.MemStorage{},
			input: input{
				name:       "whatever",
				metricType: "counter",
			},
			wantErr: true,
		},
		{
			name: "incorrect metric type",
			repo: &repository.MemStorage{},
			setUp: []models.Metrics{
				{ID: "pollcount", MType: "counter", Value: floatPtr(2)},
			},
			input: input{
				name:       "pollcount",
				metricType: "gauge",
			},
			wantErr: true,
		},
		{
			name: "invalid metric type",
			repo: &repository.MemStorage{},
			setUp: []models.Metrics{
				{ID: "pollcount", MType: "counter", Value: floatPtr(2)},
			},
			input: input{
				name:       "pollcount",
				metricType: "whatever",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewMetricService(tt.repo)
			for _, metric := range tt.setUp {
				tt.repo.SetMetric(metric)
			}

			got, gotErr := s.GetMetric(tt.input.metricType, tt.input.name)
			if tt.wantErr {
				require.Error(t, gotErr, "Unexpected success on err %s", gotErr)
			} else {
				require.NoError(t, gotErr, "Unexpected error %s", gotErr)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

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
