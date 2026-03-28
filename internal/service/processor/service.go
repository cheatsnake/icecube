package processor

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"
	"strings"

	"github.com/cheatsnake/icecube/internal/domain/image"
	"github.com/cheatsnake/icecube/internal/domain/processing"
	"github.com/cheatsnake/icecube/internal/pkg/errs"
	"github.com/cheatsnake/icecube/internal/pkg/fs"
)

type Service struct {
	logger     *slog.Logger
	resizer    Resizer
	converter  Converter
	compressor Compressor
}

func NewService(logger *slog.Logger) (*Service, error) {
	imageMagick, err := newImageMagick(logger)
	if err != nil {
		return nil, err
	}
	compressor, err := newCompressorCombined(logger)
	if err != nil {
		return nil, err
	}

	return &Service{
		logger:     logger,
		resizer:    imageMagick,
		converter:  imageMagick,
		compressor: compressor,
	}, nil
}

func (s *Service) Process(imagePath string, options *processing.Options) (string, error) {
	if options == nil {
		return "", errors.Join(errs.ErrInvalidInput, errors.New("options required"))
	}

	s.logger.Debug("Processing image", "input", imagePath, "format", options.Format, "quality", options.Quality, "maxDimension", options.MaxDimension)

	meta, err := fs.GetImageMetadata(imagePath)
	if err != nil {
		return "", err
	}

	originalFormat := image.Format(meta.Format)
	if !processing.CanConvert(originalFormat, options.Format) {
		return "", processing.ErrConversionNotSupported
	}

	uniqueTag := strings.ToLower(rand.Text())[20:]
	outputDir := path.Dir(imagePath)
	name := fs.BaseNameWithoutExtension(imagePath)
	// add unique tag and change extension to the desired format
	outputName := fmt.Sprintf("%s_%s.%s", name, uniqueTag, options.Format)
	resizedImage := imagePath

	if options.MaxDimension > 0 {
		resizedImage = path.Join(outputDir, (fmt.Sprintf("resized_%s", outputName)))
		width, height := resizeDimensions(options.MaxDimension, meta.Width, meta.Height)
		s.logger.Debug("Resizing image", "original", fmt.Sprintf("%dx%d", meta.Width, meta.Height), "target", fmt.Sprintf("%dx%d", width, height))
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
	} else {
		s.logger.Debug("Resize skipped", "maxDimension", options.MaxDimension)
	}

	convertedImage := resizedImage
	if needToConvert(originalFormat, options.Format) {
		s.logger.Debug("Converting image format", "from", originalFormat, "to", options.Format)
		convertedImage = path.Join(outputDir, (fmt.Sprintf("converted_%s", outputName)))
		err = s.converter.Convert(resizedImage, convertedImage)
		if err != nil {
			return "", err
		}
		defer os.Remove(convertedImage)
	} else {
		s.logger.Debug("Format conversion skipped", "originalFormat", originalFormat, "targetFormat", options.Format)
	}

	compressedImage := path.Join(outputDir, (fmt.Sprintf("compressed_%s", outputName)))
	compressorParams := CompressorParams{
		ImagePath:    convertedImage,
		ImageFormat:  options.Format,
		ResultPath:   compressedImage,
		Quality:      options.Quality,
		KeepMetadata: options.KeepMetadata,
		Extra:        nil,
	}
	err = s.compressor.Compress(compressorParams)
	if err != nil {
		return "", err
	}

	s.logger.Debug("Processing completed", "output", compressedImage, "format", options.Format)
	return compressedImage, nil
}
