package converter

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"
)

const minimumPDFTextCharacters = 40

func (c *PDF) extractHybridText(
	ctx context.Context,
	input string,
	output string,
) error {
	tempDir, err := os.MkdirTemp(
		filepath.Dir(input),
		"pdf-text-*",
	)
	if err != nil {
		return fmt.Errorf(
			"failed to create PDF text temp dir: %w",
			err,
		)
	}
	defer os.RemoveAll(tempDir)

	extractedPath := filepath.Join(
		tempDir,
		"extracted.txt",
	)

	if err := c.extractText(
		ctx,
		input,
		extractedPath,
	); err != nil {
		return c.extractTextWithOCR(
			ctx,
			input,
			output,
		)
	}

	content, err := os.ReadFile(
		extractedPath,
	)
	if err != nil {
		return fmt.Errorf(
			"failed to read extracted PDF text: %w",
			err,
		)
	}

	pages := splitPDFTextPages(
		string(content),
	)
	if len(pages) == 0 {
		return c.extractTextWithOCR(
			ctx,
			input,
			output,
		)
	}

	result := make(
		[]string,
		len(pages),
	)

	for index, pageText := range pages {
		if hasUsableText(pageText) {
			result[index] = pageText
			continue
		}

		ocrText, err := c.ocrPDFPage(
			ctx,
			input,
			index+1,
		)
		if err != nil {
			return fmt.Errorf(
				"OCR failed for page %d: %w",
				index+1,
				err,
			)
		}

		result[index] = ocrText
	}

	return writePDFTextPages(
		output,
		result,
	)
}

func (c *PDF) extractText(
	ctx context.Context,
	input string,
	output string,
) error {
	cmd := exec.CommandContext(
		ctx,
		c.pdfToText,
		"-layout",
		"-enc",
		"UTF-8",
		input,
		output,
	)

	if result, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf(
			"pdftotext failed: %w: %s",
			err,
			string(result),
		)
	}

	return nil
}

func splitPDFTextPages(
	content string,
) []string {
	pages := strings.Split(
		content,
		"\f",
	)

	for len(pages) > 0 &&
		strings.TrimSpace(
			pages[len(pages)-1],
		) == "" {
		pages = pages[:len(pages)-1]
	}

	if len(pages) == 0 {
		return nil
	}

	return pages
}

func hasUsableText(
	content string,
) bool {
	characters := 0

	for _, r := range content {
		if unicode.IsLetter(r) ||
			unicode.IsNumber(r) {
			characters++

			if characters >= minimumPDFTextCharacters {
				return true
			}
		}
	}

	return false
}

func writePDFTextPages(
	output string,
	pages []string,
) error {
	var text strings.Builder

	for index, page := range pages {
		if index > 0 {
			text.WriteString("\n\f\n")
		}

		text.WriteString(
			strings.TrimRight(
				page,
				" \t\r\n",
			),
		)
	}

	if err := os.WriteFile(
		output,
		[]byte(text.String()),
		0600,
	); err != nil {
		return fmt.Errorf(
			"failed to write PDF text: %w",
			err,
		)
	}

	return nil
}
