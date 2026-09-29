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
	binary, err :=
		exec.LookPath(
			"magick",
		)

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
	cmd :=
		externalCommandContext(
			ctx,
			c.binary,
			"identify",
			"-regard-warnings",
			input,
		)

	return runExternalTool(
		ctx,
		"imagemagick",
		cmd,
	)
}

func (c *ImageMagick) Convert(
	ctx context.Context,
	input string,
	output string,
) error {
	cmd :=
		externalCommandContext(
			ctx,
			c.binary,
			input,
			output,
		)

	return runExternalTool(
		ctx,
		"imagemagick",
		cmd,
	)
}
