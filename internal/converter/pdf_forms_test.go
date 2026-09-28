package converter

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestHasPDFFormFieldsDetectsAcroForm(
	t *testing.T,
) {
	t.Parallel()

	path :=
		writePDFTestDocument(
			t,
			pdfTestDocumentWithForm(),
		)

	hasForms, err :=
		HasPDFFormFields(
			path,
		)

	if err != nil {
		t.Fatalf(
			"HasPDFFormFields returned error: %v",
			err,
		)
	}

	if !hasForms {
		t.Fatal(
			"expected AcroForm fields to be detected",
		)
	}
}

func TestHasPDFFormFieldsWithoutForm(
	t *testing.T,
) {
	t.Parallel()

	path :=
		writePDFTestDocument(
			t,
			pdfTestDocumentWithoutForm(),
		)

	hasForms, err :=
		HasPDFFormFields(
			path,
		)

	if err != nil {
		t.Fatalf(
			"HasPDFFormFields returned error: %v",
			err,
		)
	}

	if hasForms {
		t.Fatal(
			"expected PDF without form fields to be accepted",
		)
	}
}

func TestHasPDFFormFieldsRejectsInvalidPDF(
	t *testing.T,
) {
	t.Parallel()

	path :=
		filepath.Join(
			t.TempDir(),
			"invalid.pdf",
		)

	if err :=
		os.WriteFile(
			path,
			[]byte(
				"not a pdf",
			),
			0600,
		); err != nil {
		t.Fatalf(
			"write invalid PDF: %v",
			err,
		)
	}

	_, err :=
		HasPDFFormFields(
			path,
		)

	if err == nil {
		t.Fatal(
			"expected invalid PDF to fail",
		)
	}
}

func writePDFTestDocument(
	t *testing.T,
	objects []string,
) string {
	t.Helper()

	var buffer bytes.Buffer

	buffer.WriteString(
		"%PDF-1.7\n",
	)

	offsets :=
		make(
			[]int,
			len(objects)+1,
		)

	for index, object := range objects {
		offsets[index+1] =
			buffer.Len()

		_, _ =
			fmt.Fprintf(
				&buffer,
				"%d 0 obj\n%s\nendobj\n",
				index+1,
				object,
			)
	}

	xrefOffset :=
		buffer.Len()

	_, _ =
		fmt.Fprintf(
			&buffer,
			"xref\n0 %d\n",
			len(objects)+1,
		)

	buffer.WriteString(
		"0000000000 65535 f \n",
	)

	for index := 1; index < len(offsets); index++ {
		_, _ =
			fmt.Fprintf(
				&buffer,
				"%010d 00000 n \n",
				offsets[index],
			)
	}

	_, _ =
		fmt.Fprintf(
			&buffer,
			"trailer\n<< /Size %d /Root 1 0 R >>\n"+
				"startxref\n%d\n%%%%EOF\n",
			len(objects)+1,
			xrefOffset,
		)

	path :=
		filepath.Join(
			t.TempDir(),
			"document.pdf",
		)

	if err :=
		os.WriteFile(
			path,
			buffer.Bytes(),
			0600,
		); err != nil {
		t.Fatalf(
			"write PDF fixture: %v",
			err,
		)
	}

	return path
}

func pdfTestDocumentWithoutForm() []string {
	return []string{
		"<< /Type /Catalog /Pages 2 0 R >>",

		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",

		"<< /Type /Page " +
			"/Parent 2 0 R " +
			"/MediaBox [0 0 200 200] " +
			"/Resources << >> " +
			"/Contents 4 0 R >>",

		"<< /Length 0 >>\nstream\n\nendstream",
	}
}

func pdfTestDocumentWithForm() []string {
	return []string{
		"<< /Type /Catalog " +
			"/Pages 2 0 R " +
			"/AcroForm 5 0 R >>",

		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",

		"<< /Type /Page " +
			"/Parent 2 0 R " +
			"/MediaBox [0 0 200 200] " +
			"/Resources << >> " +
			"/Annots [6 0 R] " +
			"/Contents 4 0 R >>",

		"<< /Length 0 >>\nstream\n\nendstream",

		"<< /Fields [6 0 R] " +
			"/DA (/Helv 12 Tf 0 g) " +
			"/DR << " +
			"/Font << /Helv 7 0 R >> " +
			">> >>",

		"<< /Type /Annot " +
			"/Subtype /Widget " +
			"/FT /Tx " +
			"/T (name) " +
			"/Rect [20 150 180 175] " +
			"/P 3 0 R >>",

		"<< /Type /Font " +
			"/Subtype /Type1 " +
			"/BaseFont /Helvetica >>",
	}
}
