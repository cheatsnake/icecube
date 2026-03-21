package imagestore

import (
	"context"
	"io"
	"log/slog"

	"github.com/cheatsnake/icecube/internal/domain/image"
)

type BlobStore interface {
	UploadImage(ctx context.Context, r io.Reader, name string, size int64) (*image.Variant, error)
	DownloadImage(ctx context.Context, id string) (io.ReadCloser, error)
	DeleteImage(ctx context.Context, id string) error
}

type MetadataStore interface {
	AddMetadata(ctx context.Context, metadata *image.Variant) error
	GetMetadataByID(ctx context.Context, id string) (*image.Variant, error)
	GetMetadataByIDs(ctx context.Context, ids []string) ([]*image.Variant, error)
	DeleteMetadataByID(ctx context.Context, id string) error
}

type Store struct {
	logger   *slog.Logger
	metadata MetadataStore
	blob     BlobStore
}

func NewStore(blobStore BlobStore, metadataStore MetadataStore, logger *slog.Logger) *Store {
	return &Store{logger: logger, blob: blobStore, metadata: metadataStore}
}

func (s *Store) GetMetadataByID(ctx context.Context, id string) (*image.Variant, error) {
	return s.metadata.GetMetadataByID(ctx, id)
}

func (s *Store) GetMetadataByIDs(ctx context.Context, ids []string) ([]*image.Variant, error) {
	return s.metadata.GetMetadataByIDs(ctx, ids)
}

func (s *Store) UploadImage(ctx context.Context, r io.Reader, name string, size int64) (*image.Variant, error) {
	metadata, err := s.blob.UploadImage(ctx, r, name, size)
	if err != nil {
		s.logger.Error("Blob upload failed", "error", err)
		return nil, err
	}
	err = s.metadata.AddMetadata(ctx, metadata)
	if err != nil {
		s.logger.Warn("Metadata add failed after blob upload", "blobID", metadata.ID, "error", err)
		return nil, err
	}
	s.logger.Info("Image uploaded", "id", metadata.ID, "name", name, "size", size)
	return metadata, nil
}

func (s *Store) DownloadImage(ctx context.Context, id string) (io.ReadCloser, error) {
	s.logger.Debug("Downloading image", "id", id)
	return s.blob.DownloadImage(ctx, id)
}

func (s *Store) DeleteImage(ctx context.Context, id string) error {
	if err := s.blob.DeleteImage(ctx, id); err != nil {
		s.logger.Error("Blob delete failed", "id", id, "error", err)
		return err
	}
	if err := s.metadata.DeleteMetadataByID(ctx, id); err != nil {
		s.logger.Warn("Metadata delete failed after blob deletion", "id", id, "error", err)
		return err
	}
	s.logger.Debug("Image deleted", "id", id)
	return nil
}
