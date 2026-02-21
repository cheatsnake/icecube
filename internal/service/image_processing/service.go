package image_processing

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/cheatsnake/icm/internal/domain/image"
	"github.com/cheatsnake/icm/internal/domain/processing"
	"github.com/cheatsnake/icm/internal/pkg/fs"
)

type Service struct {
	resizer    Resizer
	converter  Converter
	compressor Compressor
}

func NewService() (*Service, error) {
	resizer, err := newResizerImageMagick()
	if err != nil {
		return nil, err
	}
	converter, err := newConverterImageMagick()
	if err != nil {
		return nil, err
	}
	compressor, err := newCompressorCombined()
	if err != nil {
		return nil, err
	}

	return &Service{
		resizer:    resizer,
		converter:  converter,
		compressor: compressor,
	}, nil
}

func (s *Service) Process(imagePath string, options *processing.Options) (string, error) {
	if options == nil {
		return "", errors.New("options required")
	}

	meta, err := fs.GetImageMetadata(imagePath)
	if err != nil {
		return "", err
	}

	originalFormat := image.Format(strings.TrimPrefix(path.Ext(imagePath), "."))
	if !processing.CanConvert(originalFormat, options.Format) {
		return "", processing.ErrConversionNotSupported
	}

	outputDir := path.Dir(imagePath)
	baseName := path.Base(imagePath)
	outputName := strings.TrimSuffix(baseName, path.Ext(baseName)) + "." + string(options.Format)
	resizedImage := imagePath

	if options.MaxDimension > 0 {
		resizedImage = path.Join(outputDir, (fmt.Sprintf("resized_%s", baseName)))
		width, height := resizeDimensions(options.MaxDimension, meta.Width, meta.Height)
		resizerParams := ResizerParams{
			ImagePath:  imagePath,
			ResultPath: resizedImage,
			Width:      width,
			Height:     height,
		}

		err = s.resizer.Resize(resizerParams)
		if err != nil {
			return "", err
		}
		defer os.Remove(resizedImage)
	}

	convertedImage := resizedImage
	if needToConvert(originalFormat, options.Format) {
		convertedImage = path.Join(outputDir, (fmt.Sprintf("converted_%s", outputName)))
		err = s.converter.Convert(resizedImage, convertedImage)
		if err != nil {
			return "", err
		}
		defer os.Remove(convertedImage)
	}

	compressedImage := path.Join(outputDir, (fmt.Sprintf("compressed_%s", outputName)))
	compressorParams := CompressorParams{
		ImagePath:        convertedImage,
		ResultPath:       compressedImage,
		CompressionRatio: options.CompressionRatio,
		KeepMetadata:     options.KeepMetadata,
		Extra:            nil,
	}
	err = s.compressor.Compress(compressorParams)
	if err != nil {
		return "", err
	}

	return compressedImage, nil
}
