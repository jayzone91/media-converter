// Package converter provides media conversion backends.
package converter

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type ImageMagick struct {
	binary string
}

func NewImageMagick() (*ImageMagick, error) {
	binary, err := exec.LookPath("magick")
	if err != nil {
		return nil,
			fmt.Errorf(
				"imagemagick not found: %w",
				err,
			)
	}

	return &ImageMagick{
		binary: binary,
	}, nil
}

func (c *ImageMagick) Validate(
	ctx context.Context,
	input string,
) error {
	cmd := externalCommandContext(
		ctx,
		c.binary,
		"identify",
		"-regard-warnings",
		input,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return fmt.Errorf(
				"imagemagick validation failed: %w",
				ctxErr,
			)
		}

		message := strings.TrimSpace(
			string(output),
		)

		return fmt.Errorf(
			"imagemagick validation failed: %w: %s",
			err,
			message,
		)
	}

	return nil
}

func (c *ImageMagick) Convert(
	ctx context.Context,
	input string,
	output string,
) error {
	cmd := externalCommandContext(
		ctx,
		c.binary,
		input,
		output,
	)

	if output, err :=
		cmd.CombinedOutput(); err != nil {
		return fmt.Errorf(
			"imagemagick failed: %w: %s",
			err,
			string(output),
		)
	}

	return nil
}
