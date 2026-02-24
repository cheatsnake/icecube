package processor

import (
	"fmt"
	"os/exec"
)

type pngquant struct{}

func newPngquant() (*pngquant, error) {
	cli := &pngquant{}
	_, err := cli.Version()

	return cli, err
}

func (pq *pngquant) Compress(params CompressorParams) error {
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

func (pq *pngquant) Version() (string, error) {
	out, err := exec.Command("pngquant", "--version").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
