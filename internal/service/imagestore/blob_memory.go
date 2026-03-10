package imagestore

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/cheatsnake/icm/internal/domain/image"
	"github.com/cheatsnake/icm/internal/pkg/fs"
	"github.com/cheatsnake/icm/internal/pkg/uuid"
)

// BlobStoreMemory implements BlobStore interface using in-memory storage (for tests/dev)
type BlobStoreMemory struct {
	mu    sync.RWMutex
	blobs map[string]*blobData
}

type blobData struct {
	data []byte
}

// NewBlobStoreMemory creates a new efficient in-memory blob store
func NewBlobStoreMemory() *BlobStoreMemory {
	return &BlobStoreMemory{
		blobs: make(map[string]*blobData),
	}
}

// UploadImage stores an image blob in memory and returns a Variant with metadata
func (s *BlobStoreMemory) UploadImage(ctx context.Context, r io.Reader, name string, size int64) (*image.Variant, error) {
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
func (s *BlobStoreMemory) DownloadImage(ctx context.Context, id string) (io.ReadCloser, error) {
	s.mu.RLock()
	blob, exists := s.blobs[id]
	s.mu.RUnlock()

	if !exists || blob == nil {
		return nil, fmt.Errorf("blob not found: %s", id)
	}

	return io.NopCloser(bytes.NewReader(blob.data)), nil
}

// DeleteImage removes an image blob from memory
func (s *BlobStoreMemory) DeleteImage(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	blob, exists := s.blobs[id]
	if !exists {
		return fmt.Errorf("blob not found: %s", id)
	}

	blob.data = nil // Help garbage collection by clearing the reference
	delete(s.blobs, id)
	return nil
}

// DeleteImages removes multiple image blobs from memory
func (s *BlobStoreMemory) DeleteImages(ctx context.Context, ids []string) error {
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
func (s *BlobStoreMemory) extractImageMetadata(data []byte, id, originalName string) (*image.Variant, error) {
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
