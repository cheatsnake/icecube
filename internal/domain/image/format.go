package image

import "errors"

// Format represents an supported image format
type Format string

const (
	FormatJPEG Format = "jpeg"
	FormatPNG  Format = "png"
	FormatWEBP Format = "webp"
)

var (
	ErrBadFormat = errors.New("invalid or unsupported image format")
)

func ValidateFormat(f Format) error {
	switch f {
	case FormatJPEG, FormatPNG, FormatWEBP:
		return nil
	default:
		return ErrBadFormat
	}
}

func SupportedFormats() []Format {
	return []Format{FormatJPEG, FormatPNG, FormatWEBP}
}
