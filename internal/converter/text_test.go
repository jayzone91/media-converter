package converter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTextToHTML(
	t *testing.T,
) {
	directory := t.TempDir()

	inputPath := filepath.Join(
		directory,
		"test.txt",
	)

	outputPath := filepath.Join(
		directory,
		"test.html",
	)

	if err := os.WriteFile(
		inputPath,
		[]byte(
			"Hallo <script>alert('test')</script>\n& Welt",
		),
		0o600,
	); err != nil {
		t.Fatalf(
			"write input: %v",
			err,
		)
	}

	if err := TextToHTML(
		inputPath,
		outputPath,
	); err != nil {
		t.Fatalf(
			"convert text to HTML: %v",
			err,
		)
	}

	data, err := os.ReadFile(
		outputPath,
	)
	if err != nil {
		t.Fatalf(
			"read HTML output: %v",
			err,
		)
	}

	document := string(data)

	if strings.Contains(
		document,
		"<script>alert('test')</script>",
	) {
		t.Fatal(
			"HTML output contains unescaped script element",
		)
	}

	if !strings.Contains(
		document,
		"&lt;script&gt;alert(&#39;test&#39;)&lt;/script&gt;",
	) {
		t.Fatal(
			"HTML output does not contain escaped input",
		)
	}

	if !strings.Contains(
		document,
		"&amp; Welt",
	) {
		t.Fatal(
			"HTML output does not escape ampersand",
		)
	}
}
