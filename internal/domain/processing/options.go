package processing

import (
	"errors"
	"fmt"

	"github.com/cheatsnake/icm/internal/domain/image"
)

type Options struct {
	Format           image.Format   // Desired format of the processing image
	MaxDimension     int            // Desired size of the largest dimension (width or height)
	CompressionRatio int            // Desired compression ratio (relates to image quality in lossy formats and compression strength in lossless formats)
	KeepMetadata     bool           // Whether to keep metadata from the original image
	Extra            map[string]any // Additional options for processing (depends on the format)
}

const (
	maxCompressionRatio = 100
	minCompressionRatio = 1
)

var (
	ErrBadMaxDimension     = errors.New("max dimension size cannot be negative")
	ErrBadCompressionRatio = fmt.Errorf("compression ratio must be between %d and %d", minCompressionRatio, maxCompressionRatio)
)

func NewOptions(format image.Format, maxDimension, compressionRatio int, keepMetadata bool, extra map[string]any) (*Options, error) {
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
