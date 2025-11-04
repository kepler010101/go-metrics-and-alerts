// Package agent contains the background process gathering metrics.
package agent

import (
	"flag"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config stores runtime parameters for the agent.
type Config struct {
	ServerURL      string
	PollInterval   time.Duration
	ReportInterval time.Duration
	Key            string
	RateLimit      int
}

// ParseConfig builds Config from flags and environment variables.
func ParseConfig() *Config {
	addr := flag.String("a", "localhost:8080", "server address")
	reportInterval := flag.Int("r", 10, "report interval in seconds")
	pollInterval := flag.Int("p", 2, "poll interval in seconds")
	keyFlag := flag.String("k", "", "hash key")
	limitFlag := flag.Int("l", 1, "rate limit")
	flag.Parse()

	finalAddr := *addr
	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		finalAddr = envAddr
	}

	finalReportInterval := *reportInterval
	if envReport := os.Getenv("REPORT_INTERVAL"); envReport != "" {
		if val, err := strconv.Atoi(envReport); err == nil {
			finalReportInterval = val
		}
	}

	finalPollInterval := *pollInterval
	if envPoll := os.Getenv("POLL_INTERVAL"); envPoll != "" {
		if val, err := strconv.Atoi(envPoll); err == nil {
			finalPollInterval = val
		}
	}

	finalKey := *keyFlag
	if envKey := os.Getenv("KEY"); envKey != "" {
		finalKey = envKey
	}
	if finalKey != "" {
		if data, err := os.ReadFile(finalKey); err == nil {
			finalKey = strings.TrimSpace(string(data))
		}
	}

	finalLimit := *limitFlag
	if envLimit := os.Getenv("RATE_LIMIT"); envLimit != "" {
		if val, err := strconv.Atoi(envLimit); err == nil {
			finalLimit = val
		}
	}

	return &Config{
		ServerURL:      "http://" + finalAddr,
		PollInterval:   time.Duration(finalPollInterval) * time.Second,
		ReportInterval: time.Duration(finalReportInterval) * time.Second,
		Key:            finalKey,
		RateLimit:      finalLimit,
	}
}
