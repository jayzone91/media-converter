package web

import (
	"errors"
	"fmt"

	"github.com/jayzone91/media-converter/internal/converter"
)

var errPDFOptimizationSignedDocument = errors.New(
	"signed PDF must not be rewritten",
)

func validatePDFOptimizationInput(
	path string,
) error {
	hasSignatures, err :=
		converter.HasPDFSignatures(
			path,
		)

	if err != nil {
		return fmt.Errorf(
			"inspect PDF optimization compatibility: %w",
			err,
		)
	}

	if hasSignatures {
		return errPDFOptimizationSignedDocument
	}

	return nil
}
