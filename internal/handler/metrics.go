package handler

import (
	"context"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strconv"

	models "go-metrics-and-alerts/internal/model"
	"go-metrics-and-alerts/internal/service"

	"github.com/go-chi/chi/v5"
)

var SyncSaveFunc func()

type Handler struct {
	service *service.MetricsService
}

func New(svc *service.MetricsService) *Handler {
	return &Handler{service: svc}
}

func (h *Handler) UpdateMetric(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")
	metricValue := chi.URLParam(r, "value")

	if metricName == "" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	ctx := r.Context()

	switch metricType {
	case models.TypeGauge:
		value, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		if err := h.service.UpdateGauge(ctx, metricName, value); err != nil {
			log.Printf("Error updating gauge: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

	case models.TypeCounter:
		value, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		if err := h.service.UpdateCounter(ctx, metricName, value); err != nil {
			log.Printf("Error updating counter: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

	default:
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if SyncSaveFunc != nil {
		SyncSaveFunc()
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetMetric(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")
	ctx := r.Context()

	switch metricType {
	case models.TypeGauge:
		value, exists := h.service.GetGauge(ctx, metricName)
		if !exists {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		if _, err := w.Write([]byte(strconv.FormatFloat(value, 'g', -1, 64))); err != nil {
			log.Printf("Error writing response: %v", err)
		}

	case models.TypeCounter:
		value, exists := h.service.GetCounter(ctx, metricName)
		if !exists {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		if _, err := w.Write([]byte(strconv.FormatInt(value, 10))); err != nil {
			log.Printf("Error writing response: %v", err)
		}

	default:
		http.Error(w, "Bad request", http.StatusBadRequest)
	}
}

func (h *Handler) ListMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "text/html")

	tmpl := `<html><body><h1>Metrics</h1>
<h2>Gauges</h2><ul>
{{range $name, $value := .Gauges}}
<li>{{$name}}: {{$value}}</li>
{{end}}
</ul>
<h2>Counters</h2><ul>
{{range $name, $value := .Counters}}
<li>{{$name}}: {{$value}}</li>
{{end}}
</ul>
</body></html>`

	t, err := template.New("metrics").Parse(tmpl)
	if err != nil {
		log.Printf("Error parsing template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := struct {
		Gauges   map[string]float64
		Counters map[string]int64
	}{
		Gauges:   h.service.GetAllGauges(ctx),
		Counters: h.service.GetAllCounters(ctx),
	}

	if err := t.Execute(w, data); err != nil {
		log.Printf("Error executing template: %v", err)
	}
}

func (h *Handler) UpdateMetricJSON(w http.ResponseWriter, r *http.Request) {
	var metric models.Metrics
	ctx := r.Context()

	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	result, err := h.service.UpdateMetricJSON(ctx, metric)
	if err != nil {
		log.Printf("Error updating metric: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if SyncSaveFunc != nil {
		SyncSaveFunc()
	}

	w.Header().Set("Content-Type", "application/json")
	resp, err := json.Marshal(result)
	if err != nil {
		log.Printf("Error marshaling response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func (h *Handler) GetMetricJSON(w http.ResponseWriter, r *http.Request) {
	var metric models.Metrics
	ctx := r.Context()

	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	result, err := h.service.GetMetricJSON(ctx, metric)
	if err != nil {
		log.Printf("Error getting metric: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	
	if result == nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	resp, err := json.Marshal(result)
	if err != nil {
		log.Printf("Error marshaling response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func (h *Handler) UpdateMetricsBatch(w http.ResponseWriter, r *http.Request) {
	var metrics []models.Metrics
	ctx := r.Context()

	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if len(metrics) == 0 {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if err := h.service.UpdateBatch(ctx, metrics); err != nil {
		log.Printf("Error updating metrics batch: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if SyncSaveFunc != nil {
		SyncSaveFunc()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}