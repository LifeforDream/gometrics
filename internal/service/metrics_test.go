package service

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LifeforDream/gometrics/internal/audit"
	models "github.com/LifeforDream/gometrics/internal/model"
	repository "github.com/LifeforDream/gometrics/internal/repository"
	"github.com/LifeforDream/gometrics/internal/utils"
)

// fakeAuditor is a test double for the service.Auditor interface: it
// records every call so tests can assert whether (and with what) the
// observer pattern's notification was fired.
type fakeAuditor struct {
	mu    sync.Mutex
	calls []auditCall
}

type auditCall struct {
	metrics []models.Metrics
	ip      string
}

func (f *fakeAuditor) Update(metrics []models.Metrics, ip string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, auditCall{metrics: metrics, ip: ip})
}

func (f *fakeAuditor) Calls() []auditCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]auditCall(nil), f.calls...)
}

func TestGetMetrics(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		repo  MetricRepo
		input []models.Metrics
		want  []string
	}{
		{
			name: "get gauge and a counter",
			repo: repository.NewMemStorage(),
			input: []models.Metrics{
				{MType: "counter", ID: "pollcount", Delta: utils.IntPtr(2)},
				{MType: "gauge", ID: "alloc", Value: utils.FloatPtr(1.25)},
			},
			want: []string{
				"pollcount 2",
				"alloc 1.250000",
			},
		},
		{
			name: "get no metrics",
			repo: repository.NewMemStorage(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewMetricService(tt.repo, &audit.Auditor{})
			for _, metric := range tt.input {
				if metric.MType == "counter" {
					tt.repo.UpdateCounter(t.Context(), metric)
				} else {
					tt.repo.SetGauge(t.Context(), metric)
				}
			}
			got, err := s.GetMetrics(t.Context())
			require.NoError(t, err)
			assert.ElementsMatch(t, tt.want, got)
		})
	}
}

