package media

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectSignature(
	t *testing.T,
) {
	tests := []struct {
		name     string
		filename string
		data     []byte
		expected string
	}{
		{
			name:     "PNG",
			filename: "image.png",
			data: []byte{
				0x89,
				0x50,
				0x4e,
				0x47,
				0x0d,
				0x0a,
				0x1a,
				0x0a,
				0x00,
			},
			expected: "png",
		},
		{
			name:     "JPEG",
			filename: "image.jpg",
			data: []byte{
				0xff,
				0xd8,
				0xff,
				0xe0,
			},
			expected: "jpeg",
		},
		{
			name:     "WebP",
			filename: "image.webp",
			data: []byte(
				"RIFF1234WEBPVP8 ",
			),
			expected: "webp",
		},
		{
			name:     "GIF87a",
			filename: "image.gif",
			data: []byte(
				"GIF87a",
			),
			expected: "gif",
		},
		{
			name:     "GIF89a",
			filename: "image.gif",
			data: []byte(
				"GIF89a",
			),
			expected: "gif",
		},
		{
			name:     "BMP",
			filename: "image.bmp",
			data: []byte{
				0x42,
				0x4d,
				0x00,
				0x00,
			},
			expected: "bmp",
		},
		{
			name:     "TIFF little endian",
			filename: "image.tiff",
			data: []byte{
				0x49,
				0x49,
				0x2a,
				0x00,
			},
			expected: "tiff",
		},
		{
			name:     "TIFF big endian",
			filename: "image.tif",
			data: []byte{
				0x4d,
				0x4d,
				0x00,
				0x2a,
			},
			expected: "tiff",
		},
		{
			name:     "PDF",
			filename: "document.pdf",
			data: []byte(
				"%PDF-1.7\n",
			),
			expected: "pdf",
		},
		{
			name:     "SVG",
			filename: "image.svg",
			data: []byte(
				`<svg xmlns="http://www.w3.org/2000/svg"></svg>`,
			),
			expected: "svg",
		},
		{
			name:     "SVG XML",
			filename: "image.svg",
			data: []byte(
				`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg"></svg>`,
			),
			expected: "svg",
		},
		{
			name:     "AVIF",
			filename: "image.avif",
			data: []byte{
				0x00,
				0x00,
				0x00,
				0x18,
				'f',
				't',
				'y',
				'p',
				'a',
				'v',
				'i',
				'f',
				0x00,
				0x00,
				0x00,
				0x00,
				'a',
				'v',
				'i',
				'f',
			},
			expected: "avif",
		},
		{
			name:     "HEIC",
			filename: "image.heic",
			data: []byte{
				0x00,
				0x00,
				0x00,
				0x18,
				'f',
				't',
				'y',
				'p',
				'h',
				'e',
				'i',
				'c',
				0x00,
				0x00,
				0x00,
				0x00,
				'h',
				'e',
				'i',
				'c',
			},
			expected: "heic",
		},
	}

	for _, test := range tests {
		t.Run(
			test.name,
			func(t *testing.T) {
				path := writeSignatureTestFile(
					t,
					test.filename,
					test.data,
				)

				format, err :=
					DetectSignature(path)

				if err != nil {
					t.Fatalf(
						"detect signature: %v",
						err,
					)
				}

				if format.ID != test.expected {
					t.Fatalf(
						"expected %s, got %s",
						test.expected,
						format.ID,
					)
				}
			},
		)
	}
}

func TestDetectSignatureRejectsUnknownData(
	t *testing.T,
) {
	path := writeSignatureTestFile(
		t,
		"unknown.bin",
		[]byte{
			0x00,
			0x01,
			0x02,
			0x03,
		},
	)

	if _, err :=
		DetectSignature(path); err == nil {
		t.Fatal(
			"expected unknown data to be rejected",
		)
	}
}

func TestDetectSignatureRejectsFakePNG(
	t *testing.T,
) {
	path := writeSignatureTestFile(
		t,
		"fake.png",
		[]byte(
			"this is not a png",
		),
	)

	if _, err :=
		DetectSignature(path); err == nil {
		t.Fatal(
			"expected fake PNG to be rejected",
		)
	}
}

func writeSignatureTestFile(
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
