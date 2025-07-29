package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"go-metrics-and-alerts/internal/handler"
	"go-metrics-and-alerts/internal/middleware"
	"go-metrics-and-alerts/internal/repository"

	"github.com/go-chi/chi/v5"
)

func main() {
	addr := flag.String("a", "localhost:8080", "server address")
	flag.Parse()

	finalAddr := *addr
	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		finalAddr = envAddr
	}

	storage := repository.NewMemStorage()
	h := handler.New(storage)

	r := chi.NewRouter()

	r.Use(middleware.WithLogging)
	r.Use(middleware.WithGzipDecompress)
	r.Use(middleware.WithGzip)

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
