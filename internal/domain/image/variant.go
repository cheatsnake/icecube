package image

import (
	"errors"
)

// Variant represents an uploaded image
type Variant struct {
	ID           string `json:"id"`           // Unique identifier for the image variant
	OriginalName string `json:"originalName"` // Original name of the image variant
	Format       Format `json:"format"`       // Format of the image variant
	Width        int    `json:"width"`        // Width of the image variant in pixels
	Height       int    `json:"height"`       // Height of the image variant in pixels
	ByteSize     int64  `json:"byteSize"`     // Size of the image variant in bytes
}

type VariantStorage interface {
	Create(variant *Variant) error
	Get(id string) (*Variant, error)
	GetMany(ids []string) ([]*Variant, error)
	Delete(id string) error
	DeleteMany(ids []string) error
}

var (
	ErrBadWidth    = errors.New("width must be positive")
	ErrBadHeight   = errors.New("height must be positive")
	ErrBadByteSize = errors.New("byte size must be positive")
)

func NewVariant(id, originalName string, format Format, width, height int, byteSize int64) (*Variant, error) {
	if err := ValidateFormat(format); err != nil {
		return nil, err
	}
	if width <= 0 {
		return nil, ErrBadWidth
	}
	if height <= 0 {
		return nil, ErrBadHeight
	}
	if byteSize <= 0 {
		return nil, ErrBadByteSize
	}

	return &Variant{
		ID:           id,
		OriginalName: originalName,
		Format:       format,
		Width:        width,
		Height:       height,
		ByteSize:     byteSize,
	}, nil
}
