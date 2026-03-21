package processing

import (
	"errors"
	"fmt"

	"github.com/cheatsnake/icecube/internal/domain/image"
)

type Options struct {
	Format       image.Format      `json:"format"`          // Desired format of the processing image
	MaxDimension int               `json:"maxDimension"`    // Desired size of the largest dimension (width or height)
	Quality      int               `json:"quality"`         // Desired quality level (1-100, higher = better quality, larger file)
	KeepMetadata bool              `json:"keepMetadata"`    // Whether to keep metadata from the original image
	Extra        map[string]string `json:"extra,omitempty"` // Additional options for processing (depends on the format)
}

const (
	maxQuality = 100
	minQuality = 1
)

var (
	ErrBadMaxDimension = errors.New("max dimension size cannot be negative")
	ErrBadQuality      = fmt.Errorf("quality must be between %d and %d", minQuality, maxQuality)
)

func NewOptions(format image.Format, maxDimension, quality int, keepMetadata bool, extra map[string]string) (*Options, error) {
	if err := image.ValidateFormat(format); err != nil {
		return nil, err
	}
	if maxDimension < 0 {
		return nil, ErrBadMaxDimension
	}
	if quality < minQuality || quality > maxQuality {
		return nil, ErrBadQuality
	}

	return &Options{
		Format:       format,
		Quality:      quality,
		MaxDimension: maxDimension,
		KeepMetadata: keepMetadata,
		Extra:        extra,
	}, nil
}
