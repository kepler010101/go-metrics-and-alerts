package handler

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strconv"

	models "go-metrics-and-alerts/internal/model"
	"go-metrics-and-alerts/internal/repository"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	storage repository.Repository
}

func New(storage repository.Repository) *Handler {
	return &Handler{storage: storage}
}

func (h *Handler) UpdateMetric(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")
	metricValue := chi.URLParam(r, "value")

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
		if err := h.storage.UpdateGauge(metricName, value); err != nil {
			log.Printf("Error updating gauge: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

	case "counter":
		value, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		if err := h.storage.UpdateCounter(metricName, value); err != nil {
			log.Printf("Error updating counter: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

	default:
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetMetric(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")

	switch metricType {
	case "gauge":
		value, exists := h.storage.GetGauge(metricName)
		if !exists {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		if _, err := w.Write([]byte(strconv.FormatFloat(value, 'g', -1, 64))); err != nil {
			log.Printf("Error writing response: %v", err)
		}

	case "counter":
		value, exists := h.storage.GetCounter(metricName)
		if !exists {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		if _, err := w.Write([]byte(strconv.FormatInt(value, 10))); err != nil {
			log.Printf("Error writingg response: %v", err)
		}

	default:
		http.Error(w, "Bad request", http.StatusBadRequest)
	}
}

func (h *Handler) ListMetrics(w http.ResponseWriter, r *http.Request) {
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
		Gauges:   h.storage.GetAllGauges(),
		Counters: h.storage.GetAllCounters(),
	}

	if err := t.Execute(w, data); err != nil {
		log.Printf("Error executing template: %v", err)
	}
}

func (h *Handler) UpdateMetricJSON(w http.ResponseWriter, r *http.Request) {
	var metric models.Metrics

	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	switch metric.MType {
	case "gauge":
		if metric.Value == nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		if err := h.storage.UpdateGauge(metric.ID, *metric.Value); err != nil {
			log.Printf("Error updating gauge: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

	case "counter":
		if metric.Delta == nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		if err := h.storage.UpdateCounter(metric.ID, *metric.Delta); err != nil {
			log.Printf("Error updating counter: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

	default:
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetMetricJSON(w http.ResponseWriter, r *http.Request) {
	var metric models.Metrics

	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	switch metric.MType {
	case "gauge":
		value, exists := h.storage.GetGauge(metric.ID)
		if !exists {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		metric.Value = &value

	case "counter":
		value, exists := h.storage.GetCounter(metric.ID)
		if !exists {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		metric.Delta = &value

	default:
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	resp, err := json.Marshal(metric)
	if err != nil {
		log.Printf("Error marshaling response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}
