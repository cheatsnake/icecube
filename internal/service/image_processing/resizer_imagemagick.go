package image_processing

import (
	"fmt"
	"os/exec"
	"strings"
)

type resizerImageMagick struct{}

func newResizerImageMagick() (*resizerImageMagick, error) {
	cli := &resizerImageMagick{}
	_, err := cli.Version()

	return cli, err
}

func (rim *resizerImageMagick) Resize(originalPath, resultPath string, width, height int) error {
	sizeParam := fmt.Sprintf("%dx%d!", width, height)
	cmd := exec.Command("magick", "convert", originalPath, "-resize", sizeParam, resultPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("imagemagick resize error: %w", err)
	}
	return nil

}

func (rim *resizerImageMagick) Version() (string, error) {
	out, err := exec.Command("magick", "-version").Output()
	if err != nil {
		return "", fmt.Errorf("imagemagick not found: %v", err)
	}

	lines := strings.Split(string(out), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0]), nil
	}

	return "unknown version", nil
}
