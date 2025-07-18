package agent

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"runtime"
	"time"
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
			a.sendMetrics(metrics)
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

func (a *Agent) sendMetrics(metrics map[string]interface{}) {
	sent := 0
	for name, value := range metrics {
		if err := a.sendSingleMetric(name, value); err != nil {
			log.Printf("Err sending metric: %v", err)
			continue
		}
		sent++
	}
	log.Printf("Sent %d metrics", sent)
}

func (a *Agent) sendSingleMetric(name string, value interface{}) error {
	var url string
	switch v := value.(type) {
	case float64:
		url = fmt.Sprintf("%s/update/gauge/%s/%g", a.config.ServerURL, name, v)
	case int64:
		url = fmt.Sprintf("%s/update/counter/%s/%d", a.config.ServerURL, name, v)
	default:
		return fmt.Errorf("unsupported type for %s", name)
	}

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("creating request for %s: %w", name, err)
	}
	req.Header.Set("Content-Type", "text/plain")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending metric %s: %w", name, err)
	}
	resp.Body.Close()

	return nil
}
