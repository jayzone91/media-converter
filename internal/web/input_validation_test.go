package web

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateSVGFile(
	t *testing.T,
) {
	path := writeSVGTestFile(
		t,
		"valid.svg",
		`<?xml version="1.0"?>
<svg xmlns="http://www.w3.org/2000/svg">
	<rect width="10" height="10"/>
</svg>`,
	)

	if err := validateSVGFile(
		path,
	); err != nil {
		t.Fatalf(
			"validate SVG: %v",
			err,
		)
	}
}

func TestValidateSVGFileRejectsMalformedXML(
	t *testing.T,
) {
	path := writeSVGTestFile(
		t,
		"broken.svg",
		`<svg><rect></svg>`,
	)

	if err := validateSVGFile(
		path,
	); err == nil {
		t.Fatal(
			"expected malformed SVG to be rejected",
		)
	}
}

func TestValidateSVGFileRejectsWrongRoot(
	t *testing.T,
) {
	path := writeSVGTestFile(
		t,
		"wrong-root.svg",
		`<html></html>`,
	)

	if err := validateSVGFile(
		path,
	); err == nil {
		t.Fatal(
			"expected non-SVG root to be rejected",
		)
	}
}

func TestValidateSVGFileRejectsEmptyFile(
	t *testing.T,
) {
	path := writeSVGTestFile(
		t,
		"empty.svg",
		"",
	)

	if err := validateSVGFile(
		path,
	); err == nil {
		t.Fatal(
			"expected empty SVG to be rejected",
		)
	}
}

func writeSVGTestFile(
	t *testing.T,
	filename string,
	content string,
) string {
	t.Helper()

	path := filepath.Join(
		t.TempDir(),
		filename,
	)

	if err := os.WriteFile(
		path,
		[]byte(content),
		0o600,
	); err != nil {
		t.Fatalf(
			"write SVG test file: %v",
			err,
		)
	}

	return path
}
