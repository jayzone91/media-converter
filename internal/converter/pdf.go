package converter

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type PDF struct {
	pdfToText   string
	pdfToPPM    string
	tesseract   string
	tessdataDir string
	libreOffice *LibreOffice
}

func NewPDF(
	libreOffice *LibreOffice,
) (*PDF, error) {
	pdfToText, err := exec.LookPath(
		"pdftotext",
	)
	if err != nil {
		return nil, fmt.Errorf(
			"pdftotext not found: %w",
			err,
		)
	}

	pdfToPPM, err := exec.LookPath(
		"pdftoppm",
	)
	if err != nil {
		return nil, fmt.Errorf(
			"pdftoppm not found: %w",
			err,
		)
	}

	tesseract, err := exec.LookPath(
		"tesseract",
	)
	if err != nil {
		return nil, fmt.Errorf(
			"tesseract not found: %w",
			err,
		)
	}

	tessdataDir, err := findTessdataDirectory(
		tesseract,
	)
	if err != nil {
		return nil, err
	}

	return &PDF{
		pdfToText:   pdfToText,
		pdfToPPM:    pdfToPPM,
		tesseract:   tesseract,
		tessdataDir: tessdataDir,
		libreOffice: libreOffice,
	}, nil
}

func (c *PDF) ConvertToDOCX(
	ctx context.Context,
	input string,
	output string,
) error {
	textPath := strings.TrimSuffix(
		input,
		filepath.Ext(input),
	) + ".txt"

	if err := c.extractHybridText(
		ctx,
		input,
		textPath,
	); err != nil {
		return err
	}

	if err := c.libreOffice.Convert(
		ctx,
		textPath,
		output,
	); err != nil {
		return fmt.Errorf(
			"failed to create docx: %w",
			err,
		)
	}

	if _, err := os.Stat(output); err != nil {
		return fmt.Errorf(
			"docx output not created: %w",
			err,
		)
	}

	return nil
}
