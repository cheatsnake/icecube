package image_processing

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type compressorJpegoptim struct{}

func newCompressorJpegoptim() (*compressorJpegoptim, error) {
	cli := &compressorJpegoptim{}
	_, err := cli.Version()

	return cli, err
}

func (cj *compressorJpegoptim) Compress(imagePath, resultPath string, compressionRatio int, keepMetadata bool, extra map[string]any) error {
	tmpDir, err := os.MkdirTemp(filepath.Dir(resultPath), "jpegoptim")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	args := []string{
		fmt.Sprintf("-m%d", compressionRatio),
		"--dest=" + tmpDir,
	}

	if keepMetadata {
		args = append(args, "--preserve")
	} else {
		args = append(args, "--strip-all")
	}

	for key, val := range extra {
		args = append(args, fmt.Sprintf("--%s", key), fmt.Sprintf("%v", val))
	}
	args = append(args, imagePath)

	cmd := exec.Command("jpegoptim", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("jpegoptim error: %v, output: %s", err, string(output))
	}

	tmpFile := filepath.Join(tmpDir, filepath.Base(imagePath))
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		return fmt.Errorf("jpegoptim failed to compress image")
	}

	return os.Rename(tmpFile, resultPath)

}

func (cj *compressorJpegoptim) Version() (string, error) {
	out, err := exec.Command("jpegoptim", "--version").Output()
	if err != nil {
		return "", fmt.Errorf("jpegoptim not found: %w", err)
	}

	lines := strings.Split(string(out), "\n")
	return strings.TrimSpace(lines[0]), nil
}
