package web

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxPDFCreateTitleLength = 200
	maxPDFCreateBodyLength  = 100000

	minPDFCreateFontSize = 10
	maxPDFCreateFontSize = 24
)

type pdfCreateDocument struct {
	Title       string
	Body        string
	Date        string
	FontSize    int
	IncludeDate bool
}

var pdfCreateTemplate = template.Must(
	template.New(
		"pdf-create",
	).Parse(
		`<!doctype html>
<html lang="de">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<style>
	html {
		font-family: Arial, Helvetica, sans-serif;
		color: #111827;
		background: #ffffff;
	}

	body {
		margin: 0;
		padding: 0;
		font-size: {{.FontSize}}pt;
		line-height: 1.55;
		overflow-wrap: anywhere;
	}

	h1 {
		margin: 0 0 1.25rem;
		font-size: 1.8em;
		line-height: 1.2;
	}

	.document-date {
		margin: -0.75rem 0 1.5rem;
		color: #6b7280;
		font-size: 0.85em;
	}

	.document-body {
		white-space: pre-wrap;
	}

	@media print {
		body {
			-webkit-print-color-adjust: exact;
			print-color-adjust: exact;
		}
	}
</style>
</head>
<body>
{{if .Title}}<h1>{{.Title}}</h1>{{end}}
{{if .IncludeDate}}<div class="document-date">{{.Date}}</div>{{end}}
<div class="document-body">{{.Body}}</div>
</body>
</html>`,
	),
)

func createPDFDocumentHTML(
	document pdfCreateDocument,
) (string, error) {
	var buffer bytes.Buffer

	if err :=
		pdfCreateTemplate.Execute(
			&buffer,
			document,
		); err != nil {
		return "",
			fmt.Errorf(
				"render PDF document HTML: %w",
				err,
			)
	}

	return buffer.String(),
		nil
}

func validatePDFCreateDocument(
	document pdfCreateDocument,
) error {
	title :=
		strings.TrimSpace(
			document.Title,
		)

	body :=
		strings.TrimSpace(
			document.Body,
		)

	if title == "" &&
		body == "" {
		return fmt.Errorf(
			"document is empty",
		)
	}

	if utf8.RuneCountInString(
		document.Title,
	) > maxPDFCreateTitleLength {
		return fmt.Errorf(
			"title too long",
		)
	}

	if utf8.RuneCountInString(
		document.Body,
	) > maxPDFCreateBodyLength {
		return fmt.Errorf(
			"body too long",
		)
	}

	if document.FontSize <
		minPDFCreateFontSize ||
		document.FontSize >
			maxPDFCreateFontSize {
		return fmt.Errorf(
			"invalid font size",
		)
	}

	return nil
}

func newPDFCreateDocument(
	title string,
	body string,
	fontSize int,
	includeDate bool,
) pdfCreateDocument {
	return pdfCreateDocument{
		Title: strings.TrimSpace(
			title,
		),

		Body: strings.TrimSpace(
			body,
		),

		Date: time.Now().
			Format(
				"02.01.2006",
			),

		FontSize: fontSize,

		IncludeDate: includeDate,
	}
}
