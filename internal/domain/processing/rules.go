package processing

import "github.com/cheatsnake/icm/internal/domain/image"

// CanConvert checks if the given formats can be converted
func CanConvert(from, to image.Format) bool {
	if from == to {
		return true
	}

	switch from {
	case image.FormatPNG:
		return to == image.FormatWEBP || to == image.FormatJPEG
	case image.FormatWEBP:
		return to == image.FormatWEBP || to == image.FormatJPEG
	case image.FormatJPEG:
		return to == image.FormatWEBP
	default:
		return false
	}
}
