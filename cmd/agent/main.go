package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"go-metrics-and-alerts/internal/agent"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	defer stop()

	log.Printf("Build version: %s", fallback(buildVersion))
	log.Printf("Build date: %s", fallback(buildDate))
	log.Printf("Build commit: %s", fallback(buildCommit))

	cfg := agent.ParseConfig()
	a := agent.New(cfg)

	if err := a.Run(ctx); err != nil {
		log.Fatal("Agent failed:", err)
	}
}

func fallback(value string) string {
	if value == "" {
		return "N/A"
	}
	return value
}
