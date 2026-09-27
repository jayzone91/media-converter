package converter

import (
	"context"
	"fmt"
)

func (q *QPDF) ExtractRange(
	ctx context.Context,
	input string,
	startPage int,
	endPage int,
	output string,
) error {
	if startPage < 1 {
		return fmt.Errorf(
			"invalid PDF start page: %d",
			startPage,
		)
	}

	if endPage < startPage {
		return fmt.Errorf(
			"invalid PDF page range: %d-%d",
			startPage,
			endPage,
		)
	}

	pageRange := fmt.Sprintf(
		"%d-%d",
		startPage,
		endPage,
	)

	if startPage == endPage {
		pageRange = fmt.Sprintf(
			"%d",
			startPage,
		)
	}

	args := []string{
		"--warning-exit-0",
		"--stream-data=preserve",
		"--empty",
		"--pages",
		input,
		pageRange,
		"--",
		output,
	}

	return q.run(
		ctx,
		"extract-range",
		args,
	)
}
