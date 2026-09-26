// Package converter provides media conversion backends
package converter

import (
	"context"
	"fmt"
	"os/exec"
)

type FFmpeg struct {
	binary string
}

func NewFFmpeg() (*FFmpeg, error) {
	binary, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, fmt.Errorf("ffmpeg not found: %w", err)
	}

	return &FFmpeg{
		binary: binary,
	}, nil
}

func (c *FFmpeg) Convert(ctx context.Context, input, output string) error {
	cmd := exec.CommandContext(
		ctx,
		c.binary,
		"-y",
		"-i",
		input,
		output,
	)

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg failed: %w, %s", err, string(output))
	}

	return nil
}
