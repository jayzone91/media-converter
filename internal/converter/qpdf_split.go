package converter

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
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

	pageRange :=
		fmt.Sprintf(
			"%d-%d",
			startPage,
			endPage,
		)

	if startPage == endPage {
		pageRange =
			strconv.Itoa(
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

func (q *QPDF) SplitFixedSpan(
	ctx context.Context,
	input string,
	span int,
	outputDirectory string,
) ([]string, error) {
	if span < 1 {
		return nil,
			fmt.Errorf(
				"invalid PDF split span: %d",
				span,
			)
	}

	outputPattern :=
		filepath.Join(
			outputDirectory,
			"split.pdf",
		)

	args := []string{
		"--warning-exit-0",
		"--stream-data=preserve",
		"--split-pages=" +
			strconv.Itoa(
				span,
			),
		input,
		outputPattern,
	}

	if err := q.run(
		ctx,
		"split-pages",
		args,
	); err != nil {
		return nil, err
	}

	files, err :=
		filepath.Glob(
			filepath.Join(
				outputDirectory,
				"split-*.pdf",
			),
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"find split PDF files: %w",
				err,
			)
	}

	if len(files) == 0 {
		return nil,
			fmt.Errorf(
				"qpdf split produced no files",
			)
	}

	sort.Strings(
		files,
	)

	return files, nil
}
