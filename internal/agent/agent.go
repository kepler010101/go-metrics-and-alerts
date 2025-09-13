package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"runtime"
	"time"

	models "go-metrics-and-alerts/internal/model"
)

type Agent struct {
	config      *Config
	randomValue float64
	client      *http.Client
}

func New(config *Config) *Agent {
	return &Agent{
		config: config,
		client: &http.Client{},
	}
}

func (a *Agent) Run() error {
	pollTicker := time.NewTicker(a.config.PollInterval)
	reportTicker := time.NewTicker(a.config.ReportInterval)

	metrics := make(map[string]interface{})
	pollsSinceLastReport := int64(0)

	log.Printf("Agent starting, server: %s, poll: %v, report: %v",
		a.config.ServerURL, a.config.PollInterval, a.config.ReportInterval)

	for {
		select {
		case <-pollTicker.C:
			a.collectMetrics(metrics)
			pollsSinceLastReport++
			log.Printf("Collected metrics, polls since last report: %d", pollsSinceLastReport)

		case <-reportTicker.C:
			metrics["PollCount"] = pollsSinceLastReport
			log.Printf("Sending %d metrics", len(metrics))
			a.sendMetricsBatchWithRetry(metrics)
			pollsSinceLastReport = 0
		}
	}
}

func (a *Agent) collectMetrics(metrics map[string]interface{}) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	metrics["Alloc"] = float64(m.Alloc)
	metrics["BuckHashSys"] = float64(m.BuckHashSys)
	metrics["Frees"] = float64(m.Frees)
	metrics["GCCPUFraction"] = m.GCCPUFraction
	metrics["GCSys"] = float64(m.GCSys)
	metrics["HeapAlloc"] = float64(m.HeapAlloc)
	metrics["HeapIdle"] = float64(m.HeapIdle)
	metrics["HeapInuse"] = float64(m.HeapInuse)
	metrics["HeapObjects"] = float64(m.HeapObjects)
	metrics["HeapReleased"] = float64(m.HeapReleased)
	metrics["HeapSys"] = float64(m.HeapSys)
	metrics["LastGC"] = float64(m.LastGC)
	metrics["Lookups"] = float64(m.Lookups)
	metrics["MCacheInuse"] = float64(m.MCacheInuse)
	metrics["MCacheSys"] = float64(m.MCacheSys)
	metrics["MSpanInuse"] = float64(m.MSpanInuse)
	metrics["MSpanSys"] = float64(m.MSpanSys)
	metrics["Mallocs"] = float64(m.Mallocs)
	metrics["NextGC"] = float64(m.NextGC)
	metrics["NumForcedGC"] = float64(m.NumForcedGC)
	metrics["NumGC"] = float64(m.NumGC)
	metrics["OtherSys"] = float64(m.OtherSys)
	metrics["PauseTotalNs"] = float64(m.PauseTotalNs)
	metrics["StackInuse"] = float64(m.StackInuse)
	metrics["StackSys"] = float64(m.StackSys)
	metrics["Sys"] = float64(m.Sys)
	metrics["TotalAlloc"] = float64(m.TotalAlloc)

	a.randomValue = rand.Float64()
	metrics["RandomValue"] = a.randomValue
}

func (a *Agent) sendMetricsBatchWithRetry(metrics map[string]interface{}) {
	retryIntervals := []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}

	for attempt := 0; attempt <= len(retryIntervals); attempt++ {
		if attempt > 0 {
			delay := retryIntervals[attempt-1]
			log.Printf("Retrying after %v (attempt %d)", delay, attempt)
			time.Sleep(delay)
		}

		err := a.sendMetricsBatch(metrics)
		if err == nil {
			return
		}

		if attempt == len(retryIntervals) {
			log.Printf("Failed to send batch after all retries, falling back to single requests")
			a.sendMetricsWithRetry(metrics)
			return
		}

		log.Printf("Error sending batch (attempt %d): %v", attempt+1, err)
	}
}

func (a *Agent) sendMetricsBatch(metrics map[string]interface{}) error {
	var batch []models.Metrics

	for name, value := range metrics {
		var metric models.Metrics
		metric.ID = name

		switch v := value.(type) {
		case float64:
			metric.MType = models.TypeGauge
			metric.Value = &v
		case int64:
			metric.MType = models.TypeCounter
			metric.Delta = &v
		default:
			log.Printf("Unsupported type for %s", name)
			continue
		}

		batch = append(batch, metric)
	}

	if len(batch) == 0 {
		return nil
	}

	jsonData, err := json.Marshal(batch)
	if err != nil {
		return fmt.Errorf("error marshaling batch: %w", err)
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(jsonData); err != nil {
		return fmt.Errorf("error compressing batch: %w", err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("error closing gzip: %w", err)
	}

	url := fmt.Sprintf("%s/updates/", a.config.ServerURL)
	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending batch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned non-200: %d", resp.StatusCode)
	}

	log.Printf("Successfully sent batch of %d metrics", len(batch))
	return nil
}

func (a *Agent) sendMetricsWithRetry(metrics map[string]interface{}) {
	sent := 0
	for name, value := range metrics {
		if err := a.sendSingleMetricWithRetry(name, value); err != nil {
			log.Printf("Failed to send metric %s after all retries: %v", name, err)
			continue
		}
		sent++
	}
	log.Printf("Sent %d metrics", sent)
}

func (a *Agent) sendSingleMetricWithRetry(name string, value interface{}) error {
	retryIntervals := []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}

	for attempt := 0; attempt <= len(retryIntervals); attempt++ {
		if attempt > 0 {
			delay := retryIntervals[attempt-1]
			time.Sleep(delay)
		}

		err := a.sendSingleMetric(name, value)
		if err == nil {
			return nil
		}

		if attempt == len(retryIntervals) {
			return err
		}
	}

	return fmt.Errorf("failed after all retries")
}

func (a *Agent) sendSingleMetric(name string, value interface{}) error {
	var metric models.Metrics
	metric.ID = name

	switch v := value.(type) {
	case float64:
		metric.MType = models.TypeGauge
		metric.Value = &v
	case int64:
		metric.MType = models.TypeCounter
		metric.Delta = &v
	default:
		return fmt.Errorf("unsupported type for %s", name)
	}

	jsonData, err := json.Marshal(metric)
	if err != nil {
		return fmt.Errorf("marshaling metric %s: %w", name, err)
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(jsonData); err != nil {
		return fmt.Errorf("gzip write error for %s: %w", name, err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("gzip close error for %s: %w", name, err)
	}

	url := fmt.Sprintf("%s/update", a.config.ServerURL)
	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return fmt.Errorf("creating request for %s: %w", name, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending metric %s: %w", name, err)
	}
	resp.Body.Close()

	return nil
}