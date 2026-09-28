package converter

import (
	"context"
)

func (q *QPDF) Optimize(
	ctx context.Context,
	input string,
	output string,
) error {
	args := []string{
		"--warning-exit-0",

		"--object-streams=generate",

		"--compress-streams=y",

		"--decode-level=generalized",

		"--recompress-flate",

		"--compression-level=9",

		input,
		output,
	}

	return q.run(
		ctx,
		"optimize",
		args,
	)
}

func (q *QPDF) OptimizeLinearized(
	ctx context.Context,
	input string,
	output string,
) error {
	args := []string{
		"--warning-exit-0",

		"--object-streams=generate",

		"--compress-streams=y",

		"--decode-level=generalized",

		"--recompress-flate",

		"--compression-level=9",

		"--linearize",

		input,
		output,
	}

	return q.run(
		ctx,
		"optimize-linearized",
		args,
	)
}

func (q *QPDF) CheckLinearization(
	ctx context.Context,
	input string,
) error {
	args := []string{
		"--warning-exit-0",
		"--check-linearization",
		input,
	}

	return q.run(
		ctx,
		"check-linearization",
		args,
	)
}
