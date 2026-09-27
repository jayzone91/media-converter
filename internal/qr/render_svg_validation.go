package qr

import (
	"fmt"
	"regexp"
)

var hexColorPattern = regexp.MustCompile(
	`^#[0-9a-fA-F]{6}$`,
)

func validateSVGInput(
	matrix Matrix,
	style Style,
) error {
	if matrix.Size == 0 ||
		len(matrix.Bitmap) != matrix.Size {
		return fmt.Errorf(
			"invalid QR matrix",
		)
	}

	for _, row := range matrix.Bitmap {
		if len(row) != matrix.Size {
			return fmt.Errorf(
				"invalid QR matrix",
			)
		}
	}

	return validateStyleColors(
		style,
	)
}

func validateStyleColors(
	style Style,
) error {
	if !hexColorPattern.MatchString(
		style.Foreground,
	) {
		return fmt.Errorf(
			"invalid foreground color",
		)
	}

	if !hexColorPattern.MatchString(
		style.Background,
	) {
		return fmt.Errorf(
			"invalid background color",
		)
	}

	if !style.Gradient.Enabled {
		return nil
	}

	if !hexColorPattern.MatchString(
		style.Gradient.Start,
	) {
		return fmt.Errorf(
			"invalid gradient start color",
		)
	}

	if !hexColorPattern.MatchString(
		style.Gradient.End,
	) {
		return fmt.Errorf(
			"invalid gradient end color",
		)
	}

	return nil
}
