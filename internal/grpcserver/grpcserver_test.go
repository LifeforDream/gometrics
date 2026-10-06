package grpcserver

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	models "github.com/LifeforDream/gometrics/internal/model"
	myErrors "github.com/LifeforDream/gometrics/internal/model/errors"
	pb "github.com/LifeforDream/gometrics/internal/proto"
)

// fakeMetricUpdater — заглушка MetricUpdater, запоминающая метрики,
// с которыми был вызван UpdateMetrics, и возвращающая заданную ошибку.
type fakeMetricUpdater struct {
	gotMetrics  []models.Metrics
	err         error
	validateErr error
}

func (f *fakeMetricUpdater) UpdateMetrics(_ context.Context, metrics []models.Metrics) error {
	f.gotMetrics = metrics
	return f.err
}

func (f *fakeMetricUpdater) ValidateMetric(models.Metrics) error { return f.validateErr }

func TestFromProto(t *testing.T) {
	tests := []struct {
		name     string
		pbmetric *pb.Metric
		want     models.Metrics
		wantErr  bool
	}{
		{
			name:     "valid counter",
			pbmetric: pb.Metric_builder{Id: "cnt", Type: pb.Metric_COUNTER, Delta: 5}.Build(),
			want:     models.Metrics{ID: "cnt", MType: models.Counter, Delta: new(int64(5))},
		},
		{
			name:     "valid gauge",
			pbmetric: pb.Metric_builder{Id: "g", Type: pb.Metric_GAUGE, Value: 1.25}.Build(),
			want:     models.Metrics{ID: "g", MType: models.Gauge, Value: new(1.25)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fromProto(tt.pbmetric)

			assert.Equal(t, tt.want.ID, got.ID)
			assert.Equal(t, tt.want.MType, got.MType)
			assert.Equal(t, tt.want.Delta, got.Delta)
			assert.Equal(t, tt.want.Value, got.Value)
		})
	}
}

func TestMetricsServerUpdateMetrics(t *testing.T) {
	req := pb.UpdateMetricsRequest_builder{
		Metrics: []*pb.Metric{
			pb.Metric_builder{Id: "cnt", Type: pb.Metric_COUNTER, Delta: 5}.Build(),
			pb.Metric_builder{Id: "g", Type: pb.Metric_GAUGE, Value: 1.25}.Build(),
		},
	}.Build()
	wantMetrics := []models.Metrics{
		{ID: "cnt", MType: models.Counter, Delta: new(int64(5))},
		{ID: "g", MType: models.Gauge, Value: new(1.25)},
	}

	t.Run("passes mapped metrics to svc and returns empty response", func(t *testing.T) {
		svc := &fakeMetricUpdater{}
		ms := &MetricsServer{svc: svc, logger: zap.NewNop()}

		resp, err := ms.UpdateMetrics(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		require.Len(t, svc.gotMetrics, len(wantMetrics))
		for i := range wantMetrics {
			assert.Equal(t, wantMetrics[i].ID, svc.gotMetrics[i].ID)
			assert.Equal(t, wantMetrics[i].MType, svc.gotMetrics[i].MType)
			assert.Equal(t, wantMetrics[i].Delta, svc.gotMetrics[i].Delta)
			assert.Equal(t, wantMetrics[i].Value, svc.gotMetrics[i].Value)
		}
	})

	t.Run("rejects metric failing ValidateMetric (e.g. empty ID) without calling UpdateMetrics", func(t *testing.T) {
		svc := &fakeMetricUpdater{validateErr: myErrors.ErrEmptyMetricID}
		ms := &MetricsServer{svc: svc, logger: zap.NewNop()}

		resp, err := ms.UpdateMetrics(context.Background(), req)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Nil(t, svc.gotMetrics, "UpdateMetrics must not be called once ValidateMetric rejects a metric")
	})

	t.Run("maps InvalidMetricType to codes.InvalidArgument", func(t *testing.T) {
		svc := &fakeMetricUpdater{err: myErrors.InvalidMetricType{
			ExistingType: models.Gauge,
			NewType:      models.Counter,
			MetricName:   "m1",
		}}
		ms := &MetricsServer{svc: svc, logger: zap.NewNop()}

		resp, err := ms.UpdateMetrics(context.Background(), req)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("maps unexpected errors to codes.Internal", func(t *testing.T) {
		svc := &fakeMetricUpdater{err: errors.New("boom")}
		ms := &MetricsServer{svc: svc, logger: zap.NewNop()}

		resp, err := ms.UpdateMetrics(context.Background(), req)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.Internal, status.Code(err))
	})

	t.Run("empty metrics list is passed through as-is", func(t *testing.T) {
		svc := &fakeMetricUpdater{}
		ms := &MetricsServer{svc: svc, logger: zap.NewNop()}
		emptyReq := pb.UpdateMetricsRequest_builder{}.Build()

		resp, err := ms.UpdateMetrics(context.Background(), emptyReq)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Empty(t, svc.gotMetrics)
	})
}
