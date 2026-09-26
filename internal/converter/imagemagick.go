// Package converter provides media conversion backends.
package converter

import (
	"context"
	"fmt"
	"os/exec"
)

type ImageMagick struct {
	binary string
}

func NewImageMagick() (*ImageMagick, error) {
	binary, err := exec.LookPath("magick")
	if err != nil {
		return nil, fmt.Errorf("imagemagick not found: %w", err)
	}

	return &ImageMagick{
		binary: binary,
	}, nil
}

func (c *ImageMagick) Convert(ctx context.Context, input, output string) error {
	cmd := exec.CommandContext(ctx, c.binary, input, output)

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("imagemagick failed: %w: %s", err, string(output))
	}

	return nil
}
