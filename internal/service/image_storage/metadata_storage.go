package image_storage

import "github.com/cheatsnake/icm/internal/domain/image"

type MetadataStorage interface {
	Get(id string) (*image.Variant, error)
	GetMany(ids []string) ([]*image.Variant, error)
	Add(id string, metadata *image.Variant) error
	Delete(id string) error
	DeleteMany(ids []string) error
}
