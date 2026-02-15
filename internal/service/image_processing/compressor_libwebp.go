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

func (cl *compressorLibwebp) Compress(imagePath, resultPath string, compressionRatio int, keepMetadata bool, extra map[string]any) error {
	args := []string{
		"-q", fmt.Sprintf("%d", compressionRatio),
		imagePath,
		"-o", resultPath,
	}

	if keepMetadata {
		args = append(args, "-metadata", "all")
	}

	for key, val := range extra {
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
