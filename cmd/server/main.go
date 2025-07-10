package main

import (
	"flag"
	"log"
	"net/http"

	"go-metrics-and-alerts/internal/handler"
	"go-metrics-and-alerts/internal/repository"

	"github.com/go-chi/chi/v5"
)

func main() {
	addr := flag.String("a", "localhost:8080", "server address")
	flag.Parse()

	storage := repository.NewMemStorage()
	h := handler.New(storage)

	r := chi.NewRouter()

	r.Post("/update/{type}/{name}/{value}", h.UpdateMetric)
	r.Get("/value/{type}/{name}", h.GetMetric)
	r.Get("/", h.ListMetrics)

	log.Printf("Starting server on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, r))
}
