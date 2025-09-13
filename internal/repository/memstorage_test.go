package repository

import (
	"context"
	"testing"
)

func TestMemStorage(t *testing.T) {
	storage := NewMemStorage()
	ctx := context.Background()

	storage.UpdateGauge(ctx, "test", 123.45)
	value, exists := storage.GetGauge(ctx, "test")
	if !exists || value != 123.45 {
		t.Errorf("Expected 123.45, got %f", value)
	}

	storage.UpdateCounter(ctx, "counter", 10)
	storage.UpdateCounter(ctx, "counter", 5)
	counter, exists := storage.GetCounter(ctx, "counter")
	if !exists || counter != 15 {
		t.Errorf("Expected 15, got %d", counter)
	}
}