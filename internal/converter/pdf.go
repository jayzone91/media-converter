package converter

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

const minimumPDFTextCharacters = 40

type PDF struct {
	pdfToText   string
	pdfToPPM    string
	tesseract   string
	libreOffice *LibreOffice
}

func NewPDF(libreOffice *LibreOffice) (*PDF, error) {
	pdfToText, err := exec.LookPath("pdftotext")
	if err != nil {
		return nil, fmt.Errorf("pdftotext not found: %w", err)
	}

	pdfToPPM, err := exec.LookPath("pdftoppm")
	if err != nil {
		return nil, fmt.Errorf("pdftoppm not found: %w", err)
	}

	tesseract, err := exec.LookPath("tesseract")
	if err != nil {
		return nil, fmt.Errorf("tesseract not found: %w", err)
	}

	return &PDF{
		pdfToText:   pdfToText,
		pdfToPPM:    pdfToPPM,
		tesseract:   tesseract,
		libreOffice: libreOffice,
	}, nil
}

func (c *PDF) ConvertToDOCX(ctx context.Context, input, output string) error {
	textPath := strings.TrimSuffix(input, filepath.Ext(input)) + ".txt"

	if err := c.extractText(ctx, input, textPath); err != nil {
		if err := c.extractTextWithOCR(ctx, input, textPath); err != nil {
			return err
		}
	} else {
		useable, err := hasUsablePDFText(textPath)
		if err != nil {
			return err
		}

		if !useable {
			if err := c.extractTextWithOCR(ctx, input, textPath); err != nil {
				return err
			}
		}
	}

	if err := c.libreOffice.Convert(ctx, textPath, output); err != nil {
		return fmt.Errorf("failed to create docx: %w", err)
	}

	if _, err := os.Stat(output); err != nil {
		return fmt.Errorf("docx output not created: %w", err)
	}

	return nil
}

func (c *PDF) extractText(ctx context.Context, input, output string) error {
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
		return fmt.Errorf("pdftotext failed: %w, %s", err, string(result))
	}

	return nil
}

func (c *PDF) extractTextWithOCR(ctx context.Context, input, output string) error {
	dir := filepath.Dir(input)
	prefix := filepath.Join(dir, "ocr-page")

	cmd := exec.CommandContext(
		ctx,
		c.pdfToPPM,
		"-png",
		"-r",
		"300",
		input,
		prefix,
	)

	if result, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("pdftoppm failed: %w, %s", err, string(result))
	}

	pages, err := filepath.Glob(prefix + "-*.png")
	if err != nil {
		return fmt.Errorf("failed to find rendered PDF pages: %w", err)
	}

	if len(pages) == 0 {
		return fmt.Errorf("pdftoppm produced no pages")
	}

	sort.Slice(pages, func(i, j int) bool {
		return pdfPageNumber(pages[i]) < pdfPageNumber(pages[j])
	})

	var text strings.Builder

	for index, page := range pages {
		pageText, err := c.ocrPage(ctx, page)
		if err != nil {
			return fmt.Errorf("OCR failed for page %d: %w", index+1, err)
		}

		if index > 0 {
			text.WriteString("\n\n")
		}

		text.WriteString(pageText)
	}

	if err := os.WriteFile(output, []byte(text.String()), 0600); err != nil {
		return fmt.Errorf("failed to write OCR text: %w", err)
	}
	return nil
}

func (c *PDF) ocrPage(ctx context.Context, image string) (string, error) {
	cmd := exec.CommandContext(
		ctx,
		c.tesseract,
		image,
		"stdout",
		"-l",
		"deu+eng",
		"--psm",
		"3",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("tesseract failed: %w, %s", err, string(output))
	}

	return string(output), nil
}

func hasUsablePDFText(path string) (bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("failed to read extracted PDF text: %w", err)
	}

	characters := 0

	for _, r := range string(content) {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			characters++

			if characters >= minimumPDFTextCharacters {
				return true, nil
			}
		}
	}

	return false, nil
}

func pdfPageNumber(path string) int {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))

	index := strings.LastIndex(name, "-")
	if index == -1 {
		return 0
	}

	number, err := strconv.Atoi(name[index+1:])
	if err != nil {
		return 0
	}

	return number
}
