package main

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"go-metrics-and-alerts/internal/audit"
	"go-metrics-and-alerts/internal/grpcserver"
	"go-metrics-and-alerts/internal/handler"
	"go-metrics-and-alerts/internal/middleware"
	models "go-metrics-and-alerts/internal/model"
	pb "go-metrics-and-alerts/internal/proto"
	"go-metrics-and-alerts/internal/repository"

	"github.com/go-chi/chi/v5"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
)

var (
	storage         repository.Repository
	fileStoragePath string
	storeInterval   int
	db              *sql.DB
	useFileStorage  bool

	buildVersion string
	buildDate    string
	buildCommit  string
)

func runMigrations(db *sql.DB) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres", driver)
	if err != nil {
		return err
	}

	return m.Up()
}

func saveToFile() error {
	if !useFileStorage {
		return nil
	}

	var metrics []models.Metrics

	gauges := storage.GetAllGauges()
	for name, value := range gauges {
		metric := models.Metrics{
			ID:    name,
			MType: "gauge",
			Value: &value,
		}
		metrics = append(metrics, metric)
	}

	counters := storage.GetAllCounters()
	for name, delta := range counters {
		metric := models.Metrics{
			ID:    name,
			MType: "counter",
			Delta: &delta,
		}
		metrics = append(metrics, metric)
	}

	data, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	return os.WriteFile(fileStoragePath, data, 0666)
}

