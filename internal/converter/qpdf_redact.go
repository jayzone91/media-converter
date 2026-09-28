package converter

import (
	"context"
	"fmt"
	"strconv"
)

func (q *QPDF) AssembleRedactedPDF(
	ctx context.Context,
	input string,
	replacements map[int]string,
	pageCount int,
	output string,
) error {
	if pageCount < 1 {
		return fmt.Errorf(
			"PDF contains no pages",
		)
	}

	if len(replacements) == 0 {
		return fmt.Errorf(
			"at least one redacted page is required",
		)
	}

	args := []string{
		"--warning-exit-0",
		"--stream-data=preserve",
		"--empty",
		"--pages",
	}

	for page := 1; page <= pageCount; page++ {
		if replacement,
			ok := replacements[page]; ok {
			args = append(
				args,
				replacement,
				"1",
			)

			continue
		}

		args = append(
			args,
			input,
			strconv.Itoa(
				page,
			),
		)
	}

	args = append(
		args,
		"--",
		output,
	)

	return q.run(
		ctx,
		"assemble redacted PDF",
		args,
	)
}

func (q *QPDF) SanitizeRedactedPDF(
	ctx context.Context,
	input string,
	output string,
) error {
	args := []string{
		"--warning-exit-0",

		"--remove-info",
		"--remove-metadata",
		"--remove-structure",

		"--generate-appearances",
		"--flatten-annotations=all",

		input,
		output,
	}

	return q.run(
		ctx,
		"sanitize redacted PDF",
		args,
	)
}

func (q *QPDF) CheckPDF(
	ctx context.Context,
	input string,
) error {
	args := []string{
		"--warning-exit-0",
		"--check",
		input,
	}

	return q.run(
		ctx,
		"check PDF",
		args,
	)
}
