package web

import (
	"errors"
	"fmt"

	"github.com/jayzone91/media-converter/internal/converter"
)

var errPDFCompressionInteractiveForm = errors.New(
	"interactive PDF form requires lossless compression",
)

func validatePDFCompressionInput(
	path string,
	mode string,
) error {
	if mode == "lossless" {
		return nil
	}

	hasForms, err :=
		converter.HasPDFFormFields(
			path,
		)

	if err != nil {
		return fmt.Errorf(
			"inspect PDF compression compatibility: %w",
			err,
		)
	}

	if hasForms {
		return errPDFCompressionInteractiveForm
	}

	return nil
}
