package image_processing

import (
	"fmt"
	"os/exec"
)

type compressorLibwebp struct{}

func newCompressorLibwebp() (*compressorLibwebp, error) {
	cli := &compressorLibwebp{}
	_, err := cli.Version()

	return cli, err
}

func (cl *compressorLibwebp) Compress(params CompressorParams) error {
	args := []string{
		"-q", fmt.Sprintf("%d", params.CompressionRatio),
		params.ImagePath,
		"-o", params.ResultPath,
	}

	if params.KeepMetadata {
		args = append(args, "-metadata", "all")
	}

	for key, val := range params.Extra {
		args = append(args, fmt.Sprintf("-%s", key), fmt.Sprintf("%v", val))
	}

	cmd := exec.Command("cwebp", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("cwebp error: %v, output: %s", err, string(output))
	}

	return nil
}

func (cl *compressorLibwebp) Version() (string, error) {
	out, err := exec.Command("cwebp", "-version").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
