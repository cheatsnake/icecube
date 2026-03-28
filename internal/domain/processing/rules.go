package processing

import (
	"errors"

	"github.com/cheatsnake/icecube/internal/domain/image"
	"github.com/cheatsnake/icecube/internal/pkg/errs"
)

var (
	ErrConversionNotSupported = errors.Join(errs.ErrInvalidInput, errors.New("conversion from target format to source format is not supported"))
)

// CanConvert checks if conversion from one format to another is supported.
func CanConvert(from, to image.Format) bool {
	if from == to {
		return true
	}

	switch from {
	case image.FormatPNG:
		return to == image.FormatWEBP || to == image.FormatJPEG
	case image.FormatWEBP:
		return to == image.FormatJPEG
	case image.FormatJPEG:
		return to == image.FormatWEBP
	default:
		return false
	}
}
