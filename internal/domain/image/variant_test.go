package image

import (
	"testing"
)

func TestNewVariant(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		originalName string
		format      Format
		width       int
		height      int
		byteSize    int64
		wantErr     bool
	}{
		{"valid", "123", "test.png", FormatPNG, 100, 200, 5000, false},
		{"valid jpeg", "124", "photo.jpg", FormatJPEG, 800, 600, 100000, false},
		{"valid webp", "125", "image.webp", FormatWEBP, 1920, 1080, 50000, false},
		{"invalid format", "126", "test.gif", Format("gif"), 100, 200, 5000, true},
		{"zero width", "127", "test.png", FormatPNG, 0, 200, 5000, true},
		{"negative width", "128", "test.png", FormatPNG, -1, 200, 5000, true},
		{"zero height", "129", "test.png", FormatPNG, 100, 0, 5000, true},
		{"negative height", "130", "test.png", FormatPNG, 100, -1, 5000, true},
		{"zero byteSize", "131", "test.png", FormatPNG, 100, 200, 0, true},
		{"negative byteSize", "132", "test.png", FormatPNG, 100, 200, -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := NewVariant(tt.id, tt.originalName, tt.format, tt.width, tt.height, tt.byteSize)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewVariant() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && v == nil {
				t.Error("NewVariant() returned nil without error")
			}
		})
	}
}
