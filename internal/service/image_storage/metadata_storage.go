package image_storage

import (
	"context"

	"github.com/cheatsnake/icm/internal/domain/image"
)

type MetadataStorage interface {
	Get(ctx context.Context, id string) (*image.Variant, error)
	GetMany(ctx context.Context, ids []string) ([]*image.Variant, error)
	Add(ctx context.Context, id string, metadata *image.Variant) error
	Delete(ctx context.Context, id string) error
	DeleteMany(ctx context.Context, ids []string) error
}
