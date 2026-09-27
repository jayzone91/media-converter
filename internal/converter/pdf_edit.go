package converter

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

const (
	maxPDFTextLength  = 500
	maxPDFTextObjects = 500
)

var pdfTextColorPattern = regexp.MustCompile(
	`^#[0-9a-fA-F]{6}$`,
)

type PDFTextEdit struct {
	Page int

	Text string

	X float64
	Y float64

	Size int

	Color string
}

func (c *PDF) AddTextEdits(
	input string,
	output string,
	edits []PDFTextEdit,
) error {
	if len(edits) == 0 {
		return fmt.Errorf(
			"at least one text edit is required",
		)
	}

	if len(edits) > maxPDFTextObjects {
		return fmt.Errorf(
			"too many text edits",
		)
	}

	dimensions, err :=
		api.PageDimsFile(
			input,
		)
	if err != nil {
		return fmt.Errorf(
			"read PDF page dimensions: %w",
			err,
		)
	}

	if len(dimensions) == 0 {
		return fmt.Errorf(
			"PDF contains no pages",
		)
	}

	watermarks := make(
		map[int][]*model.Watermark,
	)

	for _, edit := range edits {
		if err := validatePDFTextEdit(
			edit,
			len(dimensions),
		); err != nil {
			return err
		}

		dimension :=
			dimensions[edit.Page-1]

		x :=
			edit.X *
				dimension.Width

		y :=
			dimension.Height -
				edit.Y*
					dimension.Height -
				float64(edit.Size)

		if y < 0 {
			y = 0
		}

		description :=
			fmt.Sprintf(
				"font:Helvetica, "+
					"points:%d, "+
					"scale:1 abs, "+
					"pos:bl, "+
					"off:%.4f %.4f, "+
					"align:l, "+
					"fillc:%s, "+
					"rot:0, "+
					"op:1, "+
					"rendermode:0",
				edit.Size,
				x,
				y,
				edit.Color,
			)

		watermark, err :=
			api.TextWatermark(
				edit.Text,
				description,
				true,
				false,
				types.POINTS,
			)
		if err != nil {
			return fmt.Errorf(
				"create text stamp for page %d: %w",
				edit.Page,
				err,
			)
		}

		watermarks[edit.Page] =
			append(
				watermarks[edit.Page],
				watermark,
			)
	}

	if err := api.AddWatermarksSliceMapFile(
		input,
		output,
		watermarks,
		nil,
	); err != nil {
		return fmt.Errorf(
			"apply PDF text edits: %w",
			err,
		)
	}

	return nil
}

func validatePDFTextEdit(
	edit PDFTextEdit,
	pageCount int,
) error {
	if edit.Page < 1 ||
		edit.Page > pageCount {
		return fmt.Errorf(
			"invalid page number %d",
			edit.Page,
		)
	}

	if strings.TrimSpace(
		edit.Text,
	) == "" {
		return fmt.Errorf(
			"text on page %d is empty",
			edit.Page,
		)
	}

	if len(
		[]rune(edit.Text),
	) > maxPDFTextLength {
		return fmt.Errorf(
			"text on page %d exceeds maximum length",
			edit.Page,
		)
	}

	if edit.X < 0 ||
		edit.X > 1 {
		return fmt.Errorf(
			"invalid x position on page %d",
			edit.Page,
		)
	}

	if edit.Y < 0 ||
		edit.Y > 1 {
		return fmt.Errorf(
			"invalid y position on page %d",
			edit.Page,
		)
	}

	if edit.Size < 6 ||
		edit.Size > 144 {
		return fmt.Errorf(
			"invalid font size on page %d",
			edit.Page,
		)
	}

	if !pdfTextColorPattern.MatchString(
		edit.Color,
	) {
		return fmt.Errorf(
			"invalid text color on page %d",
			edit.Page,
		)
	}

	return nil
}
