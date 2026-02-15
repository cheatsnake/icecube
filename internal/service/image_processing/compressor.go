package image_processing

type Compressor interface {
	Compress(imagePath, resultPath string, compressionRatio int, keepMetadata bool, extra map[string]any) error
}
