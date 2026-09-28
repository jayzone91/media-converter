package converter

import (
	"context"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strconv"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

const pdfRedactDPI = 180

type PDFRedaction struct {
	X float64
	Y float64

	Width  float64
	Height float64
}

func (c *PDF) RenderRedactedPage(
	ctx context.Context,
	input string,
	output string,
	page int,
	redactions []PDFRedaction,
) error {
	if page < 1 {
		return fmt.Errorf(
			"invalid PDF page number: %d",
			page,
		)
	}

	if len(redactions) == 0 {
		return fmt.Errorf(
			"at least one redaction is required",
		)
	}

	dimensions, err :=
		api.PageDimsFile(
			input,
		)
	if err != nil {
		return fmt.Errorf(
			"read PDF dimensions: %w",
			err,
		)
	}

	if page > len(dimensions) {
		return fmt.Errorf(
			"PDF page %d does not exist",
			page,
		)
	}

	for _, redaction := range redactions {
		if err :=
			validatePDFRedaction(
				redaction,
			); err != nil {
			return err
		}
	}

	tempDir, err :=
		os.MkdirTemp(
			filepath.Dir(output),
			".pdf-redact-*",
		)
	if err != nil {
		return fmt.Errorf(
			"create redaction temp directory: %w",
			err,
		)
	}

	defer os.RemoveAll(
		tempDir,
	)

	imagePath :=
		filepath.Join(
			tempDir,
			"page.png",
		)

	if err := c.renderRedactionPage(
		ctx,
		input,
		imagePath,
		page,
	); err != nil {
		return err
	}

	if err := burnPDFRedactions(
		imagePath,
		redactions,
	); err != nil {
		return err
	}

	dimension :=
		dimensions[page-1]

	importConfig :=
		fmt.Sprintf(
			"dimensions:%.4f %.4f, position:full",
			dimension.Width,
			dimension.Height,
		)

	imp, err :=
		api.Import(
			importConfig,
			types.POINTS,
		)
	if err != nil {
		return fmt.Errorf(
			"create redacted page import config: %w",
			err,
		)
	}

	if err := api.ImportImagesFile(
		[]string{
			imagePath,
		},
		output,
		imp,
		nil,
	); err != nil {
		return fmt.Errorf(
			"create redacted PDF page: %w",
			err,
		)
	}

	return nil
}

func (c *PDF) renderRedactionPage(
	ctx context.Context,
	input string,
	output string,
	page int,
) error {
	prefix :=
		output[:len(output)-len(filepath.Ext(output))]

	pageString :=
		strconv.Itoa(
			page,
		)

	cmd :=
		externalCommandContext(
			ctx,
			c.pdfToPPM,
			"-f",
			pageString,
			"-l",
			pageString,
			"-singlefile",
			"-png",
			"-r",
			strconv.Itoa(
				pdfRedactDPI,
			),
			input,
			prefix,
		)

	result, err :=
		cmd.CombinedOutput()

	if err != nil {
		if ctxErr :=
			ctx.Err(); ctxErr != nil {
			return fmt.Errorf(
				"PDF redaction rendering failed: %w",
				ctxErr,
			)
		}

		return fmt.Errorf(
			"pdftoppm redaction rendering failed: %w: %s",
			err,
			string(result),
		)
	}

	if _, err := os.Stat(
		output,
	); err != nil {
		return fmt.Errorf(
			"redaction page image was not created: %w",
			err,
		)
	}

	return nil
}

func burnPDFRedactions(
	path string,
	redactions []PDFRedaction,
) error {
	file, err :=
		os.Open(
			path,
		)
	if err != nil {
		return fmt.Errorf(
			"open redaction image: %w",
			err,
		)
	}

	source, err :=
		png.Decode(
			file,
		)

	closeErr :=
		file.Close()

	if err != nil {
		return fmt.Errorf(
			"decode redaction image: %w",
			err,
		)
	}

	if closeErr != nil {
		return fmt.Errorf(
			"close redaction image: %w",
			closeErr,
		)
	}

	bounds :=
		source.Bounds()

	target :=
		image.NewRGBA(
			bounds,
		)

	draw.Draw(
		target,
		bounds,
		source,
		bounds.Min,
		draw.Src,
	)

	black :=
		image.NewUniform(
			image.Black,
		)

	for _, redaction := range redactions {
		rectangle :=
			redactionPixelRectangle(
				bounds,
				redaction,
			)

		draw.Draw(
			target,
			rectangle,
			black,
			image.Point{},
			draw.Src,
		)
	}

	output, err :=
		os.OpenFile(
			path,
			os.O_WRONLY|
				os.O_TRUNC,
			0600,
		)
	if err != nil {
		return fmt.Errorf(
			"rewrite redaction image: %w",
			err,
		)
	}

	encodeErr :=
		png.Encode(
			output,
			target,
		)

	closeErr =
		output.Close()

	if encodeErr != nil {
		return fmt.Errorf(
			"encode redaction image: %w",
			encodeErr,
		)
	}

	if closeErr != nil {
		return fmt.Errorf(
			"close redaction output: %w",
			closeErr,
		)
	}

	return nil
}

func redactionPixelRectangle(
	bounds image.Rectangle,
	redaction PDFRedaction,
) image.Rectangle {
	width :=
		bounds.Dx()

	height :=
		bounds.Dy()

	x1 :=
		bounds.Min.X +
			int(
				redaction.X*
					float64(width),
			)

	y1 :=
		bounds.Min.Y +
			int(
				redaction.Y*
					float64(height),
			)

	x2 :=
		bounds.Min.X +
			int(
				(redaction.X+redaction.Width)*
					float64(width),
			)

	y2 :=
		bounds.Min.Y +
			int(
				(redaction.Y+redaction.Height)*
					float64(height),
			)

	if x2 <= x1 {
		x2 =
			x1 + 1
	}

	if y2 <= y1 {
		y2 =
			y1 + 1
	}

	return image.Rect(
		x1,
		y1,
		x2,
		y2,
	).Intersect(
		bounds,
	)
}

func validatePDFRedaction(
	redaction PDFRedaction,
) error {
	if redaction.X < 0 ||
		redaction.X > 1 ||
		redaction.Y < 0 ||
		redaction.Y > 1 {
		return fmt.Errorf(
			"invalid redaction position",
		)
	}

	if redaction.Width <= 0 ||
		redaction.Width > 1 ||
		redaction.Height <= 0 ||
		redaction.Height > 1 {
		return fmt.Errorf(
			"invalid redaction dimensions",
		)
	}

	if redaction.X+
		redaction.Width >
		1 {
		return fmt.Errorf(
			"redaction exceeds page width",
		)
	}

	if redaction.Y+
		redaction.Height >
		1 {
		return fmt.Errorf(
			"redaction exceeds page height",
		)
	}

	return nil
}
