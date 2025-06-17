package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go-metrics-and-alerts/internal/repository"
)

func TestUpdateMetric(t *testing.T) {
	storage := repository.NewMemStorage()
	handler := New(storage)

	tests := []struct {
		method string
		path   string
		status int
	}{
		{"POST", "/update/gauge/test/123.45", http.StatusOK},
		{"POST", "/update/counter/test/123", http.StatusOK},
		{"POST", "/update/bad/test/123", http.StatusBadRequest},
		{"POST", "/update/gauge//123", http.StatusNotFound},
		{"GET", "/update/gauge/test/123", http.StatusMethodNotAllowed},
	}

	for _, test := range tests {
		req := httptest.NewRequest(test.method, test.path, nil)
		w := httptest.NewRecorder()

		handler.UpdateMetric(w, req)

		if w.Code != test.status {
			t.Errorf("Expected %d, got %d for %s %s", test.status, w.Code, test.method, test.path)
		}
	}
}
