package converter

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGhostscriptCompressionWithEmbeddedFont(
	t *testing.T,
) {
	t.Parallel()

	ghostscript, err :=
		NewGhostscript()

	if err != nil {
		t.Skipf(
			"Ghostscript unavailable: %v",
			err,
		)
	}

	qpdf, err :=
		NewQPDF()

	if err != nil {
		t.Skipf(
			"qpdf unavailable: %v",
			err,
		)
	}

	font :=
		loadTestTrueTypeFont(
			t,
		)

	input :=
		writePDFTestDocument(
			t,
			pdfTestDocumentWithEmbeddedTrueTypeFont(
				font,
			),
		)

	ctx, cancel :=
		context.WithTimeout(
			context.Background(),
			30*time.Second,
		)

	defer cancel()

	if err :=
		qpdf.CheckPDF(
			ctx,
			input,
		); err != nil {
		t.Fatalf(
			"embedded-font input PDF is invalid: %v",
			err,
		)
	}

	requireEmbeddedPDFFont(
		t,
		ctx,
		qpdf,
		input,
	)

	for _, preset := range []PDFCompressionPreset{
		PDFCompressionBalanced,
		PDFCompressionStrong,
	} {
		preset :=
			preset

		t.Run(
			string(
				preset,
			),
			func(
				t *testing.T,
			) {
				t.Parallel()

				output :=
					filepath.Join(
						t.TempDir(),
						"compressed.pdf",
					)

				ctx, cancel :=
					context.WithTimeout(
						context.Background(),
						30*time.Second,
					)

				defer cancel()

				if err :=
					ghostscript.CompressPDF(
						ctx,
						input,
						output,
						preset,
					); err != nil {
					t.Fatalf(
						"Ghostscript compression failed: %v",
						err,
					)
				}

				info, err :=
					os.Stat(
						output,
					)

				if err != nil {
					t.Fatalf(
						"compressed PDF missing: %v",
						err,
					)
				}

				if info.Size() <= 0 {
					t.Fatal(
						"compressed PDF is empty",
					)
				}

				if err :=
					qpdf.CheckPDF(
						ctx,
						output,
					); err != nil {
					t.Fatalf(
						"compressed PDF failed qpdf validation: %v",
						err,
					)
				}

				pageCount, err :=
					qpdf.PageCount(
						ctx,
						output,
					)

				if err != nil {
					t.Fatalf(
						"read compressed PDF page count: %v",
						err,
					)
				}

				if pageCount.Count != 1 {
					t.Fatalf(
						"expected one page, got %d",
						pageCount.Count,
					)
				}

				requireEmbeddedPDFFont(
					t,
					ctx,
					qpdf,
					output,
				)
			},
		)
	}
}

func loadTestTrueTypeFont(
	t *testing.T,
) []byte {
	t.Helper()

	if override :=
		os.Getenv(
			"MEDIA_CONVERTER_TEST_TTF",
		); override != "" {
		data, err :=
			os.ReadFile(
				override,
			)

		if err != nil {
			t.Fatalf(
				"read MEDIA_CONVERTER_TEST_TTF %q: %v",
				override,
				err,
			)
		}

		return data
	}

	candidates :=
		[]string{
			`C:\Windows\Fonts\arial.ttf`,
			`C:\Windows\Fonts\calibri.ttf`,
			`C:\Windows\Fonts\segoeui.ttf`,

			"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
			"/usr/share/fonts/truetype/liberation2/LiberationSans-Regular.ttf",
			"/usr/share/fonts/truetype/freefont/FreeSans.ttf",

			"/Library/Fonts/Arial.ttf",
			"/System/Library/Fonts/Supplemental/Arial.ttf",
		}

	for _, candidate := range candidates {
		data, err :=
			os.ReadFile(
				candidate,
			)

		if err == nil &&
			len(data) > 0 {
			return data
		}
	}

	t.Skip(
		"no suitable TrueType font found; " +
			"set MEDIA_CONVERTER_TEST_TTF to a .ttf file",
	)

	return nil
}

func pdfTestDocumentWithEmbeddedTrueTypeFont(
	font []byte,
) []string {
	content :=
		"BT\n" +
			"/F1 18 Tf\n" +
			"20 100 Td\n" +
			"(A) Tj\n" +
			"ET\n"

	return []string{
		"<< /Type /Catalog /Pages 2 0 R >>",

		"<< /Type /Pages " +
			"/Kids [3 0 R] " +
			"/Count 1 >>",

		"<< /Type /Page " +
			"/Parent 2 0 R " +
			"/MediaBox [0 0 200 200] " +
			"/Resources << " +
			"/Font << /F1 5 0 R >> " +
			">> " +
			"/Contents 4 0 R >>",

		pdfTestStream(
			content,
		),

		"<< /Type /Font " +
			"/Subtype /TrueType " +
			"/BaseFont /EmbeddedTestFont " +
			"/FirstChar 65 " +
			"/LastChar 65 " +
			"/Widths [600] " +
			"/FontDescriptor 6 0 R " +
			"/Encoding /WinAnsiEncoding >>",

		"<< /Type /FontDescriptor " +
			"/FontName /EmbeddedTestFont " +
			"/Flags 32 " +
			"/FontBBox [-1000 -1000 3000 3000] " +
			"/ItalicAngle 0 " +
			"/Ascent 1000 " +
			"/Descent -300 " +
			"/CapHeight 700 " +
			"/StemV 80 " +
			"/FontFile2 7 0 R >>",

		pdfTestBinaryStream(
			font,
		),
	}
}

func pdfTestBinaryStream(
	content []byte,
) string {
	return "<< /Length " +
		intToPDFTestString(
			len(
				content,
			),
		) +
		" /Length1 " +
		intToPDFTestString(
			len(
				content,
			),
		) +
		" >>\nstream\n" +
		string(
			content,
		) +
		"\nendstream"
}

func requireEmbeddedPDFFont(
	t *testing.T,
	ctx context.Context,
	qpdf *QPDF,
	input string,
) {
	t.Helper()

	expanded :=
		filepath.Join(
			t.TempDir(),
			"expanded.pdf",
		)

	command :=
		externalCommandContext(
			ctx,
			qpdf.binary,
			"--warning-exit-0",
			"--qdf",
			"--object-streams=disable",
			input,
			expanded,
		)

	output, err :=
		command.CombinedOutput()

	if err != nil {
		t.Fatalf(
			"expand PDF for font inspection: %v: %s",
			err,
			string(
				output,
			),
		)
	}

	data, err :=
		os.ReadFile(
			expanded,
		)

	if err != nil {
		t.Fatalf(
			"read expanded PDF: %v",
			err,
		)
	}

	if bytes.Contains(
		data,
		[]byte(
			"/FontFile ",
		),
	) {
		return
	}

	if bytes.Contains(
		data,
		[]byte(
			"/FontFile2 ",
		),
	) {
		return
	}

	if bytes.Contains(
		data,
		[]byte(
			"/FontFile3 ",
		),
	) {
		return
	}

	t.Fatal(
		"expected compressed PDF to retain an embedded font",
	)
}
