package dependencies

import "testing"

func TestFirstOutputLine(
	t *testing.T,
) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "unix",

			input: "ffmpeg version 8.0\nbuilt with gcc",

			expected: "ffmpeg version 8.0",
		},
		{
			name: "windows",

			input: "qpdf version 12.0\r\nCopyright",

			expected: "qpdf version 12.0",
		},
		{
			name: "leading empty lines",

			input: "\n\n  Tesseract 5.5.0  \n",

			expected: "Tesseract 5.5.0",
		},
		{
			name:     "empty",
			input:    "",
			expected: "",
		},
	}

	for _, test := range tests {
		t.Run(
			test.name,
			func(t *testing.T) {
				actual :=
					firstOutputLine(
						test.input,
					)

				if actual !=
					test.expected {
					t.Fatalf(
						"expected %q, got %q",
						test.expected,
						actual,
					)
				}
			},
		)
	}
}

func TestGhostscriptCandidates(
	t *testing.T,
) {
	candidates :=
		ghostscriptCandidates()

	if len(candidates) == 0 {
		t.Fatal(
			"expected Ghostscript candidates",
		)
	}
}

func TestBrowserExecutableNames(
	t *testing.T,
) {
	candidates :=
		browserExecutableNames()

	if len(candidates) == 0 {
		t.Fatal(
			"expected browser candidates",
		)
	}
}
