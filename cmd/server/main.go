package main

import (
	"context"
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
	"go-metrics-and-alerts/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
)

var (
	storage         repository.Repository
	fileStoragePath string
	storeInterval   int
	pool            *pgxpool.Pool
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

	ctx := context.Background()
	var metrics []models.Metrics

	gauges := storage.GetAllGauges(ctx)
	for name, value := range gauges {
		metric := models.Metrics{
			ID:    name,
			MType: models.TypeGauge,
			Value: &value,
		}
		metrics = append(metrics, metric)
	}

	counters := storage.GetAllCounters(ctx)
	for name, delta := range counters {
		metric := models.Metrics{
			ID:    name,
			MType: models.TypeCounter,
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

	ctx := context.Background()
	for _, metric := range metrics {
		switch metric.MType {
		case models.TypeGauge:
			if metric.Value != nil {
				storage.UpdateGauge(ctx, metric.ID, *metric.Value)
			}
		case models.TypeCounter:
			if metric.Delta != nil {
				storage.UpdateCounter(ctx, metric.ID, *metric.Delta)
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
		
		pool, err = pgxpool.New(context.Background(), finalDSN)
		if err != nil {
			log.Fatal("Failed to create connection pool:", err)
		}
		defer pool.Close()
		
		if err := pool.Ping(context.Background()); err != nil {
			log.Fatal("Failed to ping database:", err)
		}
		
		db, err := sql.Open("postgres", finalDSN)
		if err != nil {
			log.Fatal("Failed to connect to database:", err)
		}
		defer db.Close()
		
		if err := runMigrations(db); err != nil && err != migrate.ErrNoChange {
			log.Printf("Failed to run migrations: %v", err)
		}

		storage, err = repository.NewPostgresStorage(pool)
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

	svc := service.NewMetricsService(storage)
	h := handler.New(svc)

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
		if pool == nil {
			http.Error(w, "Database not configured", http.StatusInternalServerError)
			return
		}
		if err := pool.Ping(r.Context()); err != nil {
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
		log.Fatal("Server failed:", err)
	}
}