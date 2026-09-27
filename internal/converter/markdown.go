package converter

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var markdownRenderer = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
	),
	goldmark.WithParserOptions(
		parser.WithAutoHeadingID(),
	),
)

var markdownHTMLTemplate = template.Must(
	template.New("markdown").Parse(`<!doctype html>
<html lang="de">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}}</title>
<style>
	html {
		font-family: Arial, Helvetica, sans-serif;
		color: #111827;
		background: #ffffff;
	}

	body {
		max-width: 960px;
		margin: 0 auto;
		padding: 48px 32px;
		line-height: 1.6;
		overflow-wrap: anywhere;
	}

	h1,
	h2,
	h3,
	h4,
	h5,
	h6 {
		line-height: 1.25;
		page-break-after: avoid;
	}

	h1 {
		margin-top: 0;
		font-size: 2em;
	}

	h2 {
		margin-top: 1.6em;
		font-size: 1.5em;
	}

	h3 {
		margin-top: 1.4em;
		font-size: 1.25em;
	}

	p {
		margin: 0 0 1em;
	}

	ul,
	ol {
		margin: 0 0 1em;
		padding-left: 2em;
	}

	blockquote {
		margin: 1em 0;
		padding: 0.25em 0 0.25em 1em;
		border-left: 3px solid #d1d5db;
		color: #4b5563;
		page-break-inside: avoid;
	}

	code {
		padding: 0.15em 0.3em;
		border-radius: 4px;
		background: #f3f4f6;
		font-family: "Cascadia Mono", Consolas, monospace;
	}

	pre {
		padding: 1em;
		border-radius: 6px;
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
	}

	th,
	td {
		padding: 0.5em 0.7em;
		border: 1px solid #d1d5db;
		text-align: left;
		vertical-align: top;
	}

	th {
		background: #f3f4f6;
	}

	thead {
		display: table-header-group;
	}

	tr {
		page-break-inside: avoid;
	}

	a {
		color: #2563eb;
	}

	hr {
		margin: 2em 0;
		border: 0;
		border-top: 1px solid #d1d5db;
	}

	@media print {
		body {
			max-width: none;
			margin: 0;
			padding: 0;
			-webkit-print-color-adjust: exact;
			print-color-adjust: exact;
		}
	}
</style>
</head>
<body>
{{.Body}}
</body>
</html>`),
)

type markdownHTMLDocument struct {
	Title string
	Body  template.HTML
}

func MarkdownToHTML(inputPath string, outputPath string) error {
	content, err := renderMarkdownDocument(inputPath)
	if err != nil {
		return err
	}

	if err := os.WriteFile(
		outputPath,
		content,
		0o600,
	); err != nil {
		return fmt.Errorf(
			"write markdown HTML: %w",
			err,
		)
	}

	return nil
}

func MarkdownToPDF(
	ctx context.Context,
	webPDF *WebPDF,
	inputPath string,
	outputPath string,
) error {
	htmlPath := filepath.Join(
		filepath.Dir(outputPath),
		"markdown-render.html",
	)

	if err := MarkdownToHTML(
		inputPath,
		htmlPath,
	); err != nil {
		return err
	}

	defer os.Remove(htmlPath)

	fileURL, err := localFileURL(htmlPath)
	if err != nil {
		return err
	}

	if err := webPDF.Render(
		ctx,
		fileURL,
		outputPath,
		WebPDFOptions{
			PaperSize:       "a4",
			RenderMode:      "desktop",
			Landscape:       false,
			PrintBackground: true,
		},
	); err != nil {
		return fmt.Errorf(
			"render markdown PDF: %w",
			err,
		)
	}

	return nil
}

func renderMarkdownDocument(
	inputPath string,
) ([]byte, error) {
	source, err := os.ReadFile(inputPath)
	if err != nil {
		return nil,
			fmt.Errorf(
				"read markdown: %w",
				err,
			)
	}

	parsed := markdownRenderer.
		Parser().
		Parse(
			text.NewReader(source),
		)

	if err := validateMarkdownAST(
		parsed,
	); err != nil {
		return nil, err
	}

	var rendered bytes.Buffer

	if err := markdownRenderer.
		Renderer().
		Render(
			&rendered,
			source,
			parsed,
		); err != nil {
		return nil,
			fmt.Errorf(
				"render markdown: %w",
				err,
			)
	}

	document := markdownHTMLDocument{
		Title: markdownTitle(
			source,
			parsed,
		),
		Body: template.HTML(
			rendered.String(),
		),
	}

	var output bytes.Buffer

	if err := markdownHTMLTemplate.Execute(
		&output,
		document,
	); err != nil {
		return nil,
			fmt.Errorf(
				"render markdown HTML document: %w",
				err,
			)
	}

	return output.Bytes(), nil
}

func validateMarkdownAST(document ast.Node) error {
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
				return ast.WalkContinue, nil
			}

			if node.Kind() == ast.KindImage {
				return ast.WalkStop,
					fmt.Errorf(
						"markdown images are not allowed",
					)
			}

			return ast.WalkContinue, nil
		},
	)
}

func markdownTitle(
	source []byte,
	document ast.Node,
) string {
	for node := document.FirstChild(); node != nil; node = node.NextSibling() {
		heading, ok := node.(*ast.Heading)
		if !ok || heading.Level != 1 {
			continue
		}

		title := strings.TrimSpace(
			string(
				heading.Text(source),
			),
		)

		if title != "" {
			return title
		}
	}

	return "Markdown-Dokument"
}

func localFileURL(
	path string,
) (string, error) {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return "",
			fmt.Errorf(
				"resolve HTML path: %w",
				err,
			)
	}

	filePath := filepath.ToSlash(
		absolutePath,
	)

	if !strings.HasPrefix(
		filePath,
		"/",
	) {
		filePath = "/" + filePath
	}

	return (&url.URL{
		Scheme: "file",
		Path:   filePath,
	}).String(), nil
}
