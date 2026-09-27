package converter

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"

	"github.com/jayzone91/media-converter/internal/media"
)

func TextToHTML(
	inputPath string,
	outputPath string,
) error {
	text, err := media.ReadTextFile(
		inputPath,
	)
	if err != nil {
		return fmt.Errorf(
			"read text file: %w",
			err,
		)
	}

	title := strings.TrimSuffix(
		filepath.Base(inputPath),
		filepath.Ext(inputPath),
	)

	document := `<!doctype html>
<html lang="de">
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>` +
		html.EscapeString(title) +
		`</title>
	<style>
		html {
			color-scheme: light dark;
		}

		body {
			box-sizing: border-box;
			max-width: 80rem;
			margin: 0 auto;
			padding: 2rem;
			font-family: system-ui, sans-serif;
			line-height: 1.5;
		}

		pre {
			margin: 0;
			white-space: pre-wrap;
			overflow-wrap: anywhere;
			font: inherit;
		}
	</style>
</head>
<body>
	<pre>` +
		html.EscapeString(text) +
		`</pre>
</body>
</html>
`

	if err := os.WriteFile(
		outputPath,
		[]byte(document),
		0o600,
	); err != nil {
		return fmt.Errorf(
			"write HTML file: %w",
			err,
		)
	}

	return nil
}
