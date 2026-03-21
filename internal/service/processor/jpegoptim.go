package processor

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type jpegoptim struct {
	logger *slog.Logger
}

func newJpegoptim(logger *slog.Logger) (*jpegoptim, error) {
	cli := &jpegoptim{logger: logger}
	_, err := cli.Version()
	if err != nil {
		logger.Error("jpegoptim not found", "error", err)
		return nil, err
	}

	return cli, nil
}

func (jo *jpegoptim) Compress(params CompressorParams) error {
	jo.logger.Debug("Running jpegoptim", "quality", params.Quality, "keepMetadata", params.KeepMetadata, "input", params.ImagePath)
	tmpDir, err := os.MkdirTemp(filepath.Dir(params.ResultPath), "jpegoptim")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	args := []string{
		fmt.Sprintf("-m%d", params.Quality),
		"--dest=" + tmpDir,
	}

	if params.KeepMetadata {
		args = append(args, "--preserve")
	} else {
		args = append(args, "--strip-all")
	}

	for key, val := range params.Extra {
		args = append(args, fmt.Sprintf("--%s", key), fmt.Sprintf("%v", val))
	}
	args = append(args, params.ImagePath)

	cmd := exec.Command("jpegoptim", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("jpegoptim error: %v, output: %s", err, string(output))
	}

	tmpFile := filepath.Join(tmpDir, filepath.Base(params.ImagePath))
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		return fmt.Errorf("jpegoptim failed to compress image")
	}

	return os.Rename(tmpFile, params.ResultPath)

}

func (jo *jpegoptim) Version() (string, error) {
	out, err := exec.Command("jpegoptim", "--version").Output()
	if err != nil {
		return "", fmt.Errorf("jpegoptim not found: %w", err)
	}

	lines := strings.Split(string(out), "\n")
	return strings.TrimSpace(lines[0]), nil
}
