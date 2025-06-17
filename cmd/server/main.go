package main

import (
	"log"
	"net/http"

	"go-metrics-and-alerts/internal/handler"
	"go-metrics-and-alerts/internal/repository"
)

func main() {
	storage := repository.NewMemStorage()
	h := handler.New(storage)

	http.HandleFunc("/update/", h.UpdateMetric)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
