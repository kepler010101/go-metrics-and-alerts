package repository

import "testing"

func TestMemStorage(t *testing.T) {
	storage := NewMemStorage()

	storage.UpdateGauge("test", 123.45)
	value, exists := storage.GetGauge("test")
	if !exists || value != 123.45 {
		t.Errorf("Expected 123.45, got %f", value)
	}

	storage.UpdateCounter("counter", 10)
	storage.UpdateCounter("counter", 5)
	counter, exists := storage.GetCounter("counter")
	if !exists || counter != 15 {
		t.Errorf("Expected 15, got %d", counter)
	}
}
