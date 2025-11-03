package handler

import (
	"net/http/httptest"
	"strings"
	"testing"

	"go-metrics-and-alerts/internal/repository"
)

func BenchmarkUpdateMetricJSON(b *testing.B) {
	storage := repository.NewMemStorage()
	h := New(storage)

	body := `{"id":"Alloc","type":"gauge","value":123.45}`
	req := httptest.NewRequest("POST", "/update", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		h.UpdateMetricJSON(rec, req.Clone(req.Context()))
	}
}

func BenchmarkUpdateMetricsBatch(b *testing.B) {
	storage := repository.NewMemStorage()
	h := New(storage)

	body := `[{"id":"Alloc","type":"gauge","value":123.45},{"id":"PollCount","type":"counter","delta":5}]`
	req := httptest.NewRequest("POST", "/updates/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		h.UpdateMetricsBatch(rec, req.Clone(req.Context()))
	}
}

func BenchmarkListMetrics(b *testing.B) {
	storage := repository.NewMemStorage()
	storage.UpdateGauge("Alloc", 123.45)
	storage.UpdateCounter("PollCount", 5)
	h := New(storage)

	req := httptest.NewRequest("GET", "/", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		h.ListMetrics(rec, req.Clone(req.Context()))
	}
}
