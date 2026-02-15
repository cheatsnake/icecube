package image_processing

import (
	"fmt"
	"os/exec"
	"strings"
)

type converterImageMagick struct{}

func newConverterImageMagick() (*converterImageMagick, error) {
	cli := &converterImageMagick{}
	_, err := cli.Version()

	return cli, err
}

func (cim *converterImageMagick) Convert(originalPath, resultPath string) error {
	cmd := exec.Command("magick", originalPath, "-strip", resultPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("imagemagick conversion failed: %w", err)
	}
	return nil
}

func (cim *converterImageMagick) Version() (string, error) {
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
