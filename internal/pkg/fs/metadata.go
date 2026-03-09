package fs

import (
	"bufio"
	"bytes"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
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

	if format == "jpg" {
		format = "jpeg"
	}

	return &ImageMetadata{
		Width:    config.Width,
		Height:   config.Height,
		Format:   format,
		ByteSize: fileInfo.Size(),
	}, nil
}

func GetImageMetadataFromReader(r io.Reader) (*ImageMetadata, io.Reader, error) {
	br := bufio.NewReader(r)

	header, err := br.Peek(32 * 1024) // 32 KB
	if err != nil && err != bufio.ErrBufferFull {
		return nil, nil, err
	}

	cfg, format, err := image.DecodeConfig(bytes.NewReader(header))
	if err != nil {
		return nil, nil, err
	}

	if format == "jpg" {
		format = "jpeg"
	}

	return &ImageMetadata{
		Width:    cfg.Width,
		Height:   cfg.Height,
		Format:   format,
		ByteSize: 0,
	}, br, nil
}
