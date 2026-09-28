package web

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestValidatePDFOptimizationInputRejectsSignatureProblem(
	t *testing.T,
) {
	t.Parallel()

	path :=
		filepath.Join(
			t.TempDir(),
			"signed.pdf",
		)

	data :=
		[]byte(
			"%PDF-1.7\n" +
				"1 0 obj\n" +
				"<< /Type /Catalog /Pages 2 0 R /AcroForm 5 0 R >>\n" +
				"endobj\n" +

				"2 0 obj\n" +
				"<< /Type /Pages /Kids [3 0 R] /Count 1 >>\n" +
				"endobj\n" +

				"3 0 obj\n" +
				"<< /Type /Page /Parent 2 0 R " +
				"/MediaBox [0 0 200 200] " +
				"/Annots [6 0 R] >>\n" +
				"endobj\n" +

				"5 0 obj\n" +
				"<< /Fields [6 0 R] /SigFlags 3 >>\n" +
				"endobj\n" +

				"6 0 obj\n" +
				"<< /Type /Annot /Subtype /Widget " +
				"/FT /Sig /T (Signature1) " +
				"/Rect [20 20 180 60] " +
				"/V 7 0 R >>\n" +
				"endobj\n" +

				"7 0 obj\n" +
				"<< /Type /Sig " +
				"/Filter /Adobe.PPKLite " +
				"/SubFilter /adbe.pkcs7.detached " +
				"/ByteRange [0 0 0 0] " +
				"/Contents <00> >>\n" +
				"endobj\n" +

				"trailer\n" +
				"<< /Root 1 0 R >>\n" +
				"%%EOF\n",
		)

	if err :=
		os.WriteFile(
			path,
			data,
			0600,
		); err != nil {
		t.Fatalf(
			"write signed PDF fixture: %v",
			err,
		)
	}

	err :=
		validatePDFOptimizationInput(
			path,
		)

	if err == nil {
		t.Fatal(
			"expected signed or malformed signed PDF to be rejected",
		)
	}

	if errors.Is(
		err,
		errPDFOptimizationSignedDocument,
	) {
		return
	}

	/*
		Auch ein Validierungsfehler ist hier korrekt:
		Der zentrale Code arbeitet fail-closed und darf
		eine problematische Signatur nicht optimieren.
	*/
}
