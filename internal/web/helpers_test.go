package web

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDetectFormatRTF(
	t *testing.T,
) {
	path := writeDetectionTestFile(
		t,
		"document.rtf",
		[]byte(
			`{\rtf1\ansi Test}`,
		),
	)

	format, err := detectFormat(
		context.Background(),
		path,
		nil,
	)
	if err != nil {
		t.Fatalf(
			"detect RTF: %v",
			err,
		)
	}

	if format.ID != "rtf" {
		t.Fatalf(
			"expected rtf, got %s",
			format.ID,
		)
	}
}

func TestDetectFormatTXT(
	t *testing.T,
) {
	path := writeDetectionTestFile(
		t,
		"document.txt",
		[]byte(
			"Dies ist eine normale Textdatei.",
		),
	)

	format, err := detectFormat(
		context.Background(),
		path,
		nil,
	)
	if err != nil {
		t.Fatalf(
			"detect TXT: %v",
			err,
		)
	}

	if format.ID != "txt" {
		t.Fatalf(
			"expected txt, got %s",
			format.ID,
		)
	}
}

func TestDetectFormatMarkdown(
	t *testing.T,
) {
	extensions := []string{
		".md",
		".mdx",
		".markdown",
	}

	for _, extension := range extensions {
		t.Run(
			extension,
			func(t *testing.T) {
				path := writeDetectionTestFile(
					t,
					"document"+extension,
					[]byte(
						"# Überschrift\n\nMarkdown-Inhalt.",
					),
				)

				format, err := detectFormat(
					context.Background(),
					path,
					nil,
				)
				if err != nil {
					t.Fatalf(
						"detect Markdown: %v",
						err,
					)
				}

				if format.ID != "markdown" {
					t.Fatalf(
						"expected markdown, got %s",
						format.ID,
					)
				}
			},
		)
	}
}

func TestDetectFormatRejectsBinaryMarkdown(
	t *testing.T,
) {
	path := writeDetectionTestFile(
		t,
		"document.md",
		[]byte{
			0x00,
			0x01,
			0x02,
			0x03,
			0xff,
			0xfe,
			0xfd,
		},
	)

	if _, err := detectFormat(
		context.Background(),
		path,
		nil,
	); err == nil {
		t.Fatal(
			"expected binary Markdown to be rejected",
		)
	}
}

func TestDetectFormatRejectsRTFWithTXTFileExtension(
	t *testing.T,
) {
	path := writeDetectionTestFile(
		t,
		"document.txt",
		[]byte(
			`{\rtf1\ansi Test}`,
		),
	)

	if _, err := detectFormat(
		context.Background(),
		path,
		nil,
	); err == nil {
		t.Fatal(
			"expected RTF with .txt extension to be rejected",
		)
	}
}

func TestDetectFormatRejectsTextWithImageExtension(
	t *testing.T,
) {
	path := writeDetectionTestFile(
		t,
		"document.png",
		[]byte(
			"Das ist keine PNG-Datei.",
		),
	)

	if _, err := detectFormat(
		context.Background(),
		path,
		nil,
	); err == nil {
		t.Fatal(
			"expected plain text with .png extension to be rejected",
		)
	}
}

func TestDetectFormatRejectsTXTWithMarkdownExtensionMismatch(
	t *testing.T,
) {
	path := writeDetectionTestFile(
		t,
		"document.txt",
		[]byte(
			"# Das ist zwar Markdown-Syntax, aber eine TXT-Datei.",
		),
	)

	format, err := detectFormat(
		context.Background(),
		path,
		nil,
	)
	if err != nil {
		t.Fatalf(
			"detect TXT: %v",
			err,
		)
	}

	if format.ID != "txt" {
		t.Fatalf(
			"expected txt, got %s",
			format.ID,
		)
	}
}

func writeDetectionTestFile(
	t *testing.T,
	filename string,
	content []byte,
) string {
	t.Helper()

	path := filepath.Join(
		t.TempDir(),
		filename,
	)

	if err := os.WriteFile(
		path,
		content,
		0o600,
	); err != nil {
		t.Fatalf(
			"write test file: %v",
			err,
		)
	}

	return path
}
