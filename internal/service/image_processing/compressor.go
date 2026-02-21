package image_processing

type CompressorParams struct {
	ImagePath        string
	ResultPath       string
	CompressionRatio int
	KeepMetadata     bool
	Extra            map[string]any
}

type Compressor interface {
	Compress(params CompressorParams) error
}
