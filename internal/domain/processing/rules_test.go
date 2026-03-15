package processing

import (
	"testing"

	"github.com/cheatsnake/icecube/internal/domain/image"
)

func TestCanConvert(t *testing.T) {
	tests := []struct {
		name     string
		from     image.Format
		to       image.Format
		expected bool
	}{
		// Same format - always allowed
		{"jpeg -> jpeg", image.FormatJPEG, image.FormatJPEG, true},
		{"png -> png", image.FormatPNG, image.FormatPNG, true},
		{"webp -> webp", image.FormatWEBP, image.FormatWEBP, true},

		// From JPEG
		{"jpeg -> webp", image.FormatJPEG, image.FormatWEBP, true},
		{"jpeg -> png", image.FormatJPEG, image.FormatPNG, false},

		// From PNG
		{"png -> webp", image.FormatPNG, image.FormatWEBP, true},
		{"png -> jpeg", image.FormatPNG, image.FormatJPEG, true},

		// From WEBP
		{"webp -> jpeg", image.FormatWEBP, image.FormatJPEG, true},
		{"webp -> png", image.FormatWEBP, image.FormatPNG, false},

		// Invalid formats
		{"gif -> jpeg", image.Format("gif"), image.FormatJPEG, false},
		{"jpeg -> gif", image.FormatJPEG, image.Format("gif"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CanConvert(tt.from, tt.to)
			if result != tt.expected {
				t.Errorf("CanConvert(%q -> %q) = %v, want %v", tt.from, tt.to, result, tt.expected)
			}
		})
	}
}
