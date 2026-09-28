package converter

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

const (
	maxPDFTextLength   = 500
	maxPDFTextObjects  = 500
	maxPDFImageObjects = 20
)

type PDFTextEdit struct {
	Page int

	Text string

	X float64
	Y float64

	Size int

	Color string
}

type PDFImageEdit struct {
	Page int

	Path string

	X float64
	Y float64

	Width float64
}

func (c *PDF) ApplyEdits(
	input string,
	output string,
	texts []PDFTextEdit,
	images []PDFImageEdit,
	drawings []PDFDrawEdit,
) error {
	if len(texts) == 0 &&
		len(images) == 0 &&
		len(drawings) == 0 {
		return fmt.Errorf(
			"at least one PDF edit is required",
		)
	}

	hasStamps :=
		len(texts) > 0 ||
			len(images) > 0

	if len(drawings) == 0 {
		return applyPDFStampEdits(
			input,
			output,
			texts,
			images,
		)
	}

	drawInput := input

	if hasStamps {
		file, err := os.CreateTemp(
			filepath.Dir(output),
			".pdf-edit-stamps-*.pdf",
		)
		if err != nil {
			return fmt.Errorf(
				"create PDF edit intermediate file: %w",
				err,
			)
		}

		intermediate :=
			file.Name()

		if err := file.Close(); err != nil {
			os.Remove(
				intermediate,
			)

			return fmt.Errorf(
				"close PDF edit intermediate file: %w",
				err,
			)
		}

		defer os.Remove(
			intermediate,
		)

		if err := applyPDFStampEdits(
			input,
			intermediate,
			texts,
			images,
		); err != nil {
			return err
		}

		drawInput =
			intermediate
	}

	return applyPDFDrawEdits(
		drawInput,
		output,
		drawings,
	)
}

func applyPDFStampEdits(
	input string,
	output string,
	texts []PDFTextEdit,
	images []PDFImageEdit,
) error {
	if len(texts) >
		maxPDFTextObjects {
		return fmt.Errorf(
			"too many text edits",
		)
	}

	if len(images) >
		maxPDFImageObjects {
		return fmt.Errorf(
			"too many image edits",
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

	for _, edit := range images {
		watermark, err :=
			imageWatermark(
				edit,
				dimensions,
			)
		if err != nil {
			return err
		}

		watermarks[edit.Page] =
			append(
				watermarks[edit.Page],
				watermark,
			)
	}

	for _, edit := range texts {
		watermark, err :=
			textWatermark(
				edit,
				dimensions,
			)
		if err != nil {
			return err
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
			"apply PDF stamp edits: %w",
			err,
		)
	}

	return nil
}
