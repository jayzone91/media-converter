package converter

import (
	"fmt"
	"os"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

func HasPDFFormFields(
	path string,
) (bool, error) {
	file, err :=
		os.Open(
			path,
		)

	if err != nil {
		return false,
			fmt.Errorf(
				"open PDF for form inspection: %w",
				err,
			)
	}

	defer file.Close()

	fields, err :=
		api.FormFields(
			file,
			nil,
		)

	if err != nil {
		return false,
			fmt.Errorf(
				"inspect PDF form fields: %w",
				err,
			)
	}

	return len(fields) > 0,
		nil
}
