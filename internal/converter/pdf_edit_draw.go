package converter

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

const (
	maxPDFDrawStrokes         = 1000
	maxPDFDrawPointsPerStroke = 5000
	maxPDFDrawWidth           = 20
	minPDFDrawWidth           = 1
)

var pdfDrawColorPattern = regexp.MustCompile(
	`^#[0-9a-fA-F]{6}$`,
)

type PDFDrawPoint struct {
	X float64
	Y float64
}

type PDFDrawEdit struct {
	Page int

	Color string

	Width float64

	Points []PDFDrawPoint
}

func applyPDFDrawEdits(
	input string,
	output string,
	drawings []PDFDrawEdit,
) error {
	if len(drawings) >
		maxPDFDrawStrokes {
		return fmt.Errorf(
			"too many drawing strokes",
		)
	}

	dimensions, err :=
		api.PageDimsFile(
			input,
		)
	if err != nil {
		return fmt.Errorf(
			"read PDF page dimensions for drawings: %w",
			err,
		)
	}

	if len(dimensions) == 0 {
		return fmt.Errorf(
			"PDF contains no pages",
		)
	}

	ctx, err :=
		api.ReadContextFile(
			input,
		)
	if err != nil {
		return fmt.Errorf(
			"read PDF for drawing edits: %w",
			err,
		)
	}

	pageStreams := make(
		map[int]*bytes.Buffer,
	)

	for _, drawing := range drawings {
		if err := validatePDFDrawEdit(
			drawing,
			len(dimensions),
		); err != nil {
			return err
		}

		buffer :=
			pageStreams[drawing.Page]

		if buffer == nil {
			buffer =
				&bytes.Buffer{}

			pageStreams[drawing.Page] =
				buffer
		}

		if err := writePDFDrawStroke(
			buffer,
			drawing,
			dimensions[drawing.Page-1],
		); err != nil {
			return err
		}
	}

	for page, buffer := range pageStreams {
		if err := appendPDFPageContent(
			ctx,
			page,
			buffer.Bytes(),
		); err != nil {
			return err
		}
	}

	if err := api.WriteContextFile(
		ctx,
		output,
	); err != nil {
		return fmt.Errorf(
			"write PDF drawing edits: %w",
			err,
		)
	}

	return nil
}

func writePDFDrawStroke(
	buffer *bytes.Buffer,
	drawing PDFDrawEdit,
	dimension types.Dim,
) error {
	red, green, blue, err :=
		parsePDFDrawColor(
			drawing.Color,
		)
	if err != nil {
		return err
	}

	first :=
		drawing.Points[0]

	fmt.Fprintf(
		buffer,
		"q\n"+
			"%.6f %.6f %.6f RG\n"+
			"%.4f w\n"+
			"1 J\n"+
			"1 j\n",
		red,
		green,
		blue,
		drawing.Width,
	)

	firstX :=
		first.X *
			dimension.Width

	firstY :=
		dimension.Height -
			first.Y*
				dimension.Height

	fmt.Fprintf(
		buffer,
		"%.4f %.4f m\n",
		firstX,
		firstY,
	)

	if len(drawing.Points) == 1 {
		fmt.Fprintf(
			buffer,
			"%.4f %.4f l\n",
			firstX+0.01,
			firstY,
		)
	} else {
		for _, point := range drawing.Points[1:] {
			x :=
				point.X *
					dimension.Width

			y :=
				dimension.Height -
					point.Y*
						dimension.Height

			fmt.Fprintf(
				buffer,
				"%.4f %.4f l\n",
				x,
				y,
			)
		}
	}

	buffer.WriteString(
		"S\nQ\n",
	)

	return nil
}

