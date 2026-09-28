package web

import (
	"context"
	"fmt"
)

func (s *Server) validatePDFOptimizationOutput(
	ctx context.Context,
	path string,
	expectedPageCount int,
	linearized bool,
) error {
	if err :=
		s.qpdf.CheckPDF(
			ctx,
			path,
		); err != nil {
		return fmt.Errorf(
			"optimized PDF failed structural validation: %w",
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
			"read optimized PDF page count: %w",
			err,
		)
	}

	if pageCount.Count !=
		expectedPageCount {
		return fmt.Errorf(
			"optimized PDF page count changed: expected %d, got %d",
			expectedPageCount,
			pageCount.Count,
		)
	}

	if linearized {
		if err :=
			s.qpdf.CheckLinearization(
				ctx,
				path,
			); err != nil {
			return fmt.Errorf(
				"optimized PDF is not correctly linearized: %w",
				err,
			)
		}
	}

	return nil
}
