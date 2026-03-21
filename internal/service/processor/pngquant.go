package processor

import (
	"fmt"
	"log/slog"
	"os/exec"
)

type pngquant struct {
	logger *slog.Logger
}

func newPngquant(logger *slog.Logger) (*pngquant, error) {
	cli := &pngquant{logger: logger}
	_, err := cli.Version()
	if err != nil {
		logger.Error("pngquant not found", "error", err)
		return nil, err
	}

	return cli, nil
}

func (pq *pngquant) Compress(params CompressorParams) error {
	pq.logger.Debug("Running pngquant", "quality", params.Quality, "keepMetadata", params.KeepMetadata, "input", params.ImagePath)
	quality := fmt.Sprintf("0-%d", params.Quality)
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
