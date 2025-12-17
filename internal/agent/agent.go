package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	mathrand "math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"

	models "go-metrics-and-alerts/internal/model"
)

const encryptedHeader = "X-Encrypted"

type Agent struct {
	config      *Config
	randomValue float64
	client      *http.Client
	metricsMu   sync.Mutex
	metrics     map[string]interface{}
	pollCount   int64
	publicKey   *rsa.PublicKey
	realIP      string
}

func New(config *Config) *Agent {
	a := &Agent{
		config:  config,
		client:  &http.Client{},
		metrics: make(map[string]interface{}),
	}
	if config != nil && config.CryptoKeyPath != "" {
		key, err := loadPublicKey(config.CryptoKeyPath)
		if err != nil {
			log.Fatalf("load public key: %v", err)
		}
		a.publicKey = key
	}
	if config != nil {
		a.realIP = detectLocalIP(config.ServerURL)
	}
	return a
}

func (a *Agent) Run(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	pollTicker := time.NewTicker(a.config.PollInterval)
	reportTicker := time.NewTicker(a.config.ReportInterval)
	defer pollTicker.Stop()
	defer reportTicker.Stop()

	log.Printf("Agent starting, server: %s, poll: %v, report: %v",
		a.config.ServerURL, a.config.PollInterval, a.config.ReportInterval)

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case <-pollTicker.C:
				a.collectRuntimeMetrics()
			}
		}
	}()

	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case <-pollTicker.C:
				a.collectSystemMetrics()
			}
		}
	}()

	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case <-reportTicker.C:
				snap := a.buildSnapshot()
				a.dispatchSnapshot(snap)
			}
		}
	}()

	<-ctx.Done()
	wg.Wait()
	snap := a.buildSnapshot()
	a.dispatchSnapshot(snap)

	return nil
}

func (a *Agent) buildSnapshot() map[string]interface{} {
	snap := make(map[string]interface{})
	a.metricsMu.Lock()
	for k, v := range a.metrics {
		snap[k] = v
	}
	pc := a.pollCount
	a.pollCount = 0
	a.metricsMu.Unlock()

	if pc < 0 {
		pc = 0
	}
	snap["PollCount"] = pc
	return snap
}

func (a *Agent) dispatchSnapshot(snap map[string]interface{}) {
	if len(snap) == 0 {
		return
	}
	if a.config.RateLimit > 1 {
		a.sendMetricsWithRetry(snap)
	} else {
		a.sendMetricsBatchWithRetry(snap)
	}
}

func (a *Agent) collectRuntimeMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	a.metricsMu.Lock()
	a.metrics["Alloc"] = float64(m.Alloc)
	a.metrics["BuckHashSys"] = float64(m.BuckHashSys)
	a.metrics["Frees"] = float64(m.Frees)
	a.metrics["GCCPUFraction"] = m.GCCPUFraction
	a.metrics["GCSys"] = float64(m.GCSys)
	a.metrics["HeapAlloc"] = float64(m.HeapAlloc)
	a.metrics["HeapIdle"] = float64(m.HeapIdle)
	a.metrics["HeapInuse"] = float64(m.HeapInuse)
	a.metrics["HeapObjects"] = float64(m.HeapObjects)
	a.metrics["HeapReleased"] = float64(m.HeapReleased)
	a.metrics["HeapSys"] = float64(m.HeapSys)
	a.metrics["LastGC"] = float64(m.LastGC)
	a.metrics["Lookups"] = float64(m.Lookups)
	a.metrics["MCacheInuse"] = float64(m.MCacheInuse)
	a.metrics["MCacheSys"] = float64(m.MCacheSys)
	a.metrics["MSpanInuse"] = float64(m.MSpanInuse)
	a.metrics["MSpanSys"] = float64(m.MSpanSys)
	a.metrics["Mallocs"] = float64(m.Mallocs)
	a.metrics["NextGC"] = float64(m.NextGC)
	a.metrics["NumForcedGC"] = float64(m.NumForcedGC)
	a.metrics["NumGC"] = float64(m.NumGC)
	a.metrics["OtherSys"] = float64(m.OtherSys)
	a.metrics["PauseTotalNs"] = float64(m.PauseTotalNs)
	a.metrics["StackInuse"] = float64(m.StackInuse)
	a.metrics["StackSys"] = float64(m.StackSys)
	a.metrics["Sys"] = float64(m.Sys)
	a.metrics["TotalAlloc"] = float64(m.TotalAlloc)

	a.randomValue = mathrand.Float64()
	a.metrics["RandomValue"] = a.randomValue
	a.pollCount++
	a.metricsMu.Unlock()
}

