package handler

import (
	"net/http"
	"strconv"
	"strings"

	"go-metrics-and-alerts/internal/repository"
)

type Handler struct {
	storage repository.Repository
}

func New(storage repository.Repository) *Handler {
	return &Handler{storage: storage}
}

func (h *Handler) UpdateMetric(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/update/")
	parts := strings.Split(path, "/")

	if len(parts) != 3 {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	metricType := parts[0]
	metricName := parts[1]
	metricValue := parts[2]

	if metricName == "" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	switch metricType {
	case "gauge":
		value, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		h.storage.UpdateGauge(metricName, value)

	case "counter":
		value, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		h.storage.UpdateCounter(metricName, value)

	default:
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}
