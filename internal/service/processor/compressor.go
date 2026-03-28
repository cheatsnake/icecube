package processor

import (
	"errors"
	"log/slog"

	"github.com/cheatsnake/icecube/internal/domain/image"
	"github.com/cheatsnake/icecube/internal/pkg/errs"
)

type CompressorParams struct {
	ImagePath    string
	ImageFormat  image.Format
	ResultPath   string
	Quality      int
	KeepMetadata bool
	Extra        map[string]any
}

type Compressor interface {
	Compress(params CompressorParams) error
}

type compressorCombined struct {
	logger    *slog.Logger
	jpegoptim Compressor
	oxipng    Compressor
	pngquant  Compressor
	libwebp   Compressor
}

func newCompressorCombined(logger *slog.Logger) (*compressorCombined, error) {
	jpegoptim, err := newJpegoptim(logger)
	if err != nil {
		return nil, err
	}
	oxipng, err := newOxipng(logger)
	if err != nil {
		return nil, err
	}
	pngquant, err := newPngquant(logger)
	if err != nil {
		return nil, err
	}
	libwebp, err := newLibwebp(logger)
	if err != nil {
		return nil, err
	}
	return &compressorCombined{
		logger:    logger,
		jpegoptim: jpegoptim,
		oxipng:    oxipng,
		pngquant:  pngquant,
		libwebp:   libwebp,
	}, nil
}

func (c *compressorCombined) Compress(params CompressorParams) error {
	var compressorName string
	switch params.ImageFormat {
	case image.FormatJPEG:
		compressorName = "jpegoptim"
	case image.FormatPNG:
		lossless, ok := params.Extra["lossless"]
		if ok && lossless.(bool) {
			compressorName = "oxipng"
		} else {
			compressorName = "pngquant"
		}
	case image.FormatWEBP:
		compressorName = "libwebp"
	default:
		return errors.Join(errs.ErrInvalidInput, image.ErrBadFormat)
	}

	c.logger.Debug("Compressing image", "format", params.ImageFormat, "quality", params.Quality, "compressor", compressorName)

	switch params.ImageFormat {
	case image.FormatJPEG:
		return c.jpegoptim.Compress(params)
	case image.FormatPNG:
		lossless, ok := params.Extra["lossless"]
		if ok && lossless.(bool) {
			return c.oxipng.Compress(params)
		}
		return c.pngquant.Compress(params)
	case image.FormatWEBP:
		return c.libwebp.Compress(params)
	default:
		return errors.Join(errs.ErrInvalidInput, image.ErrBadFormat)
	}
}
