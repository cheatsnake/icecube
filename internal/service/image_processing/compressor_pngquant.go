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

func (cp *compressorPngquant) Compress(params CompressorParams) error {
	quality := fmt.Sprintf("0-%d", params.CompressionRatio)
	args := []string{
		"--quality", quality,
		"--force",
		"--output", params.ResultPath,
		params.ImagePath,
	}

	if !params.KeepMetadata {
		args = append([]string{"--strip"}, args...)
	}

	for key, val := range params.Extra {
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
