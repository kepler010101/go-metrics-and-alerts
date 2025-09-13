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
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

var (
	storage         repository.Repository
	fileStoragePath string
	storeInterval   int
	db              *sql.DB
	useFileStorage  bool
)

func runMigrations(db *sql.DB) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres", driver)
	if err != nil {
		return err
	}

	return m.Up()
}

func saveToFile() error {
	if !useFileStorage {
		return nil
	}

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
    keyFlag := flag.String("k", "", "hash key")
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

    finalKey := *keyFlag
    if envKey := os.Getenv("KEY"); envKey != "" {
        finalKey = envKey
    }

	if finalDSN != "" {
		var err error
		db, err = sql.Open("postgres", finalDSN)
		if err != nil {
			log.Fatal("Failed to connect to database:", err)
		}
		defer db.Close()

		if err := runMigrations(db); err != nil && err != migrate.ErrNoChange {
			log.Printf("Failed to run migrations: %v", err)
		}

		storage, err = repository.NewPostgresStorage(db)
		if err != nil {
			log.Fatal("Failed to create postgres storage:", err)
		}
		useFileStorage = false
	} else if fileStoragePath != "" {
		storage = repository.NewMemStorage()
		useFileStorage = true
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
	} else {
		storage = repository.NewMemStorage()
		useFileStorage = false
	}

	if useFileStorage && storeInterval > 0 {
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

    handler.SecretKey = finalKey

	if useFileStorage && storeInterval == 0 {
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
	r.Post("/updates/", h.UpdateMetricsBatch)

	log.Printf("Starting server on %s", finalAddr)
	if err := http.ListenAndServe(finalAddr, r); err != nil {
		log.Fatal("Server fail:", err)
	}
}
