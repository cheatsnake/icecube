package processing

import (
	"testing"

	"github.com/cheatsnake/icecube/internal/domain/image"
)

func TestNewOptions(t *testing.T) {
	tests := []struct {
		name             string
		format           image.Format
		maxDimension     int
		compressionRatio int
		keepMetadata     bool
		extra            map[string]string
		wantErr          bool
	}{
		{"valid", image.FormatPNG, 100, 80, false, nil, false},
		{"valid with extra", image.FormatJPEG, 800, 90, true, map[string]string{"key": "value"}, false},
		{"valid zero maxDimension", image.FormatPNG, 0, 80, false, nil, false},
		{"valid min compression", image.FormatWEBP, 100, 1, false, nil, false},
		{"valid max compression", image.FormatJPEG, 100, 100, false, nil, false},
		{"invalid format", image.Format("gif"), 100, 80, false, nil, true},
		{"negative maxDimension", image.FormatPNG, -1, 80, false, nil, true},
		{"compression too low", image.FormatPNG, 100, 0, false, nil, true},
		{"compression too high", image.FormatPNG, 100, 101, false, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := NewOptions(tt.format, tt.maxDimension, tt.compressionRatio, tt.keepMetadata, tt.extra)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewOptions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && opts == nil {
				t.Error("NewOptions() returned nil without error")
			}
		})
	}
}

func TestNewOptions_SetsFields(t *testing.T) {
	extra := map[string]string{"key": "value"}
	opts, err := NewOptions(image.FormatWEBP, 500, 75, true, extra)
	if err != nil {
		t.Fatalf("NewOptions() error = %v", err)
	}

	if opts.Format != image.FormatWEBP {
		t.Errorf("Format = %v, want %v", opts.Format, image.FormatWEBP)
	}
	if opts.MaxDimension != 500 {
		t.Errorf("MaxDimension = %v, want 500", opts.MaxDimension)
	}
	if opts.CompressionRatio != 75 {
		t.Errorf("CompressionRatio = %v, want 75", opts.CompressionRatio)
	}
	if !opts.KeepMetadata {
		t.Error("KeepMetadata should be true")
	}
	if opts.Extra["key"] != "value" {
		t.Errorf("Extra[key] = %v, want 'value'", opts.Extra["key"])
	}
}
