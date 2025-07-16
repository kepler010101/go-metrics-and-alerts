package agent

import (
	"flag"
	"time"
)

type Config struct {
	ServerURL      string
	PollInterval   time.Duration
	ReportInterval time.Duration
}

func ParseConfig() *Config {
	addr := flag.String("a", "localhost:8080", "server address")
	reportInterval := flag.Int("r", 10, "report interval in seconds")
	pollInterval := flag.Int("p", 2, "poll interval in seconds")
	flag.Parse()

	return &Config{
		ServerURL:      "http://" + *addr,
		PollInterval:   time.Duration(*pollInterval) * time.Second,
		ReportInterval: time.Duration(*reportInterval) * time.Second,
	}
}
