package imagestore

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/cheatsnake/icm/internal/domain/image"
	"github.com/cheatsnake/icm/internal/pkg/fs"
	"github.com/cheatsnake/icm/internal/pkg/uuid"
)

type BlobStoreDisk struct {
	root string
}

func NewBlobStoreDisk(root string) *BlobStoreDisk {
	return &BlobStoreDisk{root: root}
}

// UploadImage stores an image blob and returns a Variant with metadata
func (s *BlobStoreDisk) UploadImage(ctx context.Context, r io.Reader, name string, size int64) (*image.Variant, error) {
	id := uuid.V7()
	// Create the directory for this file if it doesn't exist
	filePath := s.generateFilePathByID(id)
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Write the content
	_, err = io.Copy(file, r)
	if err != nil {
		os.Remove(filePath)
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	variant, err := s.extractImageMetadata(filePath, id, name)
	if err != nil {
		os.Remove(filePath)
		return nil, fmt.Errorf("failed to extract image metadata: %w", err)
	}

	return variant, nil
}

// DownloadImage retrieves an image blob
func (s *BlobStoreDisk) DownloadImage(ctx context.Context, id string) (io.ReadCloser, error) {
	filePath := s.generateFilePathByID(id)

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", filePath)
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, nil
}

// DeleteImage removes an image blob
func (s *BlobStoreDisk) DeleteImage(ctx context.Context, id string) error {
	filePath := s.generateFilePathByID(id)

	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", id)
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// DeleteImages removes multiple image blobs
func (s *BlobStoreDisk) DeleteImages(ctx context.Context, ids []string) error {
	var errs []error

	for _, id := range ids {
		if err := s.DeleteImage(ctx, id); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to delete some files: %v", errs)
	}

	return nil
}

func (s *BlobStoreDisk) generateFilePathByID(id string) string {
	// Use the last 2 characters of the ID as a directory to avoid too many files in one directory
	if len(id) >= 2 {
		return filepath.Join(s.root, id[len(id)-2:], id)
	}
	return filepath.Join(s.root, id)
}

func (s *BlobStoreDisk) extractImageMetadata(filePath, id, originalName string) (*image.Variant, error) {
	meta, err := fs.GetImageMetadata(filePath)
	if err != nil {
		return nil, err
	}

	imageFormat := image.Format(meta.Format)
	if err := image.ValidateFormat(imageFormat); err != nil {
		return nil, err
	}

	name := sanitizeFilename(originalName)

	return image.NewVariant(id, name, imageFormat, meta.Width, meta.Height, meta.ByteSize)
}
