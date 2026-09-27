package qr

import (
	"os"
	"testing"
)

func TestRenderSVG(t *testing.T) {
	style := DefaultStyle()

	matrix, err := Generate(
		"https://example.com",
		style,
	)
	if err != nil {
		t.Fatal(err)
	}

	output, err := RenderSVG(
		matrix,
		style,
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		"test.svg",
		output,
		0600,
	); err != nil {
		t.Fatal(err)
	}
}
