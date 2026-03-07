package http

import (
	"errors"
	"io"
	"net/http"
)

func (s *Server) handleHealthcheck(w http.ResponseWriter, r *http.Request) {
	jsonOK(w, "Service is healthy")
}

func (s *Server) handleUploadImage(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	if r.Method != http.MethodPost {
		jsonMethodNotAllowed(w)
		return
	}

	variant, err := s.imageStore.UploadImage(r.Context(), r.Body)
	if err != nil {
		s.logger.Warn("Image upload failed", "message", err.Error())
		jsonBadRequest(w, err.Error())
		return

	}

	w.WriteHeader(http.StatusCreated)
	jsonResponse(w, variant)
}

func (s *Server) handleDownloadImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonMethodNotAllowed(w)
		return
	}

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
