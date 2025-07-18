package main

import (
	"log"

	"go-metrics-and-alerts/internal/agent"
)

func main() {
	cfg := agent.ParseConfig()
	a := agent.New(cfg)

	if err := a.Run(); err != nil {
		log.Fatal("Agent failed:", err)
	}
}
