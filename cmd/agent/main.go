package main

import (
	"log"

	"go-metrics-and-alerts/internal/agent"
)

func main() {
	cfg := agent.ParseConfig()
	a := agent.New(cfg)

	log.Fatal(a.Run())
}
