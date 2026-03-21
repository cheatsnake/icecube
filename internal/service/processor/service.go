package processor

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/cheatsnake/icecube/internal/domain/errs"
	"github.com/cheatsnake/icecube/internal/domain/image"
	"github.com/cheatsnake/icecube/internal/domain/processing"
	"github.com/cheatsnake/icecube/internal/pkg/fs"
)

type Service struct {
	resizer    Resizer
	converter  Converter
	compressor Compressor
}

func NewService() (*Service, error) {
	imageMagick, err := newImageMagick()
	if err != nil {
		return nil, err
	}
	compressor, err := newCompressorCombined()
	if err != nil {
		return nil, err
	}

	return &Service{
		resizer:    imageMagick,
		converter:  imageMagick,
		compressor: compressor,
	}, nil
}

func (s *Service) Process(imagePath string, options *processing.Options) (string, error) {
	if options == nil {
		return "", errors.Join(errs.ErrInvalidInput, errors.New("options required"))
	}

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

	return compressedImage, nil
}