func TestGetMetricValue(t *testing.T) {
	type input struct {
		name       string
		metricType string
	}
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		repo  MetricRepo
		setUp []models.Metrics
		// Named input parameters for target function.
		input   input
		want    string
		wantErr bool
	}{
		{
			name: "get valid counter",
			repo: repository.NewMemStorage(),
			setUp: []models.Metrics{
				{ID: "pollcount", MType: "counter", Delta: utils.IntPtr(2)},
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
			repo: repository.NewMemStorage(),
			setUp: []models.Metrics{
				{ID: "alloc", MType: "gauge", Value: utils.FloatPtr(1.25)},
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
			repo: repository.NewMemStorage(),
			input: input{
				name:       "whatever",
				metricType: "counter",
			},
			wantErr: true,
		},
		{
			name: "incorrect metric type",
			repo: repository.NewMemStorage(),
			setUp: []models.Metrics{
				{ID: "pollcount", MType: "counter", Delta: utils.IntPtr(2)},
			},
			input: input{
				name:       "pollcount",
				metricType: "gauge",
			},
			wantErr: true,
		},
		{
			name: "invalid metric type",
			repo: repository.NewMemStorage(),
			setUp: []models.Metrics{
				{ID: "pollcount", MType: "counter", Delta: utils.IntPtr(2)},
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
			s := NewMetricService(tt.repo, &audit.Auditor{})
			for _, metric := range tt.setUp {
				if metric.MType == "counter" {
					tt.repo.UpdateCounter(t.Context(), metric)
				} else {
					tt.repo.SetGauge(t.Context(), metric)
				}
			}

			got, gotErr := s.GetMetricValue(t.Context(), tt.input.metricType, tt.input.name)
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
			repo := repository.NewMemStorage()
			s := NewMetricService(repo, &audit.Auditor{})
			for _, val := range tt.values {
				s.UpdateGaugeByName(t.Context(), tt.mname, val)
			}
			metric, err := repo.GetMetric(t.Context(), tt.mname)
			require.NoError(t, err, "metric not found in repository")
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
			repo := repository.NewMemStorage()
			s := NewMetricService(repo, &audit.Auditor{})
			for _, val := range tt.values {
				s.UpdateCounterByName(t.Context(), tt.mname, val)
			}
			metric, err := repo.GetMetric(t.Context(), tt.mname)
			require.NoError(t, err, "metric not found in repository")
			require.Equal(t, tt.wantVal, *metric.Delta, "unexpected metric value")
			require.Equal(t, "counter", metric.MType, "unexpected metric type")
		})
	}
}

func TestUpdateMetrics(t *testing.T) {
	tests := []struct {
		name    string
		setUp   []models.Metrics
		input   []models.Metrics
		wantErr bool
		verify  func(t *testing.T, s *MetricService)
	}{
		{
			name: "mixed batch applied",
			input: []models.Metrics{
				{ID: "alloc", MType: models.Gauge, Value: utils.FloatPtr(1.25)},
				{ID: "pollcount", MType: models.Counter, Delta: utils.IntPtr(3)},
			},
			verify: func(t *testing.T, s *MetricService) {
				g, err := s.GetMetric(t.Context(), models.Gauge, "alloc")
				require.NoError(t, err)
				assert.Equal(t, 1.25, *g.Value)

				c, err := s.GetMetric(t.Context(), models.Counter, "pollcount")
				require.NoError(t, err)
				assert.Equal(t, int64(3), *c.Delta)
			},
		},
		{
			name:  "counter accumulates over existing value",
			setUp: []models.Metrics{{ID: "pollcount", MType: models.Counter, Delta: utils.IntPtr(2)}},
			input: []models.Metrics{
				{ID: "pollcount", MType: models.Counter, Delta: utils.IntPtr(3)},
			},
			verify: func(t *testing.T, s *MetricService) {
				c, err := s.GetMetric(t.Context(), models.Counter, "pollcount")
				require.NoError(t, err)
				assert.Equal(t, int64(5), *c.Delta)
			},
		},
		{
			name:    "error propagated from repository",
			input:   []models.Metrics{{ID: "bad", MType: "invalid", Value: utils.FloatPtr(1.0)}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := repository.NewMemStorage()
			s := NewMetricService(repo, &audit.Auditor{})
			for _, m := range tt.setUp {
				if m.MType == models.Counter {
					repo.UpdateCounter(t.Context(), m)
				} else {
					repo.SetGauge(t.Context(), m)
				}
			}

			err := s.UpdateMetrics(t.Context(), tt.input)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			if tt.verify != nil {
				tt.verify(t, s)
			}
		})
	}
}

func TestMetricConflicts(t *testing.T) {
	repo := repository.NewMemStorage()
	s := NewMetricService(repo, &audit.Auditor{})

	// Create a gauge metric
	err := s.UpdateGaugeByName(t.Context(), "Alloc", 23.5)
	assert.NoError(t, err, "failed to create gauge metric: %v", err)

	// Attempt to update the same metric as a counter
	err = s.UpdateCounterByName(t.Context(), "Alloc", 1)
	assert.Error(t, err, "expected error when updating gauge as counter")

	// Create a counter metric
	err = s.UpdateCounterByName(t.Context(), "PollCount", 1)
	assert.NoError(t, err, "failed to create counter metric: %v", err)

	// Attempt to update the same metric as a gauge
	err = s.UpdateGaugeByName(t.Context(), "PollCount", 23.5)
	assert.Error(t, err, "expected error when updating counter as gauge")
}

func TestGetMetric(t *testing.T) {
	type input struct {
		name       string
		metricType string
	}
	tests := []struct {
		name    string
		repo    MetricRepo
		setUp   []models.Metrics
		input   input
		want    models.Metrics
		wantErr bool
	}{
		{
			name: "get valid counter",
			repo: repository.NewMemStorage(),
			setUp: []models.Metrics{
				{ID: "pollcount", MType: "counter", Delta: utils.IntPtr(2)},
			},
			input:   input{"pollcount", "counter"},
			want:    models.Metrics{ID: "pollcount", MType: "counter", Delta: utils.IntPtr(2)},
			wantErr: false,
		},
		{
			name: "get valid gauge",
			repo: repository.NewMemStorage(),
			setUp: []models.Metrics{
				{ID: "alloc", MType: "gauge", Value: utils.FloatPtr(1.25)},
			},
			input:   input{"alloc", "gauge"},
			want:    models.Metrics{ID: "alloc", MType: "gauge", Value: utils.FloatPtr(1.25)},
			wantErr: false,
		},
		{
			name:    "metric not found",
			repo:    repository.NewMemStorage(),
			input:   input{"whatever", "counter"},
			wantErr: true,
		},
		{
			name: "incorrect metric type",
			repo: repository.NewMemStorage(),
			setUp: []models.Metrics{
				{ID: "pollcount", MType: "counter", Delta: utils.IntPtr(2)},
			},
			input:   input{"pollcount", "gauge"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewMetricService(tt.repo, &audit.Auditor{})
			for _, metric := range tt.setUp {
				if metric.MType == "counter" {
					tt.repo.UpdateCounter(t.Context(), metric)
				} else {
					tt.repo.SetGauge(t.Context(), metric)
				}
			}
			got, gotErr := s.GetMetric(t.Context(), tt.input.metricType, tt.input.name)
			if tt.wantErr {
				require.Error(t, gotErr)
			} else {
				require.NoError(t, gotErr)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestAuditNotifiedOnSuccessfulUpdates(t *testing.T) {
	ctx := utils.WithClientIP(t.Context(), "192.168.0.42")

	tests := []struct {
		name    string
		call    func(s *MetricService) error
		wantIDs []string
	}{
		{
			name: "UpdateGaugeByName",
			call: func(s *MetricService) error {
				return s.UpdateGaugeByName(ctx, "Alloc", 1.25)
			},
			wantIDs: []string{"Alloc"},
		},
		{
			name: "UpdateCounterByName",
			call: func(s *MetricService) error {
				return s.UpdateCounterByName(ctx, "PollCount", 1)
			},
			wantIDs: []string{"PollCount"},
		},
		{
			name: "UpdateGauge",
			call: func(s *MetricService) error {
				return s.UpdateGauge(ctx, models.Metrics{ID: "Alloc", MType: models.Gauge, Value: utils.FloatPtr(1.25)})
			},
			wantIDs: []string{"Alloc"},
		},
		{
			name: "UpdateCounter",
			call: func(s *MetricService) error {
				return s.UpdateCounter(ctx, models.Metrics{ID: "PollCount", MType: models.Counter, Delta: utils.IntPtr(1)})
			},
			wantIDs: []string{"PollCount"},
		},
		{
			name: "UpdateMetrics batch reports all metrics in one call",
			call: func(s *MetricService) error {
				return s.UpdateMetrics(ctx, []models.Metrics{
					{ID: "Alloc", MType: models.Gauge, Value: utils.FloatPtr(1.25)},
					{ID: "PollCount", MType: models.Counter, Delta: utils.IntPtr(1)},
				})
			},
			wantIDs: []string{"Alloc", "PollCount"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fa := &fakeAuditor{}
			s := NewMetricService(repository.NewMemStorage(), fa)

			err := tt.call(s)
			require.NoError(t, err)

			calls := fa.Calls()
			require.Len(t, calls, 1, "expected exactly one audit notification")
			assert.Equal(t, "192.168.0.42", calls[0].ip)
			gotIDs := make([]string, 0, len(calls[0].metrics))
			for _, m := range calls[0].metrics {
				gotIDs = append(gotIDs, m.ID)
			}
			assert.Equal(t, tt.wantIDs, gotIDs)
		})
	}
}

func TestAuditNotNotifiedOnFailedUpdates(t *testing.T) {
	ctx := t.Context()

	tests := []struct {
		name  string
		setUp func(s *MetricService)
		call  func(s *MetricService) error
	}{
		{
			name: "UpdateCounterByName on existing gauge",
			setUp: func(s *MetricService) {
				require.NoError(t, s.UpdateGaugeByName(ctx, "Alloc", 1.25))
			},
			call: func(s *MetricService) error {
				return s.UpdateCounterByName(ctx, "Alloc", 1)
			},
		},
		{
			name: "UpdateGauge with invalid type conflict",
			setUp: func(s *MetricService) {
				require.NoError(t, s.UpdateCounterByName(ctx, "PollCount", 1))
			},
			call: func(s *MetricService) error {
				return s.UpdateGauge(ctx, models.Metrics{ID: "PollCount", MType: models.Gauge, Value: utils.FloatPtr(1.0)})
			},
		},
		{
			name: "UpdateMetrics batch with invalid metric type",
			call: func(s *MetricService) error {
				return s.UpdateMetrics(ctx, []models.Metrics{{ID: "bad", MType: "invalid", Value: utils.FloatPtr(1.0)}})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fa := &fakeAuditor{}
			s := NewMetricService(repository.NewMemStorage(), fa)
			if tt.setUp != nil {
				tt.setUp(s)
			}
			// Any audit calls fired during setUp shouldn't count toward the assertion below.
			fa.mu.Lock()
			fa.calls = nil
			fa.mu.Unlock()

			err := tt.call(s)
			require.Error(t, err)
			assert.Empty(t, fa.Calls(), "auditor should not be notified when the update failed")
		})
	}
}

func TestAuditReceivesClientIPFromMiddleware(t *testing.T) {
	fa := &fakeAuditor{}
	s := NewMetricService(repository.NewMemStorage(), fa)

	ctx := utils.WithClientIP(context.Background(), "127.0.0.1")
	s.UpdateGaugeByName(ctx, "Alloc", 1.25)

	calls := fa.Calls()
	require.Len(t, calls, 1)
	assert.Equal(t, "127.0.0.1", calls[0].ip)
}
