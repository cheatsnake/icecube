package image_processing

import (
	"fmt"
	"os/exec"
)

type compressorPngquant struct{}

func newCompressorPngquant() (*compressorPngquant, error) {
	cli := &compressorPngquant{}
	_, err := cli.Version()

	return cli, err
}

func (cp *compressorPngquant) Compress(imagePath, resultPath string, compressionRatio int, keepMetadata bool, extra map[string]any) error {
	quality := fmt.Sprintf("0-%d", compressionRatio)
	args := []string{
		"--quality", quality,
		"--force",
		"--output", resultPath,
		imagePath,
	}

	if !keepMetadata {
		args = append([]string{"--strip"}, args...)
	}

	for key, val := range extra {
		args = append([]string{fmt.Sprintf("--%s", key), fmt.Sprintf("%v", val)}, args...)
	}

	cmd := exec.Command("pngquant", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("pngquant error: %v, output: %s", err, string(output))
	}

	return nil
}

func (cp *compressorPngquant) Version() (string, error) {
	out, err := exec.Command("pngquant", "--version").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
