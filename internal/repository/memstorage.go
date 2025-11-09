package repository

import (
	"sync"

	models "go-metrics-and-alerts/internal/model"
)

// generate:reset
// MemStorage keeps metrics in memory using simple maps.
type MemStorage struct {
	gauges   map[string]float64
	counters map[string]int64
	mu       *sync.Mutex
}

// NewMemStorage creates an empty in-memory storage.
func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
		mu:       &sync.Mutex{},
	}
}

// UpdateGauge sets the gauge value.
func (m *MemStorage) UpdateGauge(name string, value float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gauges[name] = value
	return nil
}

// UpdateCounter adds the delta to the counter value.
func (m *MemStorage) UpdateCounter(name string, value int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[name] += value
	return nil
}

// GetGauge returns the gauge value and flag indicating presence.
func (m *MemStorage) GetGauge(name string) (float64, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	value, exists := m.gauges[name]
	return value, exists
}

// GetCounter returns the counter value and flag indicating presence.
func (m *MemStorage) GetCounter(name string) (int64, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	value, exists := m.counters[name]
	return value, exists
}

// GetAllGauges returns a copy of all gauge values.
func (m *MemStorage) GetAllGauges() map[string]float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make(map[string]float64)
	for k, v := range m.gauges {
		result[k] = v
	}
	return result
}

// GetAllCounters returns a copy of all counter values.
func (m *MemStorage) GetAllCounters() map[string]int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make(map[string]int64)
	for k, v := range m.counters {
		result[k] = v
	}
	return result
}

// UpdateBatch applies all metrics updates in order.
func (m *MemStorage) UpdateBatch(metrics []models.Metrics) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, metric := range metrics {
		switch metric.MType {
		case "gauge":
			if metric.Value != nil {
				m.gauges[metric.ID] = *metric.Value
			}
		case "counter":
			if metric.Delta != nil {
				m.counters[metric.ID] += *metric.Delta
			}
		}
	}
	return nil
}
