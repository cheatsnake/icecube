package image_storage

import (
	"context"
	"io"

	"github.com/cheatsnake/icm/internal/domain/image"
)

type BlobStorage interface {
	Upload(ctx context.Context, id string, r io.Reader) (*image.Variant, error)
	Download(ctx context.Context, id string) (io.ReadCloser, error)
	Delete(ctx context.Context, id string) error
	DeleteMany(ctx context.Context, ids []string) error
}
