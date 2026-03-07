package http

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/cheatsnake/icm/internal/domain/image"
	"github.com/cheatsnake/icm/internal/domain/jobs"
)

type ImageStore interface {
	UploadImage(ctx context.Context, r io.Reader) (*image.Variant, error)
	DownloadImage(ctx context.Context, id string) (io.ReadCloser, error)
	GetMetadataByID(ctx context.Context, id string) (*image.Variant, error)
	GetMetadataByIDs(ctx context.Context, ids []string) ([]*image.Variant, error)
}

type JobStore interface {
	CreateJob(ctx context.Context, job *jobs.Job) error
	GetJob(ctx context.Context, id string) (*jobs.Job, error)
}

type Worker interface {
	Run() error
}

type Server struct {
	router     *http.ServeMux
	imageStore ImageStore
	jobStore   JobStore
	worker     Worker
	logger     *slog.Logger
}

func NewServer(imageStore ImageStore, jobStore JobStore, worker Worker, logger *slog.Logger) *Server {
	return &Server{
		router:     http.NewServeMux(),
		imageStore: imageStore,
		jobStore:   jobStore,
		worker:     worker,
		logger:     logger,
	}
}

func (s *Server) Run(port int) error {
	portStr := strconv.Itoa(port)

	s.router.HandleFunc("GET "+apiPrefix+"/health", s.handleHealthcheck)
	s.router.HandleFunc("POST "+apiPrefix+"/image", s.handleUploadImage)
	s.router.HandleFunc("GET /image/{id}", s.handleDownloadImage)

	s.logger.Info("Server starts on http://localhost:" + portStr)
	err := http.ListenAndServe(":"+portStr, (s.router))
	if err != nil {
		s.logger.Error("Listen and serve failed", slog.String("info", err.Error()))
	}
	return nil
}
