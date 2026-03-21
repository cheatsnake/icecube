package processor

import (
	"fmt"
	"os/exec"
	"strings"
)

type oxipng struct{}

func newOxipng() (*oxipng, error) {
	cli := &oxipng{}
	_, err := cli.Version()

	return cli, err
}

func (op *oxipng) Compress(params CompressorParams) error {
	level := op.mapRatioToLevel(params.Quality)
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

func (op *oxipng) Version() (string, error) {
	out, err := exec.Command("oxipng", "--version").Output()
	if err != nil {
		return "", fmt.Errorf("oxipng not found: %w", err)
	}

	return strings.TrimSpace(string(out)), nil
}

func (op *oxipng) mapRatioToLevel(ratio int) int {
	if ratio <= 0 {
		return 2
	}

	level := ratio / 16
	if level > 6 {
		return 6
	}
	return level
}
