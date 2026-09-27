package web

import (
	"context"
	"errors"
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

	_, err := detectFormat(
		context.Background(),
		path,
		nil,
	)

	if err == nil {
		t.Fatal(
			"expected binary Markdown to be rejected",
		)
	}

	if !errors.Is(
		err,
		errContentMismatch,
	) {
		t.Fatalf(
			"expected content mismatch, got %v",
			err,
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

	_, err := detectFormat(
		context.Background(),
		path,
		nil,
	)

	if err == nil {
		t.Fatal(
			"expected RTF with .txt extension to be rejected",
		)
	}

	if !errors.Is(
		err,
		errFormatMismatch,
	) {
		t.Fatalf(
			"expected format mismatch, got %v",
			err,
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

	_, err := detectFormat(
		context.Background(),
		path,
		nil,
	)

	if err == nil {
		t.Fatal(
			"expected plain text with .png extension to be rejected",
		)
	}

	if !errors.Is(
		err,
		errContentMismatch,
	) {
		t.Fatalf(
			"expected content mismatch, got %v",
			err,
		)
	}
}

func TestDetectFormatRejectsUnsupportedExtension(
	t *testing.T,
) {
	path := writeDetectionTestFile(
		t,
		"document.xyz",
		[]byte(
			"Test",
		),
	)

	_, err := detectFormat(
		context.Background(),
		path,
		nil,
	)

	if err == nil {
		t.Fatal(
			"expected unsupported extension to be rejected",
		)
	}

	if !errors.Is(
		err,
		errUnsupportedExtension,
	) {
		t.Fatalf(
			"expected unsupported extension, got %v",
			err,
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

func TestDetectionErrorMessageUnsupportedExtension(
	t *testing.T,
) {
	message := detectionErrorMessage(
		"document.xyz",
		errUnsupportedExtension,
	)

	expected :=
		`Dateien mit der Endung ".xyz" werden nicht unterstützt.`

	if message != expected {
		t.Fatalf(
			"expected %q, got %q",
			expected,
			message,
		)
	}
}

func TestDetectionErrorMessageMissingExtension(
	t *testing.T,
) {
	message := detectionErrorMessage(
		"document",
		errUnsupportedExtension,
	)

	expected :=
		`Die Datei "document" hat keine unterstützte Dateiendung.`

	if message != expected {
		t.Fatalf(
			"expected %q, got %q",
			expected,
			message,
		)
	}
}

func TestDetectionErrorMessageFormatMismatch(
	t *testing.T,
) {
	message := detectionErrorMessage(
		"document.txt",
		errFormatMismatch,
	)

	expected :=
		`Dateiendung und tatsächlicher Dateityp von "document.txt" stimmen nicht überein.`

	if message != expected {
		t.Fatalf(
			"expected %q, got %q",
			expected,
			message,
		)
	}
}

func TestDetectionErrorMessageContentMismatch(
	t *testing.T,
) {
	message := detectionErrorMessage(
		"document.png",
		errContentMismatch,
	)

	expected :=
		`Der Inhalt von "document.png" entspricht nicht dem angegebenen Dateityp.`

	if message != expected {
		t.Fatalf(
			"expected %q, got %q",
			expected,
			message,
		)
	}
}

func TestDetectionErrorMessageDetectionFailed(
	t *testing.T,
) {
	message := detectionErrorMessage(
		"document.mp4",
		errDetectionFailed,
	)

	expected :=
		`Der Dateityp von "document.mp4" konnte nicht erkannt werden oder wird nicht unterstützt.`

	if message != expected {
		t.Fatalf(
			"expected %q, got %q",
			expected,
			message,
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
