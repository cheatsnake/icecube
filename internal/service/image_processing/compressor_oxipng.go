package image_processing

import (
	"fmt"
	"os/exec"
	"strings"
)

type compressorOxipng struct{}

func newCompressorOxipng() (*compressorOxipng, error) {
	cli := &compressorOxipng{}
	_, err := cli.Version()

	return cli, err
}

func (co *compressorOxipng) Compress(params CompressorParams) error {
	level := co.mapRatioToLevel(params.CompressionRatio)
	args := []string{fmt.Sprintf("-o%d", level)}

	if !params.KeepMetadata {
		args = append(args, "--strip", "all")
	}
	args = append(args, "--out", params.ResultPath)
	args = append(args, params.ImagePath)

	cmd := exec.Command("oxipng", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("oxipng error: %v, output: %s", err, string(out))
	}
	return nil
}

func (co *compressorOxipng) Version() (string, error) {
	out, err := exec.Command("oxipng", "--version").Output()
	if err != nil {
		return "", fmt.Errorf("oxipng not found: %w", err)
	}

	return strings.TrimSpace(string(out)), nil
}

func (co *compressorOxipng) mapRatioToLevel(ratio int) int {
	if ratio <= 0 {
		return 2
	}

	level := ratio / 16
	if level > 6 {
		return 6
	}
	return level
}
