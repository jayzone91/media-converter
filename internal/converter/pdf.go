package converter

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
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
		usable, err := hasUsablePDFText(textPath)
		if err != nil {
			return err
		}

		if !usable {
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

func (c *PDF) ConvertToImages(ctx context.Context, input, output, format string) error {
	switch format {
	case "png", "jpeg":
	default:
		return fmt.Errorf("unsupported PDF image format: %s", format)
	}

	tempDir, err := os.MkdirTemp(filepath.Dir(input), "pdf-pages-*")
	if err != nil {
		return fmt.Errorf("failed to create PDF image temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	prefix := filepath.Join(tempDir, "page")

	args := []string{
		"-r",
		"200",
	}

	switch format {
	case "png":
		args = append(args, "-png")

	case "jpeg":
		args = append(
			args,
			"-jpeg",
			"-jpegopt",
			"quality=90",
		)
	}

	args = append(args, input, prefix)

	cmd := exec.CommandContext(ctx, c.pdfToPPM, args...)

	if result, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("pdftoppm failed: %w: %s", err, string(result))
	}

	extension := "." + format
	if format == "jpeg" {
		extension = ".jpg"
	}

	pages, err := filepath.Glob(prefix + "-*" + extension)
	if err != nil {
		return fmt.Errorf("failed to find rendered PDF pages: %w", err)
	}

	if len(pages) == 0 {
		return fmt.Errorf("pdftoppm produced no pages")
	}

	sort.Slice(pages, func(i, j int) bool {
		return pdfPageNumber(pages[i]) < pdfPageNumber(pages[j])
	})

	if err := createImageZIP(output, pages, extension); err != nil {
		return fmt.Errorf("failed to create PDF image archive: %w", err)
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
		return fmt.Errorf("pdftotext failed: %w: %s", err, string(result))
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
		return fmt.Errorf("pdftoppm failed: %w: %s", err, string(result))
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

	if err := os.WriteFile(
		output,
		[]byte(text.String()),
		0600,
	); err != nil {
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
		return "", fmt.Errorf(
			"tesseract failed: %w: %s",
			err,
			string(output),
		)
	}

	return string(output), nil
}

func createImageZIP(output string, pages []string, extension string) error {
	file, err := os.Create(output)
	if err != nil {
		return err
	}

	archive := zip.NewWriter(file)

	for index, page := range pages {
		if err := addFileToZIP(
			archive,
			page,
			fmt.Sprintf(
				"page-%d%s",
				index+1,
				extension,
			),
		); err != nil {
			archive.Close()
			file.Close()
			return err
		}
	}

	if err := archive.Close(); err != nil {
		file.Close()
		return err
	}

	if err := file.Close(); err != nil {
		return err
	}

	return nil
}

func addFileToZIP(archive *zip.Writer, path, name string) error {
	source, err := os.Open(path)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := archive.Create(name)
	if err != nil {
		return err
	}

	if _, err := io.Copy(destination, source); err != nil {
		return err
	}

	return nil
}

func hasUsablePDFText(path string) (bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf(
			"failed to read extracted PDF text: %w",
			err,
		)
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
	name := strings.TrimSuffix(
		filepath.Base(path),
		filepath.Ext(path),
	)

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
