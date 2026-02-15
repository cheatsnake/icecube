package image_processing

import "github.com/cheatsnake/icm/internal/domain/image"

type Converter interface {
	Convert(imagePath, resultPath string) error
}

func needToConvert(from, to image.Format) bool {
	if from == to {
		return false
	}

	if from == image.FormatPNG && to == image.FormatJPEG {
		return true
	}
	if from == image.FormatWEBP && to == image.FormatJPEG {
		return true
	}
	// x -> webp
	if to == image.FormatWEBP {
		return false // no conversion needed (it included to compressor job)
	}

	return false
}
