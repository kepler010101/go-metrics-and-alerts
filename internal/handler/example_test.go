package handler_test

import (
	"bytes"
	"fmt"
	"net/http/httptest"

	"go-metrics-and-alerts/internal/handler"
	"go-metrics-and-alerts/internal/repository"
)

func ExampleHandler_UpdateMetricJSON() {
	storage := repository.NewMemStorage()
	h := handler.New(storage)

	body := []byte(`{"id":"Alloc","type":"gauge","value":123.45}`)
	req := httptest.NewRequest("POST", "/update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.UpdateMetricJSON(rr, req)

	fmt.Println(rr.Code)
	if value, ok := storage.GetGauge("Alloc"); ok {
		fmt.Printf("%.2f\n", value)
	}

	// Output:
	// 200
	// 123.45
}

func ExampleHandler_UpdateMetricsBatch() {
	storage := repository.NewMemStorage()
	h := handler.New(storage)

	body := []byte(`[{"id":"Requests","type":"counter","delta":3},{"id":"Load","type":"gauge","value":0.5}]`)
	req := httptest.NewRequest("POST", "/updates/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.UpdateMetricsBatch(rr, req)

	fmt.Println(rr.Code)
	if delta, ok := storage.GetCounter("Requests"); ok {
		fmt.Println(delta)
	}

	// Output:
	// 200
	// 3
}
