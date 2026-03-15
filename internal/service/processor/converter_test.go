package processor

import (
	"testing"

	"github.com/cheatsnake/icecube/internal/domain/image"
)

func TestNeedToConvert(t *testing.T) {
	tests := []struct {
		name     string
		from     image.Format
		to       image.Format
		expected bool
	}{
		// Same format - no conversion needed
		{"jpeg -> jpeg", image.FormatJPEG, image.FormatJPEG, false},
		{"png -> png", image.FormatPNG, image.FormatPNG, false},
		{"webp -> webp", image.FormatWEBP, image.FormatWEBP, false},

		// PNG to JPEG - conversion needed
		{"png -> jpeg", image.FormatPNG, image.FormatJPEG, true},
		{"png -> webp", image.FormatPNG, image.FormatWEBP, false},

		// WEBP to JPEG - conversion needed
		{"webp -> jpeg", image.FormatWEBP, image.FormatJPEG, true},
		{"webp -> png", image.FormatWEBP, image.FormatPNG, false},

		// JPEG to WEBP - no conversion (handled by compressor)
		{"jpeg -> webp", image.FormatJPEG, image.FormatWEBP, false},
		{"jpeg -> png", image.FormatJPEG, image.FormatPNG, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := needToConvert(tt.from, tt.to)
			if result != tt.expected {
				t.Errorf("needToConvert(%q -> %q) = %v, want %v", tt.from, tt.to, result, tt.expected)
			}
		})
	}
}
