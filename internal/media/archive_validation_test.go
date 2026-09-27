package media

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateDocumentArchive(
	t *testing.T,
) {
	path := writeArchiveTestFile(
		t,
		map[string][]byte{
			"[Content_Types].xml": []byte(
				"<Types></Types>",
			),
			"word/document.xml": []byte(
				"<document>Hello</document>",
			),
		},
	)

	if err := ValidateDocumentArchive(
		path,
	); err != nil {
		t.Fatalf(
			"validate document archive: %v",
			err,
		)
	}
}

func TestValidateDocumentArchiveRejectsEmptyArchive(
	t *testing.T,
) {
	path := filepath.Join(
		t.TempDir(),
		"empty.docx",
	)

	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}

	writer := zip.NewWriter(file)

	if err := writer.Close(); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}

	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	if err := ValidateDocumentArchive(
		path,
	); err == nil {
		t.Fatal(
			"expected empty archive to be rejected",
		)
	}
}

func TestValidateDocumentArchiveRejectsInvalidZIP(
	t *testing.T,
) {
	path := filepath.Join(
		t.TempDir(),
		"broken.docx",
	)

	if err := os.WriteFile(
		path,
		[]byte(
			"this is not a zip archive",
		),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	if err := ValidateDocumentArchive(
		path,
	); err == nil {
		t.Fatal(
			"expected invalid ZIP to be rejected",
		)
	}
}

func TestValidateArchiveEntryRejectsOversizedEntry(
	t *testing.T,
) {
	file := &zip.File{
		FileHeader: zip.FileHeader{
			Name: "huge.bin",
		},
	}

	file.UncompressedSize64 =
		maxArchiveEntrySize + 1

	file.CompressedSize64 =
		file.UncompressedSize64

	if err := validateArchiveEntry(
		file,
	); err == nil {
		t.Fatal(
			"expected oversized entry to be rejected",
		)
	}
}

func TestValidateArchiveEntryRejectsCompressionBomb(
	t *testing.T,
) {
	file := &zip.File{
		FileHeader: zip.FileHeader{
			Name: "bomb.bin",
		},
	}

	file.UncompressedSize64 =
		101 * 1024 * 1024

	file.CompressedSize64 =
		1 * 1024 * 1024

	if err := validateArchiveEntry(
		file,
	); err == nil {
		t.Fatal(
			"expected excessive compression ratio to be rejected",
		)
	}
}

func TestValidateArchiveEntryAllowsEmptyEntry(
	t *testing.T,
) {
	file := &zip.File{
		FileHeader: zip.FileHeader{
			Name: "empty.xml",
		},
	}

	if err := validateArchiveEntry(
		file,
	); err != nil {
		t.Fatalf(
			"empty entry rejected: %v",
			err,
		)
	}
}

func writeArchiveTestFile(
	t *testing.T,
	files map[string][]byte,
) string {
	t.Helper()

	path := filepath.Join(
		t.TempDir(),
		"document.docx",
	)

	output, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}

	writer := zip.NewWriter(output)

	for name, content := range files {
		entry, err := writer.Create(
			name,
		)
		if err != nil {
			_ = writer.Close()
			_ = output.Close()

			t.Fatal(err)
		}

		if _, err := bytes.NewReader(
			content,
		).WriteTo(entry); err != nil {
			_ = writer.Close()
			_ = output.Close()

			t.Fatal(err)
		}
	}

	if err := writer.Close(); err != nil {
		_ = output.Close()

		t.Fatal(err)
	}

	if err := output.Close(); err != nil {
		t.Fatal(err)
	}

	return path
}
