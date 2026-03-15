package image

import "testing"

func TestValidateFormat(t *testing.T) {
	tests := []struct {
		name    string
		format  Format
		wantErr bool
	}{
		{"valid jpeg", FormatJPEG, false},
		{"valid png", FormatPNG, false},
		{"valid webp", FormatWEBP, false},
		{"invalid empty", Format(""), true},
		{"invalid gif", Format("gif"), true},
		{"invalid bmp", Format("bmp"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFormat(tt.format)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFormat(%q) error = %v, wantErr %v", tt.format, err, tt.wantErr)
			}
		})
	}
}

func TestSupportedFormats(t *testing.T) {
	formats := SupportedFormats()
	if len(formats) != 3 {
		t.Errorf("SupportedFormats() = %d, want 3", len(formats))
	}
}
