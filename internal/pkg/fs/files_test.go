package fs

import "testing"

func TestBaseNameWithoutExtension(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		want     string
	}{
		{"simple file", "photo.jpg", "photo"},
		{"file with multiple dots", "image.test.png", "image.test"},
		{"no extension", "document", "document"},
		{"only extension", ".gitignore", ""},
		{"path with directory", "/home/user/images/photo.jpg", "photo"},
		{"hidden file", ".env", ""},
		{"trailing slash", "folder/", "folder"},
		{"nested path", "/path/to/file.name.txt", "file.name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BaseNameWithoutExtension(tt.filePath)
			if got != tt.want {
				t.Errorf("BaseNameWithoutExtension(%q) = %q, want %q", tt.filePath, got, tt.want)
			}
		})
	}
}
