package web

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jayzone91/media-converter/internal/converter"
)

func TestPDFCreateMarkdownLargeDocument(
	t *testing.T,
) {
	t.Parallel()

	var body strings.Builder

	body.WriteString(
		"# Großes Markdown-Dokument\n\n",
	)

	for index := 1; index <= 150; index++ {
		_, _ =
			fmt.Fprintf(
				&body,
				"## Abschnitt %d\n\n"+
					"Dies ist ein längerer Absatz für den Integrationstest. "+
					"Der Inhalt soll über mehrere PDF-Seiten verteilt werden "+
					"und dabei vollständig renderbar bleiben.\n\n",
				index,
			)
	}

	renderPDFCreateMarkdownFixture(
		t,
		body.String(),
		func(
			t *testing.T,
			qpdf *converter.QPDF,
			ctx context.Context,
			output string,
		) {
			requirePDFCreatePageCountGreaterThan(
				t,
				qpdf,
				ctx,
				output,
				5,
			)
		},
	)
}

func TestPDFCreateMarkdownTableAcrossPages(
	t *testing.T,
) {
	t.Parallel()

	var body strings.Builder

	body.WriteString(
		"# Große Tabelle\n\n",
	)

	body.WriteString(
		"| Nummer | Name | Beschreibung |\n",
	)

	body.WriteString(
		"| ---: | --- | --- |\n",
	)

	for index := 1; index <= 120; index++ {
		_, _ =
			fmt.Fprintf(
				&body,
				"| %d | Eintrag %d | "+
					"Beschreibung für Tabellenzeile %d mit zusätzlichem Text. |\n",
				index,
				index,
				index,
			)
	}

	renderPDFCreateMarkdownFixture(
		t,
		body.String(),
		func(
			t *testing.T,
			qpdf *converter.QPDF,
			ctx context.Context,
			output string,
		) {
			requirePDFCreatePageCountGreaterThan(
				t,
				qpdf,
				ctx,
				output,
				1,
			)
		},
	)
}

func TestPDFCreateMarkdownLongCodeBlock(
	t *testing.T,
) {
	t.Parallel()

	var body strings.Builder

	body.WriteString(
		"# Langer Codeblock\n\n```go\n",
	)

	for index := 1; index <= 250; index++ {
		_, _ =
			fmt.Fprintf(
				&body,
				"fmt.Println(\"Zeile %03d: "+
					"abcdefghijklmnopqrstuvwxyz0123456789\")\n",
				index,
			)
	}

	body.WriteString(
		"```\n",
	)

	renderPDFCreateMarkdownFixture(
		t,
		body.String(),
		func(
			t *testing.T,
			qpdf *converter.QPDF,
			ctx context.Context,
			output string,
		) {
			requirePDFCreatePageCountGreaterThan(
				t,
				qpdf,
				ctx,
				output,
				1,
			)
		},
	)
}

func TestPDFCreateMarkdownVeryLongURLAndWord(
	t *testing.T,
) {
	t.Parallel()

	longSegment :=
		strings.Repeat(
			"abcdefghijklmnopqrstuvwxyz0123456789",
			30,
		)

	body :=
		"# Lange Inhalte\n\n" +
			"Sehr lange URL:\n\n" +
			"https://example.com/" +
			longSegment +
			"?token=" +
			longSegment +
			"\n\n" +
			"Sehr langes Wort:\n\n" +
			longSegment +
			longSegment +
			"\n"

	renderPDFCreateMarkdownFixture(
		t,
		body,
		nil,
	)
}

func renderPDFCreateMarkdownFixture(
	t *testing.T,
	body string,
	check func(
		*testing.T,
		*converter.QPDF,
		context.Context,
		string,
	),
) {
	t.Helper()

	webPDF, err :=
		converter.NewWebPDF()

	if err != nil {
		t.Skipf(
			"browser unavailable: %v",
			err,
		)
	}

	qpdf, err :=
		converter.NewQPDF()

	if err != nil {
		t.Skipf(
			"qpdf unavailable: %v",
			err,
		)
	}

	document :=
		newPDFCreateDocument(
			"Integrationstest",
			body,
			12,
			false,
			true,
		)

	if err :=
		validatePDFCreateDocument(
			document,
		); err != nil {
		t.Fatalf(
			"validate PDF create document: %v",
			err,
		)
	}

	html, err :=
		createPDFDocumentHTML(
			document,
		)

	if err != nil {
		t.Fatalf(
			"create PDF document HTML: %v",
			err,
		)
	}

	output :=
		filepath.Join(
			t.TempDir(),
			"document.pdf",
		)

	ctx, cancel :=
		context.WithTimeout(
			context.Background(),
			45*time.Second,
		)

	defer cancel()

	if err :=
		webPDF.RenderHTML(
			ctx,
			html,
			output,
			converter.WebPDFOptions{
				PaperSize: "a4",

				RenderMode: "print",

				PrintBackground: true,
			},
		); err != nil {
		t.Fatalf(
			"render PDF document: %v",
			err,
		)
	}

	if err :=
		qpdf.CheckPDF(
			ctx,
			output,
		); err != nil {
		t.Fatalf(
			"created PDF failed structural validation: %v",
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
			"read created PDF page count: %v",
			err,
		)
	}

	if pageCount.Count < 1 {
		t.Fatalf(
			"expected at least one page, got %d",
			pageCount.Count,
		)
	}

	if check != nil {
		check(
			t,
			qpdf,
			ctx,
			output,
		)
	}
}

func requirePDFCreatePageCountGreaterThan(
	t *testing.T,
	qpdf *converter.QPDF,
	ctx context.Context,
	path string,
	minimum int,
) {
	t.Helper()

	result, err :=
		qpdf.PageCount(
			ctx,
			path,
		)

	if err != nil {
		t.Fatalf(
			"read PDF page count: %v",
			err,
		)
	}

	if result.Count <= minimum {
		t.Fatalf(
			"expected more than %d pages, got %d",
			minimum,
			result.Count,
		)
	}
}
