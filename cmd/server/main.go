package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"go-metrics-and-alerts/internal/handler"
	"go-metrics-and-alerts/internal/middleware"
	models "go-metrics-and-alerts/internal/model"
	"go-metrics-and-alerts/internal/repository"

	"github.com/go-chi/chi/v5"
	_ "github.com/lib/pq"
)

var (
	storage         repository.Repository
	fileStoragePath string
	storeInterval   int
	db              *sql.DB
)

func saveToFile() error {
	var metrics []models.Metrics

	gauges := storage.GetAllGauges()
	for name, value := range gauges {
		metric := models.Metrics{
			ID:    name,
			MType: "gauge",
			Value: &value,
		}
		metrics = append(metrics, metric)
	}

	counters := storage.GetAllCounters()
	for name, delta := range counters {
		metric := models.Metrics{
			ID:    name,
			MType: "counter",
			Delta: &delta,
		}
		metrics = append(metrics, metric)
	}

	data, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	return os.WriteFile(fileStoragePath, data, 0666)
}

func loadFromFile() error {
	data, err := os.ReadFile(fileStoragePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var metrics []models.Metrics
	if err := json.Unmarshal(data, &metrics); err != nil {
		return err
	}

	for _, metric := range metrics {
		switch metric.MType {
		case "gauge":
			if metric.Value != nil {
				storage.UpdateGauge(metric.ID, *metric.Value)
			}
		case "counter":
			if metric.Delta != nil {
				storage.UpdateCounter(metric.ID, *metric.Delta)
			}
		}
	}

	return nil
}

func main() {
	addr := flag.String("a", "localhost:8080", "server address")
	storeIntervalFlag := flag.Int("i", 300, "store interval in seconds")
	fileStoragePathFlag := flag.String("f", "/tmp/metrics-db.json", "file storage path")
	restore := flag.Bool("r", true, "restore from file")
	dsn := flag.String("d", "", "database DSN")
	flag.Parse()

	finalAddr := *addr
	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		finalAddr = envAddr
	}

	storeInterval = *storeIntervalFlag
	if envInterval := os.Getenv("STORE_INTERVAL"); envInterval != "" {
		if val, err := strconv.Atoi(envInterval); err == nil {
			storeInterval = val
		}
	}

	fileStoragePath = *fileStoragePathFlag
	if envPath := os.Getenv("FILE_STORAGE_PATH"); envPath != "" {
		fileStoragePath = envPath
	}

	finalRestore := *restore
	if envRestore := os.Getenv("RESTORE"); envRestore != "" {
		if val, err := strconv.ParseBool(envRestore); err == nil {
			finalRestore = val
		}
	}

	finalDSN := *dsn
	if envDSN := os.Getenv("DATABASE_DSN"); envDSN != "" {
		finalDSN = envDSN
	}

	if finalDSN != "" {
		var err error
		db, err = sql.Open("postgres", finalDSN)
		if err != nil {
			log.Printf("Failed to connect to database: %v", err)
		} else {
			defer db.Close()
		}
	}

	storage = repository.NewMemStorage()

	if finalRestore {
		if err := loadFromFile(); err != nil {
			log.Printf("Failed to load from file: %v", err)
		}
	}

	defer func() {
		if err := saveToFile(); err != nil {
			log.Printf("Failed to save to file: %v", err)
		}
	}()

	if storeInterval > 0 {
		go func() {
			ticker := time.NewTicker(time.Duration(storeInterval) * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				if err := saveToFile(); err != nil {
					log.Printf("Failed to save to file: %v", err)
				}
			}
		}()
	}

	h := handler.New(storage)

	if storeInterval == 0 {
		handler.SyncSaveFunc = func() {
			if err := saveToFile(); err != nil {
				log.Printf("Failed to sync save: %v", err)
			}
		}
	}

	r := chi.NewRouter()

	r.Use(middleware.WithLogging)
	r.Use(middleware.WithGzipDecompress)
	r.Use(middleware.WithGzip)

	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		if db == nil {
			http.Error(w, "Database not configured", http.StatusInternalServerError)
			return
		}
		if err := db.Ping(); err != nil {
			http.Error(w, "Database connection failed", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	r.Post("/update/{type}/{name}/{value}", h.UpdateMetric)
	r.Post("/update", h.UpdateMetricJSON)
	r.Post("/update/", h.UpdateMetricJSON)
	r.Get("/value/{type}/{name}", h.GetMetric)
	r.Post("/value", h.GetMetricJSON)
	r.Post("/value/", h.GetMetricJSON)
	r.Get("/", h.ListMetrics)

	log.Printf("Starting server on %s", finalAddr)
	if err := http.ListenAndServe(finalAddr, r); err != nil {
		log.Fatal("Server fail:", err)
	}
}
