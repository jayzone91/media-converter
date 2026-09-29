package converter

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

func (c *PDF) extractTextWithOCR(
	ctx context.Context,
	input string,
	output string,
) error {
	tempDir, err := os.MkdirTemp(
		filepath.Dir(input),
		"pdf-ocr-*",
	)
	if err != nil {
		return fmt.Errorf(
			"failed to create OCR temp dir: %w",
			err,
		)
	}
	defer os.RemoveAll(tempDir)

	prefix := filepath.Join(
		tempDir,
		"page",
	)

	cmd := externalCommandContext(
		ctx,
		c.pdfToPPM,
		"-png",
		"-r",
		"300",
		input,
		prefix,
	)

	if err :=
		runExternalTool(
			ctx,
			"pdftoppm",
			cmd,
		); err != nil {
		return err
	}

	pages, err := filepath.Glob(
		prefix + "-*.png",
	)
	if err != nil {
		return fmt.Errorf(
			"failed to find rendered PDF pages: %w",
			err,
		)
	}

	if len(pages) == 0 {
		return fmt.Errorf(
			"pdftoppm produced no pages",
		)
	}

	sortPDFPagePaths(pages)

	textPages := make(
		[]string,
		0,
		len(pages),
	)

	for index, page := range pages {
		pageText, err := c.ocrPage(
			ctx,
			page,
		)
		if err != nil {
			return fmt.Errorf(
				"OCR failed for page %d: %w",
				index+1,
				err,
			)
		}

		textPages = append(
			textPages,
			pageText,
		)
	}

	return writePDFTextPages(
		output,
		textPages,
	)
}

func (c *PDF) ocrPDFPage(
	ctx context.Context,
	input string,
	pageNumber int,
) (string, error) {
	tempDir, err := os.MkdirTemp(
		filepath.Dir(input),
		"pdf-ocr-page-*",
	)
	if err != nil {
		return "", fmt.Errorf(
			"failed to create page OCR temp dir: %w",
			err,
		)
	}
	defer os.RemoveAll(tempDir)

	prefix := filepath.Join(
		tempDir,
		"page",
	)

	page := strconv.Itoa(
		pageNumber,
	)

	cmd := externalCommandContext(
		ctx,
		c.pdfToPPM,
		"-f",
		page,
		"-l",
		page,
		"-singlefile",
		"-png",
		"-r",
		"300",
		input,
		prefix,
	)

	if err :=
		runExternalTool(
			ctx,
			"pdftoppm",
			cmd,
		); err != nil {
		return "", err
	}

	imagePath := prefix + ".png"

	if _, err := os.Stat(imagePath); err != nil {
		return "", fmt.Errorf(
			"rendered PDF page not created: %w",
			err,
		)
	}

	return c.ocrPage(
		ctx,
		imagePath,
	)
}

func (c *PDF) ocrPage(
	ctx context.Context,
	image string,
) (string, error) {
	cmd :=
		externalCommandContext(
			ctx,
			c.tesseract,
			image,
			"stdout",
			"--tessdata-dir",
			c.tessdataDir,
			"-l",
			"deu+eng",
			"--psm",
			"3",
		)

	stdout, _, err :=
		runExternalToolCapture(
			ctx,
			"tesseract",
			cmd,
		)

	if err != nil {
		return "", err
	}

	return stdout, nil
}
