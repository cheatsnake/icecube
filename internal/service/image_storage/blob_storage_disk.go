package image_storage

import (
	"context"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"

	domainimage "github.com/cheatsnake/icm/internal/domain/image"
	"github.com/cheatsnake/icm/internal/pkg/fs"
)

type blobStorageDisk struct {
	root string
}

func NewBlobStorageDisk(root string) BlobStorage {
	return &blobStorageDisk{root: root}
}

// Upload stores an image blob and returns a Variant with metadata
func (s *blobStorageDisk) Upload(ctx context.Context, id string, r io.Reader) (*domainimage.Variant, error) {
	// Create the directory for this file if it doesn't exist
	filePath := s.getFilePath(id)
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

	variant, err := s.extractImageMetadata(filePath)
	if err != nil {
		os.Remove(filePath)
		return nil, fmt.Errorf("failed to extract image metadata: %w", err)
	}

	return variant, nil
}

// Download retrieves an image blob
func (s *blobStorageDisk) Download(ctx context.Context, id string) (io.ReadCloser, error) {
	filePath := s.getFilePath(id)

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", id)
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, nil
}

// Delete removes an image blob
func (s *blobStorageDisk) Delete(ctx context.Context, id string) error {
	filePath := s.getFilePath(id)

	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", id)
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// DeleteMany removes multiple image blobs
func (s *blobStorageDisk) DeleteMany(ctx context.Context, ids []string) error {
	var errs []error

	for _, id := range ids {
		if err := s.Delete(ctx, id); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to delete some files: %v", errs)
	}

	return nil
}

func (s *blobStorageDisk) getFilePath(id string) string {
	// Use the last 2 characters of the ID as a directory to avoid too many files in one directory
	if len(id) >= 2 {
		return filepath.Join(s.root, id[len(id)-2:], id)
	}
	return filepath.Join(s.root, id)
}

func (s *blobStorageDisk) extractImageMetadata(filePath string) (*domainimage.Variant, error) {
	meta, err := fs.GetImageMetadata(filePath)
	if err != nil {
		return nil, err
	}

	imageFormat := domainimage.Format(meta.Format)
	if err := domainimage.ValidateFormat(imageFormat); err != nil {
		return nil, err
	}

	return domainimage.NewVariant(imageFormat, "", meta.Width, meta.Height, meta.ByteSize)
}
