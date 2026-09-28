package converter

import (
	"slices"
	"strings"
	"testing"
)

func TestGhostscriptCompressionArgsBalanced(t *testing.T) {
	t.Parallel()

	args, err :=
		ghostscriptCompressionArgs(
			"input.pdf",
			"output.pdf",
			PDFCompressionBalanced,
		)

	if err != nil {
		t.Fatalf(
			"ghostscriptCompressionArgs returned error: %v",
			err,
		)
	}

	requireGhostscriptArgument(
		t,
		args,
		"-dSAFER",
	)

	requireGhostscriptArgument(
		t,
		args,
		"-dBATCH",
	)

	requireGhostscriptArgument(
		t,
		args,
		"-dNOPAUSE",
	)

	requireGhostscriptArgument(
		t,
		args,
		"-sDEVICE=pdfwrite",
	)

	requireGhostscriptArgument(
		t,
		args,
		"-dCompatibilityLevel=1.7",
	)

	requireGhostscriptArgument(
		t,
		args,
		"-dCompressFonts=true",
	)

	requireGhostscriptArgument(
		t,
		args,
		"-dSubsetFonts=true",
	)

	requireGhostscriptArgument(
		t,
		args,
		"-dDownsampleColorImages=true",
	)

	requireGhostscriptArgument(
		t,
		args,
		"-dColorImageResolution=150",
	)

	requireGhostscriptArgument(
		t,
		args,
		"-dJPEGQ=82",
	)

	requireGhostscriptArgument(
		t,
		args,
		"-dGrayImageResolution=150",
	)

	requireGhostscriptArgument(
		t,
		args,
		"-dMonoImageResolution=300",
	)

	requireGhostscriptArgument(
		t,
		args,
		"-sOutputFile=output.pdf",
	)

	if args[len(args)-1] !=
		"input.pdf" {
		t.Fatalf(
			"expected input PDF as final argument, got %q",
			args[len(args)-1],
		)
	}

	requireGhostscriptArgumentAbsent(
		t,
		args,
		"-dNoOutputFonts",
	)

	requireGhostscriptArgumentAbsentPrefix(
		t,
		args,
		"-dCompatibilityLevel=1.3",
	)
}

func TestGhostscriptCompressionArgsStrong(t *testing.T) {
	t.Parallel()

	args, err :=
		ghostscriptCompressionArgs(
			"input.pdf",
			"output.pdf",
			PDFCompressionStrong,
		)

	if err != nil {
		t.Fatalf(
			"ghostscriptCompressionArgs returned error: %v",
			err,
		)
	}

	requireGhostscriptArgument(
		t,
		args,
		"-dCompatibilityLevel=1.7",
	)

	requireGhostscriptArgument(
		t,
		args,
		"-dCompressFonts=true",
	)

	requireGhostscriptArgument(
		t,
		args,
		"-dSubsetFonts=true",
	)

	requireGhostscriptArgument(
		t,
		args,
		"-dColorImageResolution=96",
	)

	requireGhostscriptArgument(
		t,
		args,
		"-dJPEGQ=65",
	)

	requireGhostscriptArgument(
		t,
		args,
		"-dGrayImageResolution=96",
	)

	requireGhostscriptArgument(
		t,
		args,
		"-dMonoImageResolution=200",
	)

	requireGhostscriptArgumentAbsent(
		t,
		args,
		"-dNoOutputFonts",
	)
}

func TestGhostscriptCompressionArgsRejectsUnknownPreset(
	t *testing.T,
) {
	t.Parallel()

	_, err :=
		ghostscriptCompressionArgs(
			"input.pdf",
			"output.pdf",
			PDFCompressionPreset(
				"unknown",
			),
		)

	if err == nil {
		t.Fatal(
			"expected unsupported preset error",
		)
	}
}

func TestGhostscriptCompressionPresetsDiffer(t *testing.T) {
	t.Parallel()

	balanced, err :=
		ghostscriptCompressionArgs(
			"input.pdf",
			"output.pdf",
			PDFCompressionBalanced,
		)

	if err != nil {
		t.Fatalf(
			"balanced preset failed: %v",
			err,
		)
	}

	strong, err :=
		ghostscriptCompressionArgs(
			"input.pdf",
			"output.pdf",
			PDFCompressionStrong,
		)

	if err != nil {
		t.Fatalf(
			"strong preset failed: %v",
			err,
		)
	}

	if slices.Equal(
		balanced,
		strong,
	) {
		t.Fatal(
			"balanced and strong presets must not use identical arguments",
		)
	}
}

func requireGhostscriptArgument(
	t *testing.T,
	args []string,
	expected string,
) {
	t.Helper()

	if slices.Contains(
		args,
		expected,
	) {
		return
	}

	t.Fatalf(
		"missing Ghostscript argument %q\narguments: %v",
		expected,
		args,
	)
}

func requireGhostscriptArgumentAbsent(
	t *testing.T,
	args []string,
	forbidden string,
) {
	t.Helper()

	if !slices.Contains(
		args,
		forbidden,
	) {
		return
	}

	t.Fatalf(
		"unexpected Ghostscript argument %q\narguments: %v",
		forbidden,
		args,
	)
}

func requireGhostscriptArgumentAbsentPrefix(
	t *testing.T,
	args []string,
	prefix string,
) {
	t.Helper()

	for _, argument := range args {
		if strings.HasPrefix(
			argument,
			prefix,
		) {
			t.Fatalf(
				"unexpected Ghostscript argument %q",
				argument,
			)
		}
	}
}
