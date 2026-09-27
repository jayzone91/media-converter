package converter

import (
	"context"
)

func (q *QPDF) Optimize(
	ctx context.Context,
	input string,
	output string,
) error {
	return q.optimize(
		ctx,
		input,
		output,
		false,
	)
}

func (q *QPDF) OptimizeLinearized(
	ctx context.Context,
	input string,
	output string,
) error {
	return q.optimize(
		ctx,
		input,
		output,
		true,
	)
}

func (q *QPDF) optimize(
	ctx context.Context,
	input string,
	output string,
	linearize bool,
) error {
	args := []string{
		"--warning-exit-0",
		"--object-streams=generate",
		"--stream-data=compress",
		"--recompress-flate",
		"--compression-level=9",
	}

	if linearize {
		args = append(
			args,
			"--linearize",
		)
	}

	args = append(
		args,
		input,
		output,
	)

	return q.run(
		ctx,
		"optimize",
		args,
	)
}
