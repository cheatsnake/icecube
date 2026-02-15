package image_processing

import "math"

type Resizer interface {
	Resize(imagePath, resultPath string, width, height int) error
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
