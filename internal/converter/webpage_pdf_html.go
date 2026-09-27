package converter

import (
	"context"
	"encoding/base64"
)

func (c *WebPDF) RenderHTML(
	ctx context.Context,
	html string,
	output string,
	options WebPDFOptions,
) error {
	dataURL :=
		"data:text/html;charset=utf-8;base64," +
			base64.StdEncoding.EncodeToString(
				[]byte(html),
			)

	return c.Render(
		ctx,
		dataURL,
		output,
		options,
	)
}
