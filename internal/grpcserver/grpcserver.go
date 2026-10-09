// Package grpcserver реализует приёмку метрик по транспорту GRPC.
package grpcserver

import (
	"context"
	"errors"
	"net"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	models "github.com/LifeforDream/gometrics/internal/model"
	myErrors "github.com/LifeforDream/gometrics/internal/model/errors"
	pb "github.com/LifeforDream/gometrics/internal/proto"
)

// MetricUpdater реализует то, что нужно серверу для сохранения метрик.
type MetricUpdater interface {
	UpdateMetrics(ctx context.Context, metrics []models.Metrics) error
	ValidateMetric(metric models.Metrics) error
}

// MetricsServer реализует приёмку метрик по grpc и сохранение через svc.
type MetricsServer struct {
	pb.UnimplementedMetricsServer
	svc    MetricUpdater
	logger *zap.Logger
}

func fromProto(pbmetric *pb.Metric) models.Metrics {
	metric := models.Metrics{ID: pbmetric.GetId()}
	switch pbmetric.GetType() {
	case pb.Metric_COUNTER:
		d := pbmetric.GetDelta()
		metric.Delta = &d
		metric.MType = models.Counter
	case pb.Metric_GAUGE:
		v := pbmetric.GetValue()
		metric.Value = &v
		metric.MType = models.Gauge
	}
	return metric
}

// UpdateMetrics получает метрики по gRPC, конвертирует их в модели и передаёт в svc.
func (ms *MetricsServer) UpdateMetrics(ctx context.Context, req *pb.UpdateMetricsRequest) (*pb.UpdateMetricsResponse, error) {
	metrics := make([]models.Metrics, 0, len(req.GetMetrics()))
	for _, pbmetric := range req.GetMetrics() {
		metric := fromProto(pbmetric)
		if err := ms.svc.ValidateMetric(metric); err != nil {
			ms.logger.Debug("invalid metric", zap.String("id", metric.ID), zap.Error(err))
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		metrics = append(metrics, metric)
	}
	err := ms.svc.UpdateMetrics(ctx, metrics)
	if err != nil {
		if invalidTypeErr, ok := errors.AsType[myErrors.InvalidMetricType](err); ok {
			ms.logger.Debug("invalid metric type", zap.String("newType", invalidTypeErr.NewType))
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		ms.logger.Error("unexpected error", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal server error while updating metrics")
	}
	return &pb.UpdateMetricsResponse{}, nil
}

// New создаёт grpc-сервер для обработки метрик.
// Запуск и остановка должны производиться извне.
// Если creds не переданы, сервер использует plaintext.
func New(svc MetricUpdater, network *net.IPNet, logger *zap.Logger, creds credentials.TransportCredentials) *grpc.Server {
	if creds == nil {
		creds = insecure.NewCredentials()
	}
	s := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			ClientIPInterceptor(),
			TrustedSubnetInterceptor(network),
		),
		grpc.Creds(creds),
	)
	pb.RegisterMetricsServer(s, &MetricsServer{svc: svc, logger: logger})
	return s
}
