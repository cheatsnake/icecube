package processor

import (
	"errors"

	"github.com/cheatsnake/icecube/internal/domain/errs"
	"github.com/cheatsnake/icecube/internal/domain/image"
)

type CompressorParams struct {
	ImagePath        string
	ImageFormat      image.Format
	ResultPath       string
	CompressionRatio int
	KeepMetadata     bool
	Extra            map[string]any
}

type Compressor interface {
	Compress(params CompressorParams) error
}

type compressorCombined struct {
	jpegoptim Compressor
	oxipng    Compressor
	pngquant  Compressor
	libwebp   Compressor
}

func newCompressorCombined() (*compressorCombined, error) {
	jpegoptim, err := newJpegoptim()
	if err != nil {
		return nil, err
	}
	oxipng, err := newOxipng()
	if err != nil {
		return nil, err
	}
	pngquant, err := newPngquant()
	if err != nil {
		return nil, err
	}
	libwebp, err := newLibwebp()
	if err != nil {
		return nil, err
	}
	return &compressorCombined{
		jpegoptim: jpegoptim,
		oxipng:    oxipng,
		pngquant:  pngquant,
		libwebp:   libwebp,
	}, nil
}

func (c *compressorCombined) Compress(params CompressorParams) error {
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
