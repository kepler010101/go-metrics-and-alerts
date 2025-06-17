package main

import (
	"log"

	"go-metrics-and-alerts/internal/agent"
)

func main() {
	a := agent.New("http://localhost:8080")
	log.Fatal(a.Run())
}
