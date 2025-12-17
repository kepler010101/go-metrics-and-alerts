package grpcserver

import (
	"context"

	"go-metrics-and-alerts/internal/handler"
	models "go-metrics-and-alerts/internal/model"
	pb "go-metrics-and-alerts/internal/proto"
	"go-metrics-and-alerts/internal/repository"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	pb.UnimplementedMetricsServer
	Storage repository.Repository
}

func (s *Server) UpdateMetrics(ctx context.Context, req *pb.UpdateMetricsRequest) (*pb.UpdateMetricsResponse, error) {
	_ = ctx
	if s == nil || s.Storage == nil {
		return nil, status.Error(codes.Internal, "storage not configured")
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	list := req.GetMetrics()
	if len(list) == 0 {
		return &pb.UpdateMetricsResponse{}, nil
	}

	out := make([]models.Metrics, 0, len(list))
	for _, m := range list {
		if m == nil || m.Id == "" {
			continue
		}

		switch m.Type {
		case pb.Metric_GAUGE:
			v := m.Value
			out = append(out, models.Metrics{
				ID:    m.Id,
				MType: "gauge",
				Value: &v,
			})
		case pb.Metric_COUNTER:
			d := m.Delta
			out = append(out, models.Metrics{
				ID:    m.Id,
				MType: "counter",
				Delta: &d,
			})
		}
	}

	if len(out) == 0 {
		return &pb.UpdateMetricsResponse{}, nil
	}

	if err := s.Storage.UpdateBatch(out); err != nil {
		return nil, status.Error(codes.Internal, "storage error")
	}

	if handler.SyncSaveFunc != nil {
		handler.SyncSaveFunc()
	}

	return &pb.UpdateMetricsResponse{}, nil
}
