package converter

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestQPDFOptimizeLinearizedLargePDF(
	t *testing.T,
) {
	t.Parallel()

	qpdf, err :=
		NewQPDF()

	if err != nil {
		t.Skipf(
			"qpdf unavailable: %v",
			err,
		)
	}

	const pageCount = 40

	input :=
		writePDFTestDocument(
			t,
			pdfTestDocumentWithPages(
				pageCount,
			),
		)

	output :=
		filepath.Join(
			t.TempDir(),
			"linearized.pdf",
		)

	ctx, cancel :=
		context.WithTimeout(
			context.Background(),
			30*time.Second,
		)

	defer cancel()

	if err :=
		qpdf.OptimizeLinearized(
			ctx,
			input,
			output,
		); err != nil {
		t.Fatalf(
			"OptimizeLinearized failed: %v",
			err,
		)
	}

	if err :=
		qpdf.CheckPDF(
			ctx,
			output,
		); err != nil {
		t.Fatalf(
			"linearized PDF failed structural validation: %v",
			err,
		)
	}

	if err :=
		qpdf.CheckLinearization(
			ctx,
			output,
		); err != nil {
		t.Fatalf(
			"linearized PDF failed linearization check: %v",
			err,
		)
	}

	result, err :=
		qpdf.PageCount(
			ctx,
			output,
		)

	if err != nil {
		t.Fatalf(
			"read linearized PDF page count: %v",
			err,
		)
	}

	if result.Count !=
		pageCount {
		t.Fatalf(
			"expected %d pages, got %d",
			pageCount,
			result.Count,
		)
	}
}

func TestQPDFOptimizeAlreadyOptimizedPDF(
	t *testing.T,
) {
	t.Parallel()

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
			pdfTestDocumentWithPages(
				5,
			),
		)

	first :=
		filepath.Join(
			t.TempDir(),
			"optimized-first.pdf",
		)

	second :=
		filepath.Join(
			t.TempDir(),
			"optimized-second.pdf",
		)

	ctx, cancel :=
		context.WithTimeout(
			context.Background(),
			30*time.Second,
		)

	defer cancel()

	if err :=
		qpdf.Optimize(
			ctx,
			input,
			first,
		); err != nil {
		t.Fatalf(
			"first optimization failed: %v",
			err,
		)
	}

	if err :=
		qpdf.Optimize(
			ctx,
			first,
			second,
		); err != nil {
		t.Fatalf(
			"second optimization failed: %v",
			err,
		)
	}

	for _, path := range []string{
		first,
		second,
	} {
		if err :=
			qpdf.CheckPDF(
				ctx,
				path,
			); err != nil {
			t.Fatalf(
				"optimized PDF %q failed validation: %v",
				path,
				err,
			)
		}

		result, err :=
			qpdf.PageCount(
				ctx,
				path,
			)

		if err != nil {
			t.Fatalf(
				"read optimized PDF page count: %v",
				err,
			)
		}

		if result.Count != 5 {
			t.Fatalf(
				"expected 5 pages, got %d",
				result.Count,
			)
		}
	}
}

func TestQPDFOptimizePreservesAcroForm(
	t *testing.T,
) {
	t.Parallel()

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
			pdfTestDocumentWithForm(),
		)

	output :=
		filepath.Join(
			t.TempDir(),
			"optimized-form.pdf",
		)

	ctx, cancel :=
		context.WithTimeout(
			context.Background(),
			30*time.Second,
		)

	defer cancel()

	hasForm, err :=
		HasPDFFormFields(
			input,
		)

	if err != nil {
		t.Fatalf(
			"inspect input form: %v",
			err,
		)
	}

	if !hasForm {
		t.Fatal(
			"input fixture must contain an AcroForm",
		)
	}

	if err :=
		qpdf.Optimize(
			ctx,
			input,
			output,
		); err != nil {
		t.Fatalf(
			"optimization failed: %v",
			err,
		)
	}

	if err :=
		qpdf.CheckPDF(
			ctx,
			output,
		); err != nil {
		t.Fatalf(
			"optimized form PDF failed validation: %v",
			err,
		)
	}

	hasForm, err =
		HasPDFFormFields(
			output,
		)

	if err != nil {
		t.Fatalf(
			"inspect optimized form: %v",
			err,
		)
	}

	if !hasForm {
		t.Fatal(
			"AcroForm was lost during optimization",
		)
	}
}

func pdfTestDocumentWithPages(
	pageCount int,
) []string {
	if pageCount < 1 {
		panic(
			"pageCount must be positive",
		)
	}

	objects :=
		make(
			[]string,
			2+
				pageCount*2,
		)

	objects[0] =
		"<< /Type /Catalog /Pages 2 0 R >>"

	var kids strings.Builder

	for pageIndex := 0; pageIndex <
		pageCount; pageIndex++ {
		pageObject :=
			3 +
				pageIndex

		if pageIndex > 0 {
			kids.WriteByte(
				' ',
			)
		}

		_, _ =
			fmt.Fprintf(
				&kids,
				"%d 0 R",
				pageObject,
			)
	}

	objects[1] =
		fmt.Sprintf(
			"<< /Type /Pages /Kids [%s] /Count %d >>",
			kids.String(),
			pageCount,
		)

	contentStart :=
		3 +
			pageCount

	for pageIndex := 0; pageIndex <
		pageCount; pageIndex++ {
		contentObject :=
			contentStart +
				pageIndex

		objects[2+pageIndex] =
			fmt.Sprintf(
				"<< /Type /Page "+
					"/Parent 2 0 R "+
					"/MediaBox [0 0 595 842] "+
					"/Resources << >> "+
					"/Contents %d 0 R >>",
				contentObject,
			)

		content :=
			fmt.Sprintf(
				"q\n"+
					"0.95 g\n"+
					"50 50 495 742 re\n"+
					"f\n"+
					"Q\n"+
					"%% page %d\n",
				pageIndex+1,
			)

		objects[2+
			pageCount+
			pageIndex] =
			pdfTestStream(
				content,
			)
	}

	return objects
}
