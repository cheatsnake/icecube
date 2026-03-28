package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/cheatsnake/icecube/internal/infra/config"
	"github.com/cheatsnake/icecube/internal/service/processor"
	"github.com/cheatsnake/icecube/internal/store/imagestore"
	"github.com/cheatsnake/icecube/internal/transport/http"
)

func main() {
	configPath := flag.String("config", config.DefaultConfigPath, "Path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("Failed to load config", "error", err.Error())
		os.Exit(1)
	}

	parsedLevel := parseLogLevel(cfg.Server.LogLevel)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: parsedLevel}))

	logger.Info("Starting icecube server", "config", *configPath)

	stores, err := cfg.LoadStores(context.Background(), logger)
	if err != nil {
		logger.Error("Failed to initialize stores", "error", err.Error())
		os.Exit(1)
	}

	imageStore := imagestore.NewStore(stores.ImageBlobStore, stores.ImageMetadataStore, logger.With("module", "imagestore"))
	processorService, err := processor.NewService(logger.With("module", "processor"))
	if err != nil {
		logger.Error("Failed to create processor service", "error", err.Error())
		os.Exit(1)
	}

	workerPool := processor.NewWorkerPool(
		processorService,
		stores.JobStore,
		imageStore,
		stores.KafkaProducer,
		logger.With("module", "processor"),
		cfg.Server.MaxWorkers,
	)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-stop
		logger.Info("Shutting down server...")
		workerPool.Stop()
		if stores.KafkaProducer != nil {
			stores.KafkaProducer.Close()
		}
		os.Exit(0)
	}()

	go workerPool.Run()

	server := http.NewServer(imageStore, stores.JobStore, logger.With("module", "http"))
	server.Run(cfg.Server.Port)
}

func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelDebug
	}
}
