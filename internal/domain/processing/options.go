package processing

import (
	"errors"
	"fmt"

	"github.com/cheatsnake/icm/internal/domain/image"
)

type Options struct {
	Format           image.Format      `json:"format"`           // Desired format of the processing image
	MaxDimension     int               `json:"maxDimension"`     // Desired size of the largest dimension (width or height)
	CompressionRatio int               `json:"compressionRatio"` // Desired compression ratio (relates to image quality in lossy formats and compression strength in lossless formats)
	KeepMetadata     bool              `json:"keepMetadata"`     // Whether to keep metadata from the original image
	Extra            map[string]string `json:"extra,omitempty"`  // Additional options for processing (depends on the format)
}

const (
	maxCompressionRatio = 100
	minCompressionRatio = 1
)

var (
	ErrBadMaxDimension     = errors.New("max dimension size cannot be negative")
	ErrBadCompressionRatio = fmt.Errorf("compression ratio must be between %d and %d", minCompressionRatio, maxCompressionRatio)
)

func NewOptions(format image.Format, maxDimension, compressionRatio int, keepMetadata bool, extra map[string]string) (*Options, error) {
	if err := image.ValidateFormat(format); err != nil {
		return nil, err
	}
	if maxDimension < 0 {
		return nil, ErrBadMaxDimension
	}
	if compressionRatio < minCompressionRatio || compressionRatio > maxCompressionRatio {
		return nil, ErrBadCompressionRatio
	}

	return &Options{
		Format:           format,
		MaxDimension:     maxDimension,
		CompressionRatio: compressionRatio,
		KeepMetadata:     keepMetadata,
		Extra:            extra,
	}, nil
}
