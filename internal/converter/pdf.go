package converter

import (
	"archive/zip"
	"bytes"
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
	tessdataDir string
	libreOffice *LibreOffice
}

func NewPDF(libreOffice *LibreOffice) (*PDF, error) {
	pdfToText, err := exec.LookPath("pdftotext")
	if err != nil {
		return nil, fmt.Errorf(
			"pdftotext not found: %w",
			err,
		)
	}

	pdfToPPM, err := exec.LookPath("pdftoppm")
	if err != nil {
		return nil, fmt.Errorf(
			"pdftoppm not found: %w",
			err,
		)
	}

	tesseract, err := exec.LookPath("tesseract")
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

func (c *PDF) ConvertToImages(
	ctx context.Context,
	input string,
	output string,
	format string,
) error {
	switch format {
	case "png", "jpeg":
	default:
		return fmt.Errorf(
			"unsupported PDF image format: %s",
			format,
		)
	}

	tempDir, err := os.MkdirTemp(
		filepath.Dir(input),
		"pdf-pages-*",
	)
	if err != nil {
		return fmt.Errorf(
			"failed to create PDF image temp dir: %w",
			err,
		)
	}
	defer os.RemoveAll(tempDir)

	prefix := filepath.Join(
		tempDir,
		"page",
	)

	args := []string{
		"-r",
		"200",
	}

	switch format {
	case "png":
		args = append(
			args,
			"-png",
		)

	case "jpeg":
		args = append(
			args,
			"-jpeg",
			"-jpegopt",
			"quality=90",
		)
	}

	args = append(
		args,
		input,
		prefix,
	)

	cmd := exec.CommandContext(
		ctx,
		c.pdfToPPM,
		args...,
	)

	if result, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf(
			"pdftoppm failed: %w: %s",
			err,
			string(result),
		)
	}

	extension := "." + format

	if format == "jpeg" {
		extension = ".jpg"
	}

	pages, err := filepath.Glob(
		prefix + "-*" + extension,
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

	sort.Slice(
		pages,
		func(i, j int) bool {
			return pdfPageNumber(pages[i]) <
				pdfPageNumber(pages[j])
		},
	)

	if err := createImageZIP(
		output,
		pages,
		extension,
	); err != nil {
		return fmt.Errorf(
			"failed to create PDF image archive: %w",
			err,
		)
	}

	return nil
}

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

	if err := writePDFTextPages(
		output,
		result,
	); err != nil {
		return err
	}

	return nil
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
		return fmt.Errorf(
			"pdftoppm failed: %w: %s",
			err,
			string(result),
		)
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

	sort.Slice(
		pages,
		func(i, j int) bool {
			return pdfPageNumber(pages[i]) <
				pdfPageNumber(pages[j])
		},
	)

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

	cmd := exec.CommandContext(
		ctx,
		c.pdfToPPM,
		"-f",
		strconv.Itoa(pageNumber),
		"-l",
		strconv.Itoa(pageNumber),
		"-singlefile",
		"-png",
		"-r",
		"300",
		input,
		prefix,
	)

	if result, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf(
			"pdftoppm failed: %w: %s",
			err,
			string(result),
		)
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
	cmd := exec.CommandContext(
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

	var stderr bytes.Buffer

	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf(
			"tesseract failed: %w: %s",
			err,
			strings.TrimSpace(
				stderr.String(),
			),
		)
	}

	return string(output), nil
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

			if characters >=
				minimumPDFTextCharacters {
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
			/*
				Form feed keeps a logical page boundary.
				LibreOffice generally converts this more
				cleanly than joining pages with blank lines.
			*/
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

func createImageZIP(
	output string,
	pages []string,
	extension string,
) error {
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

func addFileToZIP(
	archive *zip.Writer,
	path string,
	name string,
) error {
	source, err := os.Open(path)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := archive.Create(
		name,
	)
	if err != nil {
		return err
	}

	if _, err := io.Copy(
		destination,
		source,
	); err != nil {
		return err
	}

	return nil
}

func pdfPageNumber(
	path string,
) int {
	name := strings.TrimSuffix(
		filepath.Base(path),
		filepath.Ext(path),
	)

	index := strings.LastIndex(
		name,
		"-",
	)
	if index == -1 {
		return 0
	}

	number, err := strconv.Atoi(
		name[index+1:],
	)
	if err != nil {
		return 0
	}

	return number
}

func findTessdataDirectory(
	tesseract string,
) (string, error) {
	candidates := []string{
		os.Getenv(
			"TESSDATA_PREFIX",
		),

		filepath.Join(
			filepath.Dir(tesseract),
			"tessdata",
		),

		"/usr/share/tesseract-ocr/5/tessdata",
		"/usr/share/tesseract-ocr/4.00/tessdata",
		"/usr/share/tessdata",
	}

	for _, candidate := range candidates {
		candidate = strings.TrimSpace(
			candidate,
		)

		if candidate == "" {
			continue
		}

		deu := filepath.Join(
			candidate,
			"deu.traineddata",
		)

		eng := filepath.Join(
			candidate,
			"eng.traineddata",
		)

		if _, err := os.Stat(deu); err != nil {
			continue
		}

		if _, err := os.Stat(eng); err != nil {
			continue
		}

		return candidate, nil
	}

	return "", fmt.Errorf(
		"tesseract language data not found: deu.traineddata and eng.traineddata are required",
	)
}
