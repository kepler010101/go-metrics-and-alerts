package repository

import (
	"context"
	"sync"

	models "go-metrics-and-alerts/internal/model"
)

type MemStorage struct {
	gauges   map[string]float64
	counters map[string]int64
	mu       *sync.Mutex
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
		mu:       &sync.Mutex{},
	}
}

func (m *MemStorage) UpdateGauge(ctx context.Context, name string, value float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gauges[name] = value
	return nil
}

func (m *MemStorage) UpdateCounter(ctx context.Context, name string, value int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[name] += value
	return nil
}

func (m *MemStorage) GetGauge(ctx context.Context, name string) (float64, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	value, exists := m.gauges[name]
	return value, exists
}

func (m *MemStorage) GetCounter(ctx context.Context, name string) (int64, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	value, exists := m.counters[name]
	return value, exists
}

func (m *MemStorage) GetAllGauges(ctx context.Context) map[string]float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make(map[string]float64)
	for k, v := range m.gauges {
		result[k] = v
	}
	return result
}

func (m *MemStorage) GetAllCounters(ctx context.Context) map[string]int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make(map[string]int64)
	for k, v := range m.counters {
		result[k] = v
	}
	return result
}

func (m *MemStorage) UpdateBatch(ctx context.Context, metrics []models.Metrics) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, metric := range metrics {
		switch metric.MType {
		case models.TypeGauge:
			if metric.Value != nil {
				m.gauges[metric.ID] = *metric.Value
			}
		case models.TypeCounter:
			if metric.Delta != nil {
				m.counters[metric.ID] += *metric.Delta
			}
		}
	}
	return nil
}