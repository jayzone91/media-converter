package converter

import (
	"fmt"
	"os"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

func RemoveAllPDFAttachments(
	input string,
	output string,
) (int, error) {
	file, err :=
		os.Open(
			input,
		)
	if err != nil {
		return 0,
			fmt.Errorf(
				"open PDF for attachment inspection: %w",
				err,
			)
	}

	attachments, listErr :=
		api.Attachments(
			file,
			nil,
		)

	closeErr :=
		file.Close()

	if listErr != nil {
		return 0,
			fmt.Errorf(
				"inspect PDF attachments: %w",
				listErr,
			)
	}

	if closeErr != nil {
		return 0,
			fmt.Errorf(
				"close PDF after attachment inspection: %w",
				closeErr,
			)
	}

	if len(attachments) == 0 {
		if err :=
			copyPDFFile(
				input,
				output,
			); err != nil {
			return 0, err
		}

		return 0, nil
	}

	if err :=
		api.RemoveAttachmentsFile(
			input,
			output,
			nil,
			nil,
		); err != nil {
		return 0,
			fmt.Errorf(
				"remove PDF attachments: %w",
				err,
			)
	}

	return len(attachments), nil
}

func copyPDFFile(
	input string,
	output string,
) error {
	source, err :=
		os.Open(
			input,
		)
	if err != nil {
		return fmt.Errorf(
			"open PDF for copy: %w",
			err,
		)
	}

	defer source.Close()

	target, err :=
		os.OpenFile(
			output,
			os.O_WRONLY|
				os.O_CREATE|
				os.O_TRUNC,
			0600,
		)
	if err != nil {
		return fmt.Errorf(
			"create PDF copy: %w",
			err,
		)
	}

	if _, err :=
		target.ReadFrom(
			source,
		); err != nil {
		_ = target.Close()

		return fmt.Errorf(
			"copy PDF: %w",
			err,
		)
	}

	if err :=
		target.Close(); err != nil {
		return fmt.Errorf(
			"close PDF copy: %w",
			err,
		)
	}

	return nil
}
