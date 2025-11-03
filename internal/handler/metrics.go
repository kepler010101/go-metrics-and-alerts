package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"go-metrics-and-alerts/internal/audit"
	models "go-metrics-and-alerts/internal/model"
	"go-metrics-and-alerts/internal/repository"

	"github.com/go-chi/chi/v5"
)

var SyncSaveFunc func()
var SecretKey string

var metricsTemplate = template.Must(template.New("metrics").Parse(`<html><body><h1>Metrics</h1>
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
</body></html>`))

type Handler struct {
	storage repository.Repository
	auditor audit.Notifier
}

func New(storage repository.Repository) *Handler {
	return &Handler{storage: storage}
}

func (h *Handler) SetAuditor(a audit.Notifier) {
	h.auditor = a
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

	if SyncSaveFunc != nil {
		SyncSaveFunc()
	}

	h.publishAudit(r, []string{metricName})

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

	data := struct {
		Gauges   map[string]float64
		Counters map[string]int64
	}{
		Gauges:   h.storage.GetAllGauges(),
		Counters: h.storage.GetAllCounters(),
	}

	if err := metricsTemplate.Execute(w, data); err != nil {
		log.Printf("Error executing template: %v", err)
	}
}

func (h *Handler) UpdateMetricJSON(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if SecretKey != "" {
		hmacHash := hmac.New(sha256.New, []byte(SecretKey))
		hmacHash.Write(body)
		expected := hex.EncodeToString(hmacHash.Sum(nil))
		got := r.Header.Get("HashSHA256")
		if got == "" || got != expected {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
	}

	var metric models.Metrics
	if err := json.Unmarshal(body, &metric); err != nil {
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

	if SyncSaveFunc != nil {
		SyncSaveFunc()
	}

	h.publishAudit(r, []string{metric.ID})

	w.Header().Set("Content-Type", "application/json")
	resp, err := json.Marshal(metric)
	if err != nil {
		log.Printf("Error marshaling response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if SecretKey != "" {
		hmacResp := hmac.New(sha256.New, []byte(SecretKey))
		hmacResp.Write(resp)
		w.Header().Set("HashSHA256", hex.EncodeToString(hmacResp.Sum(nil)))
	}

	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func (h *Handler) GetMetricJSON(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if SecretKey != "" {
		hmacHash := hmac.New(sha256.New, []byte(SecretKey))
		hmacHash.Write(body)
		expected := hex.EncodeToString(hmacHash.Sum(nil))
		got := r.Header.Get("HashSHA256")
		if got == "" || got != expected {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
	}

	var metric models.Metrics
	if err := json.Unmarshal(body, &metric); err != nil {
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

	if SecretKey != "" {
		hmacResp := hmac.New(sha256.New, []byte(SecretKey))
		hmacResp.Write(resp)
		w.Header().Set("HashSHA256", hex.EncodeToString(hmacResp.Sum(nil)))
	}

	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func (h *Handler) UpdateMetricsBatch(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if SecretKey != "" {
		hmacHash := hmac.New(sha256.New, []byte(SecretKey))
		hmacHash.Write(body)
		expected := hex.EncodeToString(hmacHash.Sum(nil))
		got := r.Header.Get("HashSHA256")
		if got == "" || got != expected {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
	}

	var metrics []models.Metrics
	if err := json.Unmarshal(body, &metrics); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if len(metrics) == 0 {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if err := h.storage.UpdateBatch(metrics); err != nil {
		log.Printf("Error updating metrics batch: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if SyncSaveFunc != nil {
		SyncSaveFunc()
	}

	names := make([]string, 0, len(metrics))
	for _, metric := range metrics {
		if metric.ID != "" {
			names = append(names, metric.ID)
		}
	}

	h.publishAudit(r, names)

	w.Header().Set("Content-Type", "application/json")
	if SecretKey != "" {
		hmacResp := hmac.New(sha256.New, []byte(SecretKey))
		w.Header().Set("HashSHA256", hex.EncodeToString(hmacResp.Sum(nil)))
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) publishAudit(r *http.Request, names []string) {
	if h == nil || h.auditor == nil || len(names) == 0 {
		return
	}

	ip := r.RemoteAddr
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}

	event := audit.Event{
		Timestamp: time.Now().Unix(),
		Metrics:   names,
		IPAddress: ip,
	}
	h.auditor.Publish(event)
}
