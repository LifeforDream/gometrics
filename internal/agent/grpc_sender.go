package agent

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	models "github.com/LifeforDream/gometrics/internal/model"
	pb "github.com/LifeforDream/gometrics/internal/proto"
)

const grpcServiceConfig = `{
	"methodConfig": [{
		"name": [{"service": "metrics.Metrics"}],
		"retryPolicy": {
		  "maxAttempts": 3,
		  "initialBackoff": "1s",
		  "maxBackoff": "5s",
		  "backoffMultiplier": 2.0,
		  "retryableStatusCodes": ["UNAVAILABLE"]
		}
	}]
}`

const realIPKey = "x-real-ip"

type grpcBatchSender struct {
	conn   *grpc.ClientConn
	client pb.MetricsClient
	hostIP string
}

func newGRPCBatchSender(addr, hostIP string, opts ...grpc.DialOption) (*grpcBatchSender, error) {
	base := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(grpcServiceConfig),
	}
	conn, err := grpc.NewClient(addr, append(base, opts...)...)
	if err != nil {
		return nil, fmt.Errorf("error creating grpc client: %w", err)
	}
	return &grpcBatchSender{conn: conn, client: pb.NewMetricsClient(conn), hostIP: hostIP}, nil
}

func (s *grpcBatchSender) Send(ctx context.Context, metrics []models.Metrics) error {
	pbmetrics, err := toProto(metrics)
	if err != nil {
		return fmt.Errorf("error converting metrics: %w", err)
	}
	ctx = metadata.AppendToOutgoingContext(ctx, realIPKey, s.hostIP)
	req := pb.UpdateMetricsRequest_builder{Metrics: pbmetrics}.Build()

	_, err = s.client.UpdateMetrics(ctx, req)
	if err != nil {
		return fmt.Errorf("error sending metrics via grpc: %w", err)
	}
	return nil
}

func (s *grpcBatchSender) Close() error {
	return s.conn.Close()
}

func toProto(metrics []models.Metrics) ([]*pb.Metric, error) {
	pbmetrics := make([]*pb.Metric, 0, len(metrics))
	for _, metric := range metrics {
		pbmetricbuilder := pb.Metric_builder{}
		switch metric.MType {
		case models.Counter:
			pbmetricbuilder.Type = pb.Metric_COUNTER
			pbmetricbuilder.Delta = *metric.Delta
		case models.Gauge:
			pbmetricbuilder.Type = pb.Metric_GAUGE
			pbmetricbuilder.Value = *metric.Value
		default:
			return nil, fmt.Errorf("invalid metric type: %s", metric.MType)
		}
		pbmetricbuilder.Id = metric.ID
		pbmetrics = append(pbmetrics, pbmetricbuilder.Build())
	}
	return pbmetrics, nil
}
