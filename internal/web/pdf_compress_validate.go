package web

import (
	"context"
	"fmt"
)

func (s *Server) validatePDFCompressionOutput(
	ctx context.Context,
	path string,
	expectedPageCount int,
) error {
	if expectedPageCount < 1 {
		return fmt.Errorf(
			"invalid expected PDF page count: %d",
			expectedPageCount,
		)
	}

	if err :=
		s.qpdf.CheckPDF(
			ctx,
			path,
		); err != nil {
		return fmt.Errorf(
			"compressed PDF failed structural validation: %w",
			err,
		)
	}

	pageCount, err :=
		s.qpdf.PageCount(
			ctx,
			path,
		)

	if err != nil {
		return fmt.Errorf(
			"read compressed PDF page count: %w",
			err,
		)
	}

	if pageCount.Count !=
		expectedPageCount {
		return fmt.Errorf(
			"compressed PDF page count changed: expected %d, got %d",
			expectedPageCount,
			pageCount.Count,
		)
	}

	return nil
}