func (a *Agent) collectSystemMetrics() {
	vm, err := mem.VirtualMemory()
	if err == nil {
		a.metricsMu.Lock()
		a.metrics["TotalMemory"] = float64(vm.Total)
		a.metrics["FreeMemory"] = float64(vm.Free)
		a.metricsMu.Unlock()
	}

	per, err := cpu.Percent(0, true)
	if err == nil {
		a.metricsMu.Lock()
		for i := range per {
			name := fmt.Sprintf("CPUutilization%d", i+1)
			a.metrics[name] = per[i]
		}
		a.metricsMu.Unlock()
	}
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
			metric.MType = "gauge"
			metric.Value = &v
		case int64:
			metric.MType = "counter"
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

	var hashHeader string
	if a.config.Key != "" {
		h := hmac.New(sha256.New, []byte(a.config.Key))
		h.Write(jsonData)
		hashHeader = hex.EncodeToString(h.Sum(nil))
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(jsonData); err != nil {
		return fmt.Errorf("error compressing batch: %w", err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("error closing gzip: %w", err)
	}

	payload := buf.Bytes()
	encrypted := false
	if a.publicKey != nil {
		payload, err = encryptPayload(a.publicKey, payload)
		if err != nil {
			return fmt.Errorf("encrypt batch: %w", err)
		}
		encrypted = true
	}

	url := fmt.Sprintf("%s/updates/", a.config.ServerURL)
	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")
	if a.realIP != "" {
		req.Header.Set("X-Real-IP", a.realIP)
	}
	if hashHeader != "" {
		req.Header.Set("HashSHA256", hashHeader)
	}
	if encrypted {
		req.Header.Set(encryptedHeader, "1")
	}

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
	concurrency := a.config.RateLimit
	if concurrency <= 0 {
		concurrency = 1
	}

	type item struct {
		n string
		v interface{}
	}

	tasks := make(chan item)
	var wg sync.WaitGroup
	var sentMu sync.Mutex
	sent := 0

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for it := range tasks {
				if err := a.sendSingleMetricWithRetry(it.n, it.v); err != nil {
					log.Printf("Failed to send metric %s after all retries: %v", it.n, err)
					continue
				}
				sentMu.Lock()
				sent++
				sentMu.Unlock()
			}
		}()
	}

	go func() {
		for name, value := range metrics {
			tasks <- item{n: name, v: value}
		}
		close(tasks)
	}()

	wg.Wait()
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
		metric.MType = "gauge"
		metric.Value = &v
	case int64:
		metric.MType = "counter"
		metric.Delta = &v
	default:
		return fmt.Errorf("unsupported type for %s", name)
	}

	jsonData, err := json.Marshal(metric)
	if err != nil {
		return fmt.Errorf("marshaling metric %s: %w", name, err)
	}

	var hashHeader string
	if a.config.Key != "" {
		h := hmac.New(sha256.New, []byte(a.config.Key))
		h.Write(jsonData)
		hashHeader = hex.EncodeToString(h.Sum(nil))
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(jsonData); err != nil {
		return fmt.Errorf("gzip write error for %s: %w", name, err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("gzip close error for %s: %w", name, err)
	}

	payload := buf.Bytes()
	encrypted := false
	if a.publicKey != nil {
		payload, err = encryptPayload(a.publicKey, payload)
		if err != nil {
			return fmt.Errorf("encrypt metric %s: %w", name, err)
		}
		encrypted = true
	}

	url := fmt.Sprintf("%s/update", a.config.ServerURL)
	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("creating request for %s: %w", name, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")
	if a.realIP != "" {
		req.Header.Set("X-Real-IP", a.realIP)
	}
	if hashHeader != "" {
		req.Header.Set("HashSHA256", hashHeader)
	}
	if encrypted {
		req.Header.Set(encryptedHeader, "1")
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending metric %s: %w", name, err)
	}
	resp.Body.Close()

	return nil
}

func loadPublicKey(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("invalid public key data")
	}

	pubAny, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	pub, ok := pubAny.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA public key")
	}

	return pub, nil
}

func encryptPayload(key *rsa.PublicKey, data []byte) ([]byte, error) {
	chunkSize := key.Size() - 11
	if chunkSize <= 0 {
		return nil, fmt.Errorf("invalid key size")
	}

	var out bytes.Buffer
	for offset := 0; offset < len(data); offset += chunkSize {
		end := offset + chunkSize
		if end > len(data) {
			end = len(data)
		}
		encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, key, data[offset:end])
		if err != nil {
			return nil, err
		}
		out.Write(encrypted)
	}
	return out.Bytes(), nil
}

func detectLocalIP(serverURL string) string {
	u, err := url.Parse(serverURL)
	if err != nil {
		return ""
	}

	host := u.Host
	if host == "" {
		host = u.Path
	}
	if host == "" {
		return ""
	}

	if !strings.Contains(host, ":") {
		host += ":80"
	}

	conn, err := net.Dial("udp", host)
	if err != nil {
		return ""
	}
	defer conn.Close()

	udp, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok || udp.IP == nil {
		return ""
	}
	return udp.IP.String()
}
