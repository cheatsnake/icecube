package imagestore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"

	"github.com/cheatsnake/icecube/internal/domain/image"
	"github.com/cheatsnake/icecube/internal/pkg/errs"
	"github.com/cheatsnake/icecube/internal/pkg/fs"
	"github.com/cheatsnake/icecube/internal/pkg/uuid"
)

type blobStoreMemory struct {
	logger *slog.Logger
	mu     sync.RWMutex
	blobs  map[string]*blobData
}

type blobData struct {
	data []byte
}

// newBlobStoreMemory creates a new efficient in-memory blob store
func newBlobStoreMemory(logger *slog.Logger) *blobStoreMemory {
	return &blobStoreMemory{
		logger: logger,
		blobs:  make(map[string]*blobData),
	}
}

// NewTestBlobStoreMemory creates a new in-memory blob store for testing
func NewTestBlobStoreMemory(logger *slog.Logger) *blobStoreMemory {
	return newBlobStoreMemory(logger)
}

// UploadImage stores an image blob in memory and returns a Variant with metadata
func (s *blobStoreMemory) UploadImage(ctx context.Context, r io.Reader, name string, size int64) (*image.Variant, error) {
	id := uuid.V7()

	// Read the entire blob into memory
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read blob data: %w", err)
	}

	meta, err := s.extractImageMetadata(data, id, name)
	if err != nil {
		return nil, fmt.Errorf("failed to extract image metadata: %w", err)
	}

	// Store the blob in memory
	s.mu.Lock()
	s.blobs[id] = &blobData{data: data}
	s.mu.Unlock()

	return meta, nil
}

// DownloadImage retrieves an image blob from memory
func (s *blobStoreMemory) DownloadImage(ctx context.Context, id string) (io.ReadCloser, error) {
	s.mu.RLock()
	blob, exists := s.blobs[id]
	s.mu.RUnlock()

	if !exists || blob == nil {
		return nil, errors.Join(errs.ErrNotFound, errors.New("blob not found: "+id))
	}

	return io.NopCloser(bytes.NewReader(blob.data)), nil
}

// DeleteImage removes an image blob from memory
func (s *blobStoreMemory) DeleteImage(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	blob, exists := s.blobs[id]
	if !exists {
		return errors.Join(errs.ErrNotFound, errors.New("blob not found: "+id))
	}

	blob.data = nil // Help garbage collection by clearing the reference
	delete(s.blobs, id)
	return nil
}

// DeleteImages removes multiple image blobs from memory
func (s *blobStoreMemory) DeleteImages(ctx context.Context, ids []string) error {
	var errs []error

	for _, id := range ids {
		if err := s.DeleteImage(ctx, id); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to delete some blobs: %v", errs)
	}

	return nil
}

// extractImageMetadata extracts metadata from blob data
func (s *blobStoreMemory) extractImageMetadata(data []byte, id, originalName string) (*image.Variant, error) {
	meta, _, err := fs.GetImageMetadataFromReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	imageFormat := image.Format(meta.Format)
	if err := image.ValidateFormat(imageFormat); err != nil {
		return nil, err
	}

	name := sanitizeFilename(originalName)
	meta.ByteSize = int64(len(data))

	return image.NewVariant(id, name, imageFormat, meta.Width, meta.Height, meta.ByteSize)
}
