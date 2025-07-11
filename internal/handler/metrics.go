package handler

import (
	"html/template"
	"net/http"
	"strconv"

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
		w.Write([]byte(strconv.FormatFloat(value, 'g', -1, 64)))

	case "counter":
		value, exists := h.storage.GetCounter(metricName)
		if !exists {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(strconv.FormatInt(value, 10)))

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

	t, _ := template.New("metrics").Parse(tmpl)

	data := struct {
		Gauges   map[string]float64
		Counters map[string]int64
	}{
		Gauges:   h.storage.GetAllGauges(),
		Counters: h.storage.GetAllCounters(),
	}

	t.Execute(w, data)
}
