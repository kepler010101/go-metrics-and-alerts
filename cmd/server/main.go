package main

import (
	"log"
	"net/http"

	"go-metrics-and-alerts/internal/handler"
	"go-metrics-and-alerts/internal/repository"

	"github.com/go-chi/chi/v5"
)

func main() {
	storage := repository.NewMemStorage()
	h := handler.New(storage)

	r := chi.NewRouter()

	r.Post("/update/{type}/{name}/{value}", h.UpdateMetric)
	r.Get("/value/{type}/{name}", h.GetMetric)
	r.Get("/", h.ListMetrics)

	log.Fatal(http.ListenAndServe(":8080", r))
}
