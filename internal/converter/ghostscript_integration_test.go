package converter

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGhostscriptCompressionWithTransparency(
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

	input :=
		writePDFTestDocument(
			t,
			pdfTestDocumentWithTransparency(),
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
			},
		)
	}
}

func pdfTestDocumentWithTransparency() []string {
	content :=
		"q\n" +
			"/GS1 gs\n" +
			"1 0 0 rg\n" +
			"20 20 100 100 re\n" +
			"f\n" +
			"Q\n" +
			"BT\n" +
			"/F1 18 Tf\n" +
			"20 150 Td\n" +
			"(Transparency test) Tj\n" +
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
			"/ExtGState << /GS1 5 0 R >> " +
			"/Font << /F1 6 0 R >> " +
			">> " +
			"/Contents 4 0 R >>",

		pdfTestStream(
			content,
		),

		"<< /Type /ExtGState " +
			"/ca 0.5 " +
			"/CA 0.5 >>",

		"<< /Type /Font " +
			"/Subtype /Type1 " +
			"/BaseFont /Helvetica >>",
	}
}

func pdfTestStream(
	content string,
) string {
	return "<< /Length " +
		intToPDFTestString(
			len(
				[]byte(
					content,
				),
			),
		) +
		" >>\nstream\n" +
		content +
		"endstream"
}

func intToPDFTestString(
	value int,
) string {
	if value == 0 {
		return "0"
	}

	var digits [20]byte

	position :=
		len(
			digits,
		)

	for value > 0 {
		position--

		digits[position] =
			byte(
				'0' +
					value%10,
			)

		value /= 10
	}

	return string(
		digits[position:],
	)
}