func loadFromFile() error {
	data, err := os.ReadFile(fileStoragePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var metrics []models.Metrics
	if err := json.Unmarshal(data, &metrics); err != nil {
		return err
	}

	for _, metric := range metrics {
		switch metric.MType {
		case "gauge":
			if metric.Value != nil {
				storage.UpdateGauge(metric.ID, *metric.Value)
			}
		case "counter":
			if metric.Delta != nil {
				storage.UpdateCounter(metric.ID, *metric.Delta)
			}
		}
	}

	return nil
}

func main() {
	fileCfg := loadServerConfigFile()

	addrDefault := "localhost:8080"
	if fileCfg != nil && fileCfg.Address != "" {
		addrDefault = fileCfg.Address
	}

	restoreDefault := true
	if fileCfg != nil && fileCfg.Restore != nil {
		restoreDefault = *fileCfg.Restore
	}

	storeIntervalDefault := 300
	if fileCfg != nil && fileCfg.StoreInterval != "" {
		if d, err := time.ParseDuration(fileCfg.StoreInterval); err == nil {
			storeIntervalDefault = int(d / time.Second)
		}
	}

	filePathDefault := "/tmp/metrics-db.json"
	if fileCfg != nil && fileCfg.StoreFile != "" {
		filePathDefault = fileCfg.StoreFile
	}

	dsnDefault := ""
	if fileCfg != nil && fileCfg.DatabaseDSN != "" {
		dsnDefault = fileCfg.DatabaseDSN
	}

	cryptoDefault := ""
	if fileCfg != nil && fileCfg.CryptoKey != "" {
		cryptoDefault = fileCfg.CryptoKey
	}

	trustedSubnetDefault := ""
	if fileCfg != nil && fileCfg.TrustedSubnet != "" {
		trustedSubnetDefault = fileCfg.TrustedSubnet
	}

	addr := flag.String("a", addrDefault, "server address")
	storeIntervalFlag := flag.Int("i", storeIntervalDefault, "store interval in seconds")
	fileStoragePathFlag := flag.String("f", filePathDefault, "file storage path")
	restore := flag.Bool("r", restoreDefault, "restore from file")
	dsn := flag.String("d", dsnDefault, "database DSN")
	keyFlag := flag.String("k", "", "hash key")
	auditFileFlag := flag.String("audit-file", "", "audit file path")
	auditURLFlag := flag.String("audit-url", "", "audit url")
	cryptoKeyFlag := flag.String("crypto-key", cryptoDefault, "path to private key")
	trustedSubnetFlag := flag.String("t", trustedSubnetDefault, "trusted subnet in CIDR")
	grpcAddrFlag := flag.String("grpc-address", "", "grpc server address")
	configFlag := flag.String("config", "", "path to config file")
	shortConfigFlag := flag.String("c", "", "path to config file (shorthand)")
	flag.Parse()

	_ = configFlag
	_ = shortConfigFlag

	log.Printf("Build version: %s", fallback(buildVersion))
	log.Printf("Build date: %s", fallback(buildDate))
	log.Printf("Build commit: %s", fallback(buildCommit))

	finalAddr := *addr
	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		finalAddr = envAddr
	}

	storeInterval = *storeIntervalFlag
	if envInterval := os.Getenv("STORE_INTERVAL"); envInterval != "" {
		if val, err := strconv.Atoi(envInterval); err == nil {
			storeInterval = val
		}
	}

	fileStoragePath = *fileStoragePathFlag
	if envPath := os.Getenv("FILE_STORAGE_PATH"); envPath != "" {
		fileStoragePath = envPath
	}

	finalRestore := *restore
	if envRestore := os.Getenv("RESTORE"); envRestore != "" {
		if val, err := strconv.ParseBool(envRestore); err == nil {
			finalRestore = val
		}
	}

	finalDSN := *dsn
	if envDSN := os.Getenv("DATABASE_DSN"); envDSN != "" {
		finalDSN = envDSN
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

	finalAuditFile := *auditFileFlag
	if envAuditFile := os.Getenv("AUDIT_FILE"); envAuditFile != "" {
		finalAuditFile = envAuditFile
	}

	finalAuditURL := *auditURLFlag
	if envAuditURL := os.Getenv("AUDIT_URL"); envAuditURL != "" {
		finalAuditURL = envAuditURL
	}

	finalCryptoKey := *cryptoKeyFlag
	if envCrypto := os.Getenv("CRYPTO_KEY"); envCrypto != "" {
		finalCryptoKey = envCrypto
	}

	finalTrustedSubnet := *trustedSubnetFlag
	if envSubnet := os.Getenv("TRUSTED_SUBNET"); envSubnet != "" {
		finalTrustedSubnet = envSubnet
	}

	var trustedSubnet *net.IPNet
	if finalTrustedSubnet != "" {
		_, n, err := net.ParseCIDR(finalTrustedSubnet)
		if err != nil {
			log.Fatalf("Failed to parse trusted subnet: %v", err)
		}
		trustedSubnet = n
	}

	finalGRPCAddr := *grpcAddrFlag
	if envGRPC := os.Getenv("GRPC_ADDRESS"); envGRPC != "" {
		finalGRPCAddr = envGRPC
	}

	var privateKey *rsa.PrivateKey
	if finalCryptoKey != "" {
		var err error
		privateKey, err = loadPrivateKey(finalCryptoKey)
		if err != nil {
			log.Fatalf("Failed to load private key: %v", err)
		}
	}

	if finalDSN != "" {
		var err error
		db, err = sql.Open("postgres", finalDSN)
		if err != nil {
			log.Fatal("Failed to connect to database:", err)
		}
		defer db.Close()

		if err := runMigrations(db); err != nil && err != migrate.ErrNoChange {
			log.Printf("Failed to run migrations: %v", err)
		}

		storage, err = repository.NewPostgresStorage(db)
		if err != nil {
			log.Fatal("Failed to create postgres storage:", err)
		}
		useFileStorage = false
	} else if fileStoragePath != "" {
		storage = repository.NewMemStorage()
		useFileStorage = true
		if finalRestore {
			if err := loadFromFile(); err != nil {
				log.Printf("Failed to load from file: %v", err)
			}
		}
		defer func() {
			if err := saveToFile(); err != nil {
				log.Printf("Failed to save to file: %v", err)
			}
		}()
	} else {
		storage = repository.NewMemStorage()
		useFileStorage = false
	}

	if useFileStorage && storeInterval > 0 {
		go func() {
			ticker := time.NewTicker(time.Duration(storeInterval) * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				if err := saveToFile(); err != nil {
					log.Printf("Failed to save to file: %v", err)
				}
			}
		}()
	}

	h := handler.New(storage)

	var auditor audit.Notifier
	publisher := audit.NewPublisher()
	if finalAuditFile != "" {
		publisher.Register(audit.NewFileListener(finalAuditFile))
	}
	if finalAuditURL != "" {
		publisher.Register(audit.NewHTTPListener(finalAuditURL))
	}
	if publisher.HasListeners() {
		auditor = publisher
	}

	if auditor != nil {
		h.SetAuditor(auditor)
	}

	handler.SecretKey = finalKey

	if useFileStorage && storeInterval == 0 {
		handler.SyncSaveFunc = func() {
			if err := saveToFile(); err != nil {
				log.Printf("Failed to sync save: %v", err)
			}
		}
	}

	r := chi.NewRouter()

	r.Use(middleware.WithLogging)
	r.Use(middleware.WithTrustedSubnet(trustedSubnet))
	r.Use(middleware.WithDecrypt(privateKey))
	r.Use(middleware.WithGzipDecompress)
	r.Use(middleware.WithGzip)

	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		if db == nil {
			http.Error(w, "Database not configured", http.StatusInternalServerError)
			return
		}
		if err := db.Ping(); err != nil {
			http.Error(w, "Database connection failed", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	r.Post("/update/{type}/{name}/{value}", h.UpdateMetric)
	r.Post("/update", h.UpdateMetricJSON)
	r.Post("/update/", h.UpdateMetricJSON)
	r.Get("/value/{type}/{name}", h.GetMetric)
	r.Post("/value", h.GetMetricJSON)
	r.Post("/value/", h.GetMetricJSON)
	r.Get("/", h.ListMetrics)
	r.Post("/updates/", h.UpdateMetricsBatch)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	defer stop()

	var grpcSrv *grpc.Server
	var grpcLis net.Listener
	if finalGRPCAddr != "" {
		lis, err := net.Listen("tcp", finalGRPCAddr)
		if err != nil {
			log.Fatal("Failed to start gRPC listener:", err)
		}
		grpcLis = lis

		grpcSrv = grpc.NewServer(grpc.UnaryInterceptor(grpcserver.TrustedSubnetInterceptor(trustedSubnet)))
		pb.RegisterMetricsServer(grpcSrv, &grpcserver.Server{Storage: storage})

		go func() {
			if err := grpcSrv.Serve(grpcLis); err != nil {
				log.Printf("gRPC server error: %v", err)
			}
		}()
		log.Printf("Starting gRPC server on %s", finalGRPCAddr)
	}

	srv := &http.Server{
		Addr:    finalAddr,
		Handler: r,
	}

	done := make(chan struct{})
	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)

		if grpcSrv != nil {
			grpcSrv.GracefulStop()
		}
		if grpcLis != nil {
			_ = grpcLis.Close()
		}

		if useFileStorage {
			if err := saveToFile(); err != nil {
				log.Printf("Failed to save during shutdown: %v", err)
			}
		}

		close(done)
	}()

	log.Printf("Starting server on %s", finalAddr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal("Server fail:", err)
	}
	<-done
}

func fallback(value string) string {
	if value == "" {
		return "N/A"
	}
	return value
}

type serverFileConfig struct {
	Address       string `json:"address"`
	Restore       *bool  `json:"restore"`
	StoreInterval string `json:"store_interval"`
	StoreFile     string `json:"store_file"`
	DatabaseDSN   string `json:"database_dsn"`
	CryptoKey     string `json:"crypto_key"`
	TrustedSubnet string `json:"trusted_subnet"`
}

func loadServerConfigFile() *serverFileConfig {
	path := getServerConfigPathFromArgs()
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

	var cfg serverFileConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil
	}
	return &cfg
}

func getServerConfigPathFromArgs() string {
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

func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("invalid private key data")
	}

	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	rsaKey, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA private key")
	}
	return rsaKey, nil
}
