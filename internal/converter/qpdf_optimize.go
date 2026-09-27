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
		"--stream-data=compress",
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
