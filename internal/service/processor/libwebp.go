package processor

import (
	"fmt"
	"log/slog"
	"os/exec"
)

type libwebp struct {
	logger *slog.Logger
}

func newLibwebp(logger *slog.Logger) (*libwebp, error) {
	cli := &libwebp{logger: logger}
	_, err := cli.Version()
	if err != nil {
		logger.Error("libwebp (cwebp) not found", "error", err)
		return nil, err
	}

	return cli, nil
}

func (lw *libwebp) Compress(params CompressorParams) error {
	lw.logger.Debug("Running libwebp (cwebp)", "quality", params.Quality, "keepMetadata", params.KeepMetadata, "input", params.ImagePath)
	args := []string{
		"-q", fmt.Sprintf("%d", params.Quality),
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

func (lw *libwebp) Version() (string, error) {
	out, err := exec.Command("cwebp", "-version").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
