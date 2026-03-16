package processing

import (
	"errors"

	"github.com/cheatsnake/icecube/internal/domain/errs"
	"github.com/cheatsnake/icecube/internal/domain/image"
)

var (
	ErrConversionNotSupported = errors.Join(errs.ErrInvalidInput, errors.New("conversion from target format to source format is not supported"))
)

// CanConvert checks if the given formats can be converted
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
