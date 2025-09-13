package service

import (
	"context"

	models "go-metrics-and-alerts/internal/model"
	"go-metrics-and-alerts/internal/repository"
)

type MetricsService struct {
	repo repository.Repository
}

func NewMetricsService(repo repository.Repository) *MetricsService {
	return &MetricsService{
		repo: repo,
	}
}

func (s *MetricsService) UpdateGauge(ctx context.Context, name string, value float64) error {
	return s.repo.UpdateGauge(ctx, name, value)
}

func (s *MetricsService) UpdateCounter(ctx context.Context, name string, value int64) error {
	return s.repo.UpdateCounter(ctx, name, value)
}

func (s *MetricsService) GetGauge(ctx context.Context, name string) (float64, bool) {
	return s.repo.GetGauge(ctx, name)
}

func (s *MetricsService) GetCounter(ctx context.Context, name string) (int64, bool) {
	return s.repo.GetCounter(ctx, name)
}

func (s *MetricsService) GetAllGauges(ctx context.Context) map[string]float64 {
	return s.repo.GetAllGauges(ctx)
}

func (s *MetricsService) GetAllCounters(ctx context.Context) map[string]int64 {
	return s.repo.GetAllCounters(ctx)
}

func (s *MetricsService) UpdateBatch(ctx context.Context, metrics []models.Metrics) error {
	return s.repo.UpdateBatch(ctx, metrics)
}

func (s *MetricsService) UpdateMetricJSON(ctx context.Context, metric models.Metrics) (*models.Metrics, error) {
	switch metric.MType {
	case models.TypeGauge:
		if metric.Value != nil {
			err := s.repo.UpdateGauge(ctx, metric.ID, *metric.Value)
			if err != nil {
				return nil, err
			}
		}
	case models.TypeCounter:
		if metric.Delta != nil {
			err := s.repo.UpdateCounter(ctx, metric.ID, *metric.Delta)
			if err != nil {
				return nil, err
			}
		}
	}
	return &metric, nil
}

func (s *MetricsService) GetMetricJSON(ctx context.Context, metric models.Metrics) (*models.Metrics, error) {
	switch metric.MType {
	case models.TypeGauge:
		value, exists := s.repo.GetGauge(ctx, metric.ID)
		if !exists {
			return nil, nil
		}
		metric.Value = &value
	case models.TypeCounter:
		value, exists := s.repo.GetCounter(ctx, metric.ID)
		if !exists {
			return nil, nil
		}
		metric.Delta = &value
	}
	return &metric, nil
}