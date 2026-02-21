package image_processing

import (
	"path/filepath"
	"strings"

	"github.com/cheatsnake/icm/internal/domain/image"
)

type compressorCombined struct {
	jpegoptim Compressor
	oxipng    Compressor
	pngquant  Compressor
	libwebp   Compressor
}

func newCompressorCombined() (*compressorCombined, error) {
	jpegoptim, err := newCompressorJpegoptim()
	if err != nil {
		return nil, err
	}
	oxipng, err := newCompressorOxipng()
	if err != nil {
		return nil, err
	}
	pngquant, err := newCompressorPngquant()
	if err != nil {
		return nil, err
	}
	libwebp, err := newCompressorLibwebp()
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
	targetFormat := image.Format(strings.TrimPrefix(filepath.Ext(params.ResultPath), "."))

	switch targetFormat {
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
		return image.ErrBadFormat
	}
}
