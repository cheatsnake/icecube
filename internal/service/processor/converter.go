package processor

import "github.com/cheatsnake/icm/internal/domain/image"

type Converter interface {
	Convert(imagePath, resultPath string) error
}

func needToConvert(from, to image.Format) bool {
	if from == image.FormatPNG && to == image.FormatJPEG {
		return true
	}
	if from == image.FormatWEBP && to == image.FormatJPEG {
		return true
	}

	// ? -> webp (no conversion needed, it included to compressor job)

	return false
}
