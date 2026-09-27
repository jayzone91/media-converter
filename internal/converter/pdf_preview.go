package converter

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

func (c *PDF) RenderPreviewPage(
	ctx context.Context,
	input string,
	output string,
	page int,
) error {
	if page < 1 {
		return fmt.Errorf(
			"invalid PDF page number: %d",
			page,
		)
	}

	if err := os.MkdirAll(
		filepath.Dir(output),
		0700,
	); err != nil {
		return fmt.Errorf(
			"failed to create PDF preview directory: %w",
			err,
		)
	}

	tempFile, err := os.CreateTemp(
		filepath.Dir(output),
		"preview-*.jpg",
	)
	if err != nil {
		return fmt.Errorf(
			"failed to create PDF preview temp file: %w",
			err,
		)
	}

	tempPath := tempFile.Name()

	if err := tempFile.Close(); err != nil {
		_ = os.Remove(
			tempPath,
		)

		return fmt.Errorf(
			"failed to close PDF preview temp file: %w",
			err,
		)
	}

	if err := os.Remove(
		tempPath,
	); err != nil {
		return fmt.Errorf(
			"failed to prepare PDF preview temp file: %w",
			err,
		)
	}

	tempPrefix :=
		tempPath[:len(tempPath)-len(filepath.Ext(tempPath))]

	pageString :=
		strconv.Itoa(page)

	cmd := exec.CommandContext(
		ctx,
		c.pdfToPPM,
		"-f",
		pageString,
		"-l",
		pageString,
		"-singlefile",
		"-jpeg",
		"-jpegopt",
		"quality=75",
		"-r",
		"90",
		input,
		tempPrefix,
	)

	result, err := cmd.CombinedOutput()
	if err != nil {
		_ = os.Remove(
			tempPrefix + ".jpg",
		)

		return fmt.Errorf(
			"pdftoppm preview rendering failed: %w: %s",
			err,
			string(result),
		)
	}

	renderedPath :=
		tempPrefix + ".jpg"

	if _, err := os.Stat(
		renderedPath,
	); err != nil {
		return fmt.Errorf(
			"PDF preview was not created: %w",
			err,
		)
	}

	if err := os.Rename(
		renderedPath,
		output,
	); err != nil {
		if _, statErr :=
			os.Stat(
				output,
			); statErr == nil {
			_ = os.Remove(
				renderedPath,
			)

			return nil
		}

		_ = os.Remove(
			renderedPath,
		)

		return fmt.Errorf(
			"failed to store PDF preview: %w",
			err,
		)
	}

	return nil
}
