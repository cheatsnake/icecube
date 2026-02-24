package processor

import "math"

type ResizerParams struct {
	ImagePath  string
	ResultPath string
	Width      int
	Height     int
}

type Resizer interface {
	Resize(params ResizerParams) error
}

func resizeDimensions(maxDimension, originalWidth, originalHeight int) (newWidth, newHeight int) {
	if originalWidth <= maxDimension && originalHeight <= maxDimension {
		return originalWidth, originalHeight
	}

	if originalWidth > originalHeight {
		ratio := float64(maxDimension) / float64(originalWidth)
		newWidth = maxDimension
		newHeight = int(math.Round(float64(originalHeight) * ratio))
	} else {
		ratio := float64(maxDimension) / float64(originalHeight)
		newHeight = maxDimension
		newWidth = int(math.Round(float64(originalWidth) * ratio))
	}

	return newWidth, newHeight
}
