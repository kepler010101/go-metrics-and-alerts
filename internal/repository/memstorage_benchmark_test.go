package repository

import (
	"testing"

	models "go-metrics-and-alerts/internal/model"
)

func BenchmarkMemStorageUpdateBatch(b *testing.B) {
	storage := NewMemStorage()
	val := 123.45
	delta := int64(5)
	items := []models.Metrics{
		{ID: "Alloc", MType: models.Gauge, Value: &val},
		{ID: "PollCount", MType: models.Counter, Delta: &delta},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := storage.UpdateBatch(items); err != nil {
			b.Fatalf("update batch: %v", err)
		}
	}
}
