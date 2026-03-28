package imagestore

import "testing"

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Normal files
		{"simple name", "photo.jpg", "photo"},
		{"with spaces", "my photo.png", "my_photo"},
		{"with dots", "photo.test.jpg", "photo.test"},
		{"with dash", "my-photo.webp", "my-photo"},
		{"with underscore", "my_photo.png", "my_photo"},

		// Special characters
		{"with hash", "photo#1.png", "photo_1"},
		{"with at", "user@name.jpg", "user_name"},
		{"with percent", "file%name.png", "file_name"},
		{"with spaces only", "   photo   ", "photo"},

		// Edge cases
		{"only special chars", "@#$%", "unknown"},
		{"only dots", "...", "unknown"},
		{"only underscores", "___", "unknown"},
		{"empty string", "", "unknown"},
		{"extension only", ".jpg", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeFilename_MaxLength(t *testing.T) {
	longName := "a" + string(make([]byte, 200)) // 201 characters
	result := sanitizeFilename(longName)
	if len(result) > 128 {
		t.Errorf("sanitizeFilename() length = %d, want max 128", len(result))
	}
}

func TestSanitizeFilename_PreservesValidChars(t *testing.T) {
	valid := "abc123ABC_-.jpg"
	result := sanitizeFilename(valid)
	if result != "abc123ABC_-" {
		t.Errorf("sanitizeFilename(%q) = %q, want 'abc123ABC_-'", valid, result)
	}
}
