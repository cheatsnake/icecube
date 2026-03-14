package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/cheatsnake/icm/internal/service/imagestore"
	"github.com/cheatsnake/icm/internal/service/jobstore"
	"github.com/cheatsnake/icm/internal/service/processor"
	"github.com/cheatsnake/icm/internal/transport/http"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	jobStore := jobstore.NewJobStoreMemory()
	imageBlobStore := imagestore.NewBlobStoreMemory()
	imageMetadataStore := imagestore.NewMetadataStoreMemory()
	imageStore := imagestore.NewStore(imageBlobStore, imageMetadataStore)
	processorService, _ := processor.NewService()

	worker := processor.NewWorker(processorService, jobStore, imageStore, logger.With("module", "processor"))
	ticker := time.NewTicker(1 * time.Second)
	go func() {
		for range ticker.C {
			worker.Run()
		}
	}()

	server := http.NewServer(imageStore, jobStore, logger.With("module", "http"))
	server.Run(3000)
}
