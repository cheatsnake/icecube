package fs

import (
	"path/filepath"
	"strings"
)

func BaseNameWithoutExtension(filePath string) string {
	filename := filepath.Base(filePath)
	return strings.TrimSuffix(filename, filepath.Ext(filename))
}
