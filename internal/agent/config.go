// Package agent contains the background process gathering metrics.
package agent

import (
	"encoding/json"
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
	CryptoKeyPath  string
	GRPCAddress    string
}

// ParseConfig builds Config from flags and environment variables.
func ParseConfig() *Config {
	fileCfg := loadAgentConfigFile()

	addrDefault := "localhost:8080"
	if fileCfg != nil && fileCfg.Address != "" {
		addrDefault = fileCfg.Address
	}

	reportDefault := 10
	if fileCfg != nil && fileCfg.ReportInterval != "" {
		if d, err := time.ParseDuration(fileCfg.ReportInterval); err == nil {
			reportDefault = int(d / time.Second)
		}
	}

	pollDefault := 2
	if fileCfg != nil && fileCfg.PollInterval != "" {
		if d, err := time.ParseDuration(fileCfg.PollInterval); err == nil {
			pollDefault = int(d / time.Second)
		}
	}

	cryptoDefault := ""
	if fileCfg != nil && fileCfg.CryptoKey != "" {
		cryptoDefault = fileCfg.CryptoKey
	}

	addr := flag.String("a", addrDefault, "server address")
	reportInterval := flag.Int("r", reportDefault, "report interval in seconds")
	pollInterval := flag.Int("p", pollDefault, "poll interval in seconds")
	keyFlag := flag.String("k", "", "hash key")
	limitFlag := flag.Int("l", 1, "rate limit")
	cryptoKeyFlag := flag.String("crypto-key", cryptoDefault, "path to public key")
	grpcAddrFlag := flag.String("grpc-address", "", "grpc server address")
	configFlag := flag.String("config", "", "path to config file")
	shortConfigFlag := flag.String("c", "", "path to config file (shorthand)")
	flag.Parse()
	_ = configFlag
	_ = shortConfigFlag

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

	finalKey := strings.TrimSpace(*keyFlag)
	if envKey := os.Getenv("KEY"); envKey != "" {
		finalKey = strings.TrimSpace(envKey)
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

	finalCryptoKey := *cryptoKeyFlag
	if envCrypto := os.Getenv("CRYPTO_KEY"); envCrypto != "" {
		finalCryptoKey = envCrypto
	}

	finalGRPCAddr := *grpcAddrFlag
	if envGRPC := os.Getenv("GRPC_ADDRESS"); envGRPC != "" {
		finalGRPCAddr = envGRPC
	}

	return &Config{
		ServerURL:      "http://" + finalAddr,
		PollInterval:   time.Duration(finalPollInterval) * time.Second,
		ReportInterval: time.Duration(finalReportInterval) * time.Second,
		Key:            finalKey,
		RateLimit:      finalLimit,
		CryptoKeyPath:  finalCryptoKey,
		GRPCAddress:    finalGRPCAddr,
	}
}

type agentFileConfig struct {
	Address        string `json:"address"`
	ReportInterval string `json:"report_interval"`
	PollInterval   string `json:"poll_interval"`
	CryptoKey      string `json:"crypto_key"`
}

func loadAgentConfigFile() *agentFileConfig {
	path := getConfigPathFromArgs()
	if path == "" {
		if env := os.Getenv("CONFIG"); env != "" {
			path = env
		}
	}
	if path == "" {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var cfg agentFileConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil
	}
	return &cfg
}

func getConfigPathFromArgs() string {
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-config=") {
			return strings.TrimPrefix(arg, "-config=")
		}
		if strings.HasPrefix(arg, "-c=") {
			return strings.TrimPrefix(arg, "-c=")
		}
		if arg == "-config" || arg == "-c" {
			if i+1 < len(args) {
				return args[i+1]
			}
		}
	}
	return ""
}
