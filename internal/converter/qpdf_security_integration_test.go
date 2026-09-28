package converter

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestQPDFEncryptDecryptAES256(
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

	tests :=
		[]struct {
			name          string
			userPassword  string
			ownerPassword string
		}{
			{
				name: "ascii",

				userPassword: "MediaConverter-2026!",

				ownerPassword: "Owner-MediaConverter-2026!",
			},
			{
				name: "unicode",

				userPassword: "Grüße-ÄÖÜ-€-测试",

				ownerPassword: "Eigentümer-🔐-测试",
			},
			{
				name: "spaces",

				userPassword: "password with spaces",

				ownerPassword: "owner password with spaces",
			},
			{
				name: "long",

				userPassword: strings.Repeat(
					"a",
					100,
				),

				ownerPassword: strings.Repeat(
					"b",
					100,
				),
			},
		}

	for _, test := range tests {
		test :=
			test

		t.Run(
			test.name,
			func(
				t *testing.T,
			) {
				t.Parallel()

				input :=
					writePDFTestDocument(
						t,
						pdfTestDocumentWithPages(
							3,
						),
					)

				encrypted :=
					filepath.Join(
						t.TempDir(),
						"encrypted.pdf",
					)

				decrypted :=
					filepath.Join(
						t.TempDir(),
						"decrypted.pdf",
					)

				ctx, cancel :=
					context.WithTimeout(
						context.Background(),
						30*time.Second,
					)

				defer cancel()

				if err :=
					qpdf.Encrypt(
						ctx,
						input,
						test.userPassword,
						test.ownerPassword,
						encrypted,
					); err != nil {
					t.Fatalf(
						"encrypt PDF: %v",
						err,
					)
				}

				requireQPDFAES256Encryption(
					t,
					ctx,
					qpdf,
					encrypted,
					test.userPassword,
				)

				if err :=
					qpdf.Decrypt(
						ctx,
						encrypted,
						test.userPassword,
						decrypted,
					); err != nil {
					t.Fatalf(
						"decrypt PDF: %v",
						err,
					)
				}

				if err :=
					qpdf.CheckPDF(
						ctx,
						decrypted,
					); err != nil {
					t.Fatalf(
						"decrypted PDF failed structural validation: %v",
						err,
					)
				}

				pageCount, err :=
					qpdf.PageCount(
						ctx,
						decrypted,
					)

				if err != nil {
					t.Fatalf(
						"read decrypted PDF page count: %v",
						err,
					)
				}

				if pageCount.Count != 3 {
					t.Fatalf(
						"expected 3 pages, got %d",
						pageCount.Count,
					)
				}
			},
		)
	}
}

func TestQPDFDecryptRejectsWrongPassword(
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
				1,
			),
		)

	encrypted :=
		filepath.Join(
			t.TempDir(),
			"encrypted.pdf",
		)

	decrypted :=
		filepath.Join(
			t.TempDir(),
			"decrypted.pdf",
		)

	ctx, cancel :=
		context.WithTimeout(
			context.Background(),
			30*time.Second,
		)

	defer cancel()

	if err :=
		qpdf.Encrypt(
			ctx,
			input,
			"correct-password",
			"correct-owner-password",
			encrypted,
		); err != nil {
		t.Fatalf(
			"encrypt PDF: %v",
			err,
		)
	}

	err =
		qpdf.Decrypt(
			ctx,
			encrypted,
			"wrong-password",
			decrypted,
		)

	if err == nil {
		t.Fatal(
			"expected decryption with wrong password to fail",
		)
	}
}

func TestQPDFDecryptWithOwnerPassword(
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
				2,
			),
		)

	encrypted :=
		filepath.Join(
			t.TempDir(),
			"encrypted.pdf",
		)

	decrypted :=
		filepath.Join(
			t.TempDir(),
			"decrypted.pdf",
		)

	ctx, cancel :=
		context.WithTimeout(
			context.Background(),
			30*time.Second,
		)

	defer cancel()

	if err :=
		qpdf.Encrypt(
			ctx,
			input,
			"user-password",
			"owner-password",
			encrypted,
		); err != nil {
		t.Fatalf(
			"encrypt PDF: %v",
			err,
		)
	}

	if err :=
		qpdf.Decrypt(
			ctx,
			encrypted,
			"owner-password",
			decrypted,
		); err != nil {
		t.Fatalf(
			"decrypt PDF with owner password: %v",
			err,
		)
	}

	if err :=
		qpdf.CheckPDF(
			ctx,
			decrypted,
		); err != nil {
		t.Fatalf(
			"owner-password decrypted PDF failed validation: %v",
			err,
		)
	}
}

func requireQPDFAES256Encryption(
	t *testing.T,
	ctx context.Context,
	qpdf *QPDF,
	input string,
	password string,
) {
	t.Helper()

	passwordFile, err :=
		createQPDFPasswordFile(
			password,
		)

	if err != nil {
		t.Fatalf(
			"create qpdf password file: %v",
			err,
		)
	}

	defer func() {
		_ = os.Remove(
			passwordFile,
		)
	}()

	command :=
		externalCommandContext(
			ctx,
			qpdf.binary,
			"--warning-exit-0",
			"--password-file="+
				passwordFile,
			"--show-encryption",
			input,
		)

	output, err :=
		command.CombinedOutput()

	if err != nil {
		t.Fatalf(
			"inspect PDF encryption: %v: %s",
			err,
			string(
				output,
			),
		)
	}

	info :=
		string(
			output,
		)

	if !strings.Contains(
		info,
		"R = 6",
	) {
		t.Fatalf(
			"expected PDF encryption revision 6:\n%s",
			info,
		)
	}

	if !strings.Contains(
		info,
		"AESv3",
	) {
		t.Fatalf(
			"expected AES-256/AESv3 encryption:\n%s",
			info,
		)
	}
}
