package main

import (
	"flag"
	"log"
	"time"

	"go-metrics-and-alerts/internal/agent"
)

func main() {
	addr := flag.String("a", "localhost:8080", "server address")
	reportInterval := flag.Int("r", 10, "report interval in seconds")
	pollInterval := flag.Int("p", 2, "poll interval in seconds")
	flag.Parse()

	serverURL := "http://" + *addr
	a := agent.New(serverURL, time.Duration(*pollInterval)*time.Second, time.Duration(*reportInterval)*time.Second)

	log.Printf("Agent starting, server: %s, poll: %ds, report: %ds", serverURL, *pollInterval, *reportInterval)
	log.Fatal(a.Run())
}
