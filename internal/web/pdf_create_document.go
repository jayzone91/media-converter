package web

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

const (
	maxPDFCreateTitleLength = 200
	maxPDFCreateBodyLength  = 100000

	minPDFCreateFontSize = 10
	maxPDFCreateFontSize = 24
)

var errPDFCreateMarkdownImage = errors.New(
	"markdown images are not allowed",
)

type pdfCreateDocument struct {
	Title string

	Body string

	BodyHTML template.HTML

	Date string

	FontSize int

	IncludeDate bool

	Markdown bool
}

var pdfCreateMarkdown = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
	),
	goldmark.WithParserOptions(
		parser.WithAutoHeadingID(),
	),
)

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

	h1,
	h2,
	h3,
	h4,
	h5,
	h6 {
		page-break-after: avoid;
		line-height: 1.25;
	}

	h1 {
		margin: 0 0 1.25rem;
		font-size: 1.8em;
	}

	h2 {
		margin: 1.6em 0 0.7em;
		font-size: 1.45em;
	}

	h3 {
		margin: 1.4em 0 0.6em;
		font-size: 1.2em;
	}

	p {
		margin: 0 0 0.9em;
	}

	ul,
	ol {
		margin: 0 0 1em;
		padding-left: 1.8em;
	}

	li {
		margin: 0.25em 0;
	}

	blockquote {
		margin: 1em 0;
		padding: 0.2em 0 0.2em 1em;
		border-left: 3px solid #d1d5db;
		color: #4b5563;
	}

	code {
		padding: 0.12em 0.3em;
		border-radius: 0.25em;
		background: #f3f4f6;
		font-family: "Cascadia Mono", Consolas, monospace;
		font-size: 0.9em;
	}

	pre {
		margin: 1em 0;
		padding: 0.9em;
		border-radius: 0.4em;
		background: #f3f4f6;
		white-space: pre-wrap;
		overflow-wrap: anywhere;
		page-break-inside: avoid;
	}

	pre code {
		padding: 0;
		background: transparent;
	}

	table {
		width: 100%;
		margin: 1em 0;
		border-collapse: collapse;
		font-size: 0.92em;
	}

	th,
	td {
		padding: 0.5em 0.65em;
		border: 1px solid #d1d5db;
		text-align: left;
		vertical-align: top;
	}

	th {
		background: #f3f4f6;
		font-weight: 600;
	}

	a {
		color: #2563eb;
		text-decoration: underline;
	}

	hr {
		margin: 1.5em 0;
		border: 0;
		border-top: 1px solid #d1d5db;
	}

	.document-date {
		margin: -0.75rem 0 1.5rem;
		color: #6b7280;
		font-size: 0.85em;
	}

	.document-body-text {
		white-space: pre-wrap;
	}

	.document-body-markdown > :first-child {
		margin-top: 0;
	}

	.document-body-markdown > :last-child {
		margin-bottom: 0;
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
{{if .Markdown}}
<div class="document-body-markdown">{{.BodyHTML}}</div>
{{else}}
<div class="document-body-text">{{.BodyHTML}}</div>
{{end}}
</body>
</html>`,
	),
)

func createPDFDocumentHTML(
	document pdfCreateDocument,
) (string, error) {
	bodyHTML, err :=
		createPDFDocumentBodyHTML(
			document,
		)

	if err != nil {
		return "",
			err
	}

	document.BodyHTML =
		bodyHTML

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

func createPDFDocumentBodyHTML(
	document pdfCreateDocument,
) (template.HTML, error) {
	if !document.Markdown {
		return template.HTML(
				template.HTMLEscapeString(
					document.Body,
				),
			),
			nil
	}

	source :=
		[]byte(
			document.Body,
		)

	parsed :=
		pdfCreateMarkdown.
			Parser().
			Parse(
				text.NewReader(
					source,
				),
			)

	if err :=
		validatePDFCreateMarkdownAST(
			parsed,
		); err != nil {
		return "",
			err
	}

	var buffer bytes.Buffer

	if err :=
		pdfCreateMarkdown.
			Renderer().
			Render(
				&buffer,
				source,
				parsed,
			); err != nil {
		return "",
			fmt.Errorf(
				"render markdown: %w",
				err,
			)
	}

	return template.HTML(
			buffer.String(),
		),
		nil
}

func validatePDFCreateMarkdownAST(
	document ast.Node,
) error {
	return ast.Walk(
		document,
		func(
			node ast.Node,
			entering bool,
		) (
			ast.WalkStatus,
			error,
		) {
			if !entering {
				return ast.WalkContinue,
					nil
			}

			if node.Kind() ==
				ast.KindImage {
				return ast.WalkStop,
					errPDFCreateMarkdownImage
			}

			return ast.WalkContinue,
				nil
		},
	)
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

	if document.Markdown {
		source :=
			[]byte(
				document.Body,
			)

		parsed :=
			pdfCreateMarkdown.
				Parser().
				Parse(
					text.NewReader(
						source,
					),
				)

		if err :=
			validatePDFCreateMarkdownAST(
				parsed,
			); err != nil {
			return err
		}
	}

	return nil
}

func newPDFCreateDocument(
	title string,
	body string,
	fontSize int,
	includeDate bool,
	markdown bool,
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

		Markdown: markdown,
	}
}
