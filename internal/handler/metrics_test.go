package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-metrics-and-alerts/internal/repository"
	"go-metrics-and-alerts/internal/service"

	"github.com/go-chi/chi/v5"
)

func TestUpdateMetric(t *testing.T) {
	storage := repository.NewMemStorage()
	svc := service.NewMetricsService(storage)
	handler := New(svc)

	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", handler.UpdateMetric)

	tests := []struct {
		path   string
		status int
	}{
		{"/update/gauge/test/123.45", http.StatusOK},
		{"/update/counter/test/123", http.StatusOK},
		{"/update/bad/test/123", http.StatusBadRequest},
	}

	for _, test := range tests {
		req := httptest.NewRequest("POST", test.path, nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != test.status {
			t.Fatalf("Expected %d, got %d for %s", test.status, w.Code, test.path)
		}
	}
}

func TestGetMetric(t *testing.T) {
	storage := repository.NewMemStorage()
	svc := service.NewMetricsService(storage)
	handler := New(svc)
	
	ctx := context.Background()
	storage.UpdateGauge(ctx, "test", 123.45)
	storage.UpdateCounter(ctx, "counter", 100)

	r := chi.NewRouter()
	r.Get("/value/{type}/{name}", handler.GetMetric)

	tests := []struct {
		path   string
		status int
		body   string
	}{
		{"/value/gauge/test", http.StatusOK, "123.45"},
		{"/value/counter/counter", http.StatusOK, "100"},
		{"/value/gauge/missing", http.StatusNotFound, ""},
	}

	for _, test := range tests {
		req := httptest.NewRequest("GET", test.path, nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != test.status {
			t.Fatalf("Expected %d, got %d for %s", test.status, w.Code, test.path)
		}

		if test.body != "" && !strings.Contains(w.Body.String(), test.body) {
			t.Fatalf("Expected body to contain %s, got %s", test.body, w.Body.String())
		}
	}
}