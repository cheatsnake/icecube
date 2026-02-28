package imagestore

import (
	"context"
	"io"

	"github.com/cheatsnake/icm/internal/domain/image"
	domainimage "github.com/cheatsnake/icm/internal/domain/image"
	"github.com/cheatsnake/icm/internal/pkg/uuid"
)

type BlobStore interface {
	UploadImage(ctx context.Context, id string, r io.Reader) (*domainimage.Variant, error)
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
	metadata MetadataStore
	blob     BlobStore
}

func NewStore(blobStore BlobStore, metadataStore MetadataStore) *Store {
	return &Store{blob: blobStore, metadata: metadataStore}
}

func (s *Store) GetMetadataByID(ctx context.Context, id string) (*image.Variant, error) {
	return s.metadata.GetMetadataByID(ctx, id)
}

func (s *Store) GetMetadataByIDs(ctx context.Context, ids []string) ([]*image.Variant, error) {
	return s.metadata.GetMetadataByIDs(ctx, ids)
}

func (s *Store) UploadImage(ctx context.Context, r io.Reader) (*image.Variant, error) {
	id := uuid.V7()
	metadata, err := s.blob.UploadImage(ctx, id, r)
	if err != nil {
		return nil, err
	}
	err = s.metadata.AddMetadata(ctx, metadata)
	if err != nil {
		return nil, err
	}
	return metadata, nil
}

func (s *Store) DownloadImage(ctx context.Context, id string) (io.ReadCloser, error) {
	return s.blob.DownloadImage(ctx, id)
}

func (s *Store) DeleteImage(ctx context.Context, id string) error {
	if err := s.blob.DeleteImage(ctx, id); err != nil {
		return err
	}
	return s.metadata.DeleteMetadataByID(ctx, id)
}
