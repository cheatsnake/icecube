package imagestore

import (
	"context"
	"database/sql"
	"io"

	"github.com/cheatsnake/icm/internal/domain/image"
	"github.com/cheatsnake/icm/internal/pkg/uuid"
)

type Store struct {
	metadata *metadataStorePostgres
	blob     *blobStoreDisk
}

func NewStore(conn *sql.DB) (*Store, error) {
	disk := newBlobStoreDisk("static")
	postgres, err := newMetadataStorePostgres(conn)
	if err != nil {
		return nil, err
	}

	return &Store{blob: disk, metadata: postgres}, nil
}

func (s *Store) GetMetadata(ctx context.Context, id string) (*image.Variant, error) {
	return s.metadata.GetMetadataByID(ctx, id)
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
