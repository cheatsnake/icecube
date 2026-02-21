package fs

import (
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"

	_ "golang.org/x/image/webp"
)

type ImageMetadata struct {
	Width    int
	Height   int
	Format   string
	ByteSize int64
}

func GetImageMetadata(imagePath string) (*ImageMetadata, error) {
	fileInfo, err := os.Stat(imagePath)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(imagePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config, format, err := image.DecodeConfig(file)
	if err != nil {
		return nil, err
	}

	return &ImageMetadata{
		Width:    config.Width,
		Height:   config.Height,
		Format:   format,
		ByteSize: fileInfo.Size(),
	}, nil
}
