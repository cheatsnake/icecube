package imagestore

import (
	"regexp"
	"strings"

	"github.com/cheatsnake/icecube/internal/pkg/fs"
)

const maxFilenameLength = 128

var regexFilenameSanitizer = regexp.MustCompile(`[^a-zA-Z0-9._-]`)
var regexMultiUnderscore = regexp.MustCompile(`_+`)

func sanitizeFilename(name string) string {
	name = fs.BaseNameWithoutExtension(name)
	name = regexFilenameSanitizer.ReplaceAllString(name, "_")
	name = regexMultiUnderscore.ReplaceAllString(name, "_")
	name = strings.Trim(name, "_.")

	if name == "" {
		return "unknown"
	}

	if len(name) > maxFilenameLength {
		name = name[:maxFilenameLength]
	}

	return name
}
