package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	models "github.com/LifeforDream/gometrics/internal/model"
	pb "github.com/LifeforDream/gometrics/internal/proto"
)

func TestToProto(t *testing.T) {
	tests := []struct {
		name    string
		metrics []models.Metrics
		want    []*pb.Metric
		wantErr bool
	}{
		{
			name:    "counter",
			metrics: []models.Metrics{{ID: "cnt", MType: models.Counter, Delta: new(int64(5))}},
			want:    []*pb.Metric{pb.Metric_builder{Id: "cnt", Type: pb.Metric_COUNTER, Delta: 5}.Build()},
		},
		{
			name:    "gauge",
			metrics: []models.Metrics{{ID: "g", MType: models.Gauge, Value: new(1.25)}},
			want:    []*pb.Metric{pb.Metric_builder{Id: "g", Type: pb.Metric_GAUGE, Value: 1.25}.Build()},
		},
		{
			name: "multiple metrics preserve order",
			metrics: []models.Metrics{
				{ID: "a", MType: models.Counter, Delta: new(int64(1))},
				{ID: "b", MType: models.Gauge, Value: new(2.5)},
			},
			want: []*pb.Metric{
				pb.Metric_builder{Id: "a", Type: pb.Metric_COUNTER, Delta: 1}.Build(),
				pb.Metric_builder{Id: "b", Type: pb.Metric_GAUGE, Value: 2.5}.Build(),
			},
		},
		{
			name:    "unsupported type returns error",
			metrics: []models.Metrics{{MType: "unknown"}},
			wantErr: true,
		},
		{
			name:    "empty input gives empty output",
			metrics: []models.Metrics{},
			want:    []*pb.Metric{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toProto(tt.metrics)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.Len(t, got, len(tt.want))
			for i := range tt.want {
				assert.Equal(t, tt.want[i].GetId(), got[i].GetId())
				assert.Equal(t, tt.want[i].GetType(), got[i].GetType())
				assert.Equal(t, tt.want[i].GetDelta(), got[i].GetDelta())
				assert.Equal(t, tt.want[i].GetValue(), got[i].GetValue())
			}
		})
	}
}
