package processor

import (
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
)

type imageMagick struct {
	logger *slog.Logger
}

func newImageMagick(logger *slog.Logger) (*imageMagick, error) {
	cli := &imageMagick{logger: logger}
	_, err := cli.Version()
	if err != nil {
		logger.Error("imagemagick not found", "error", err)
		return nil, err
	}

	return cli, nil
}

func (im *imageMagick) Convert(originalPath, resultPath string) error {
	cmd := exec.Command("magick", originalPath, "-strip", resultPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("imagemagick conversion failed: %w", err)
	}
	return nil
}

func (im *imageMagick) Resize(params ResizerParams) error {
	sizeParam := fmt.Sprintf("%dx%d!", params.Width, params.Height)
	cmd := exec.Command("magick", "convert", params.ImagePath, "-resize", sizeParam, params.ResultPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("imagemagick resize error: %w", err)
	}
	return nil
}

func (im *imageMagick) Version() (string, error) {
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
