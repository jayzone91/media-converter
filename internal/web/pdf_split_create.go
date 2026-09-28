package web

import (
	"context"
	"fmt"
	"path/filepath"
)

func (s *Server) createPDFSplitParts(
	ctx context.Context,
	input string,
	outputDirectory string,
	ranges []pdfPageRange,
) ([]string, error) {
	if len(ranges) < 2 {
		return nil,
			fmt.Errorf(
				"at least two PDF split ranges are required",
			)
	}

	if span, ok :=
		fixedPDFSplitSpan(
			ranges,
		); ok {
		files, err :=
			s.qpdf.SplitFixedSpan(
				ctx,
				input,
				span,
				outputDirectory,
			)
		if err != nil {
			return nil, err
		}

		if len(files) !=
			len(ranges) {
			return nil,
				fmt.Errorf(
					"qpdf produced %d split files, expected %d",
					len(files),
					len(ranges),
				)
		}

		return files, nil
	}

	files := make(
		[]string,
		0,
		len(ranges),
	)

	for index, pageRange := range ranges {
		outputPath :=
			filepath.Join(
				outputDirectory,
				fmt.Sprintf(
					"part-%03d.pdf",
					index+1,
				),
			)

		if err :=
			s.qpdf.ExtractRange(
				ctx,
				input,
				pageRange.Start,
				pageRange.End,
				outputPath,
			); err != nil {
			return nil,
				fmt.Errorf(
					"create PDF split part %d (%d-%d): %w",
					index+1,
					pageRange.Start,
					pageRange.End,
					err,
				)
		}

		files = append(
			files,
			outputPath,
		)
	}

	return files, nil
}

func fixedPDFSplitSpan(
	ranges []pdfPageRange,
) (int, bool) {
	if len(ranges) < 2 {
		return 0, false
	}

	if ranges[0].Start != 1 {
		return 0, false
	}

	span :=
		ranges[0].End -
			ranges[0].Start +
			1

	if span < 1 {
		return 0, false
	}

	for index, pageRange := range ranges {
		expectedStart :=
			index*span +
				1

		if pageRange.Start !=
			expectedStart {
			return 0, false
		}

		size :=
			pageRange.End -
				pageRange.Start +
				1

		last :=
			index ==
				len(ranges)-1

		if !last &&
			size != span {
			return 0, false
		}

		if last &&
			(size < 1 ||
				size > span) {
			return 0, false
		}
	}

	return span, true
}
