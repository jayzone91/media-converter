package converter

import (
	"testing"
)

func TestHasPDFSignaturesWithoutSignature(
	t *testing.T,
) {
	t.Parallel()

	path :=
		writePDFTestDocument(
			t,
			pdfTestDocumentWithoutForm(),
		)

	hasSignatures, err :=
		HasPDFSignatures(
			path,
		)

	if err != nil {
		t.Fatalf(
			"HasPDFSignatures returned error: %v",
			err,
		)
	}

	if hasSignatures {
		t.Fatal(
			"expected unsigned PDF",
		)
	}
}

func TestHasPDFSignaturesDoesNotAcceptSignatureFieldAsUnsigned(
	t *testing.T,
) {
	t.Parallel()

	path :=
		writePDFTestDocument(
			t,
			pdfTestDocumentWithSignatureField(),
		)

	hasSignatures, err :=
		HasPDFSignatures(
			path,
		)

	if err == nil &&
		!hasSignatures {
		t.Fatal(
			"PDF containing a signature field must not be treated as unsigned",
		)
	}
}

func pdfTestDocumentWithSignatureField() []string {
	return []string{
		"<< /Type /Catalog " +
			"/Pages 2 0 R " +
			"/AcroForm 5 0 R >>",

		"<< /Type /Pages " +
			"/Kids [3 0 R] " +
			"/Count 1 >>",

		"<< /Type /Page " +
			"/Parent 2 0 R " +
			"/MediaBox [0 0 200 200] " +
			"/Resources << >> " +
			"/Annots [6 0 R] " +
			"/Contents 4 0 R >>",

		"<< /Length 0 >>\nstream\n\nendstream",

		"<< /Fields [6 0 R] " +
			"/SigFlags 3 >>",

		"<< /Type /Annot " +
			"/Subtype /Widget " +
			"/FT /Sig " +
			"/T (Signature1) " +
			"/Rect [20 20 180 60] " +
			"/P 3 0 R " +
			"/V 7 0 R >>",

		"<< /Type /Sig " +
			"/Filter /Adobe.PPKLite " +
			"/SubFilter /adbe.pkcs7.detached " +
			"/ByteRange [0 0 0 0] " +
			"/Contents <00> " +
			"/M (D:20260928140000+02'00') >>",
	}
}
