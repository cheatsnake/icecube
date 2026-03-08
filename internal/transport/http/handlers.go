package http

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/cheatsnake/icm/internal/domain/jobs"
	"github.com/cheatsnake/icm/internal/domain/processing"
)

func (s *Server) handleHealthcheck(w http.ResponseWriter, r *http.Request) {
	jsonOK(w, "Service is healthy")
}

func (s *Server) handleUploadImage(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	err := r.ParseMultipartForm(maxBodySize)
	if err != nil {
		jsonBadRequest(w, "invalid multipart form")
		return
	}

	files := r.MultipartForm.File["file"]
	if len(files) == 0 {
		jsonBadRequest(w, "no images provided")
		return
	}

	var results []any

	for _, fh := range files {
		file, err := fh.Open()

		if err != nil {
			jsonBadRequest(w, "failed to open file")
			return
		}

		buf := make([]byte, 512)
		n, err := file.Read(buf)
		if err != nil && err != io.EOF {
			file.Close()
			jsonBadRequest(w, "failed to read file")
			return
		}

		contentType := http.DetectContentType(buf[:n])
		if !strings.HasPrefix(contentType, "image/") {
			file.Close()
			jsonBadRequest(w, "only image files allowed")
			return
		}

		_, err = file.Seek(0, io.SeekStart)
		if err != nil {
			file.Close()
			jsonBadRequest(w, "failed to reset file reader")
			return
		}

		variant, err := s.imageStore.UploadImage(r.Context(), file, fh.Filename)
		file.Close()

		if err != nil {
			s.logger.Warn("Image upload failed", "message", err.Error())
			jsonBadRequest(w, err.Error())
			return
		}

		results = append(results, variant)
	}

	w.WriteHeader(http.StatusCreated)
	jsonResponse(w, results)
}

func (s *Server) handleDownloadImage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		jsonBadRequest(w, "Missing image ID")
		return
	}

	metadata, err := s.imageStore.GetMetadataByID(r.Context(), id)
	if err != nil {
		s.logger.Error("Failed to retrieve image metadata", "message", err.Error())
		jsonBadRequest(w, "Failed to retrieve image metadata")
		return
	}
	if metadata == nil {
		jsonNotFound(w, "Image not found")
		return
	}

	reader, err := s.imageStore.DownloadImage(r.Context(), id)
	if err != nil {
		s.logger.Error("Failed to download image", "message", err.Error())
		jsonBadRequest(w, "Failed to download image")
		return
	}
	defer reader.Close()

	// Sniff the content type from the first 512 bytes.
	buf := make([]byte, 512)
	n, err := io.ReadFull(reader, buf)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		jsonInternalError(w, "Failed to read image data")
		return
	}

	contentType := http.DetectContentType(buf[:n])
	w.Header().Set("Content-Type", contentType)

	if _, err := w.Write(buf[:n]); err != nil {
		return
	}
	if _, err := io.Copy(w, reader); err != nil {
		return
	}
}

func (s *Server) handleImageMetadata(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		jsonBadRequest(w, "Image ID not provided")
		return
	}

	metadata, err := s.imageStore.GetMetadataByID(r.Context(), id)
	if err != nil {
		jsonNotFound(w, "Image not found")
		return
	}

	jsonResponse(w, metadata)
}

func (s *Server) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	payload, err := jsonBodyParse[struct {
		OriginalID string               `json:"originalID"`
		Options    []processing.Options `json:"options"`
	}](r)
	if err != nil {
		jsonBadRequest(w, err.Error())
		return
	}

	job, err := jobs.NewJob(payload.OriginalID)
	if err != nil {
		jsonBadRequest(w, err.Error())
		return
	}
	for _, opt := range payload.Options {
		job.AddTask(&opt)
	}

	if err := s.jobStore.CreateJob(r.Context(), job); err != nil {
		jsonBadRequest(w, err.Error())
		return
	}

	jsonResponse(w, job)
}

func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		jsonBadRequest(w, "Missing job ID")
		return
	}

	job, err := s.jobStore.GetJob(r.Context(), id)
	if err != nil {
		jsonBadRequest(w, err.Error())
		return
	}

	jsonResponse(w, job)
}