func appendPDFPageContent(
	ctx *model.Context,
	page int,
	content []byte,
) error {
	pageDict, _, _, err :=
		ctx.PageDict(
			page,
			false,
		)
	if err != nil {
		return fmt.Errorf(
			"read page %d for drawing: %w",
			page,
			err,
		)
	}

	if pageDict == nil {
		return fmt.Errorf(
			"page %d not found",
			page,
		)
	}

	stream, err :=
		ctx.XRefTable.NewStreamDictForBuf(
			content,
		)
	if err != nil {
		return fmt.Errorf(
			"create drawing stream for page %d: %w",
			page,
			err,
		)
	}

	if err := stream.Encode(); err != nil {
		return fmt.Errorf(
			"encode drawing stream for page %d: %w",
			page,
			err,
		)
	}

	streamReference, err :=
		ctx.XRefTable.IndRefForNewObject(
			*stream,
		)
	if err != nil {
		return fmt.Errorf(
			"store drawing stream for page %d: %w",
			page,
			err,
		)
	}

	existing, found :=
		pageDict.Find(
			"Contents",
		)

	if !found ||
		existing == nil {
		pageDict.Insert(
			"Contents",
			*streamReference,
		)

		return nil
	}

	resolved, err :=
		ctx.XRefTable.Dereference(
			existing,
		)
	if err != nil {
		return fmt.Errorf(
			"resolve page %d contents: %w",
			page,
			err,
		)
	}

	switch value :=
		resolved.(type) {
	case types.Array:
		contents :=
			append(
				types.Array{},
				value...,
			)

		contents =
			append(
				contents,
				*streamReference,
			)

		pageDict.Insert(
			"Contents",
			contents,
		)

	case types.StreamDict:
		pageDict.Insert(
			"Contents",
			types.Array{
				existing,
				*streamReference,
			},
		)

	default:
		return fmt.Errorf(
			"unsupported page %d contents type %T",
			page,
			resolved,
		)
	}

	return nil
}

func validatePDFDrawEdit(
	drawing PDFDrawEdit,
	pageCount int,
) error {
	if drawing.Page < 1 ||
		drawing.Page > pageCount {
		return fmt.Errorf(
			"invalid drawing page %d",
			drawing.Page,
		)
	}

	if !pdfDrawColorPattern.MatchString(
		drawing.Color,
	) {
		return fmt.Errorf(
			"invalid drawing color on page %d",
			drawing.Page,
		)
	}

	if drawing.Width <
		minPDFDrawWidth ||
		drawing.Width >
			maxPDFDrawWidth {
		return fmt.Errorf(
			"invalid drawing width on page %d",
			drawing.Page,
		)
	}

	if len(drawing.Points) == 0 {
		return fmt.Errorf(
			"drawing on page %d contains no points",
			drawing.Page,
		)
	}

	if len(drawing.Points) >
		maxPDFDrawPointsPerStroke {
		return fmt.Errorf(
			"drawing on page %d contains too many points",
			drawing.Page,
		)
	}

	for _, point := range drawing.Points {
		if point.X < 0 ||
			point.X > 1 ||
			point.Y < 0 ||
			point.Y > 1 {
			return fmt.Errorf(
				"invalid drawing point on page %d",
				drawing.Page,
			)
		}
	}

	return nil
}

func parsePDFDrawColor(
	value string,
) (float64, float64, float64, error) {
	if !pdfDrawColorPattern.MatchString(
		value,
	) {
		return 0,
			0,
			0,
			fmt.Errorf(
				"invalid drawing color",
			)
	}

	raw :=
		value[1:]

	red, err :=
		strconv.ParseUint(
			raw[0:2],
			16,
			8,
		)
	if err != nil {
		return 0, 0, 0, err
	}

	green, err :=
		strconv.ParseUint(
			raw[2:4],
			16,
			8,
		)
	if err != nil {
		return 0, 0, 0, err
	}

	blue, err :=
		strconv.ParseUint(
			raw[4:6],
			16,
			8,
		)
	if err != nil {
		return 0, 0, 0, err
	}

	return float64(red) / 255,
		float64(green) / 255,
		float64(blue) / 255,
		nil
}
