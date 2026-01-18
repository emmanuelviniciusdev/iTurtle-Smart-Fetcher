package downloader

import (
	"context"
	"fmt"
	"os/exec"
)

// Runner executes external commands and returns their combined output.
type Runner interface {
	Run(ctx context.Context, name string, args ...string) (string, error)
}

// ExecRunner executes commands using the local shell utilities (yt-dlp, ffmpeg).
type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("%s failed: %w (output: %s)", name, err, output)
	}
	return string(output), nil
}
