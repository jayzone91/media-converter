package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/jayzone91/media-converter/internal/converter"
)

var (
	errImageCountMismatch = errors.New(
		"image metadata count does not match uploaded images",
	)

	errTooManyPDFEditImages = errors.New(
		"too many images",
	)
)

type pdfEditRequest struct {
	UploadID string `json:"upload_id"`

	Texts []pdfTextEditRequest `json:"texts"`

	Images []pdfImageEditRequest `json:"images"`
}

type pdfTextEditRequest struct {
	Page int `json:"page"`

	Text string `json:"text"`

	X float64 `json:"x"`
	Y float64 `json:"y"`

	Size int `json:"size"`

	Color string `json:"color"`
}

type pdfImageEditRequest struct {
	Page int `json:"page"`

	X float64 `json:"x"`
	Y float64 `json:"y"`

	Width float64 `json:"width"`
}

func decodePDFEditRequest(
	value string,
) (pdfEditRequest, error) {
	if strings.TrimSpace(
		value,
	) == "" {
		return pdfEditRequest{},
			fmt.Errorf(
				"metadata is missing",
			)
	}

	decoder :=
		json.NewDecoder(
			strings.NewReader(
				value,
			),
		)

	decoder.DisallowUnknownFields()

	var request pdfEditRequest

	if err := decoder.Decode(
		&request,
	); err != nil {
		return pdfEditRequest{},
			fmt.Errorf(
				"decode PDF edit metadata: %w",
				err,
			)
	}

	return request, nil
}

func validatePDFTextEditRequest(
	requests []pdfTextEditRequest,
	pageCount int,
) ([]converter.PDFTextEdit, error) {
	edits := make(
		[]converter.PDFTextEdit,
		0,
		len(requests),
	)

	for _, request := range requests {
		if err :=
			validatePDFTextEditValues(
				request,
				pageCount,
			); err != nil {
			return nil, err
		}

		edits = append(
			edits,
			converter.PDFTextEdit{
				Page: request.Page,

				Text: request.Text,

				X: request.X,
				Y: request.Y,

				Size: request.Size,

				Color: request.Color,
			},
		)
	}

	return edits, nil
}

func validatePDFTextEditValues(
	request pdfTextEditRequest,
	pageCount int,
) error {
	if request.Page < 1 ||
		request.Page > pageCount {
		return fmt.Errorf(
			"invalid text page",
		)
	}

	if request.X < 0 ||
		request.X > 1 ||
		request.Y < 0 ||
		request.Y > 1 {
		return fmt.Errorf(
			"invalid text position",
		)
	}

	if request.Size < 6 ||
		request.Size > 144 {
		return fmt.Errorf(
			"invalid font size",
		)
	}

	if strings.TrimSpace(
		request.Text,
	) == "" {
		return fmt.Errorf(
			"empty text",
		)
	}

	return nil
}

func validatePDFImageEditRequest(
	request pdfImageEditRequest,
	pageCount int,
) error {
	if request.Page < 1 ||
		request.Page > pageCount {
		return fmt.Errorf(
			"invalid image page",
		)
	}

	if request.X < 0 ||
		request.X > 1 ||
		request.Y < 0 ||
		request.Y > 1 {
		return fmt.Errorf(
			"invalid image position",
		)
	}

	if request.Width < 0.05 ||
		request.Width > 1 {
		return fmt.Errorf(
			"invalid image width",
		)
	}

	return nil
}

func handlePDFEditValidationError(
	s *Server,
	w http.ResponseWriter,
	r *http.Request,
	uploadID string,
	err error,
) {
	s.logWarn(
		r,
		"PDF edit rejected",
		"reason",
		err.Error(),
		"upload_id",
		uploadID,
	)

	http.Error(
		w,
		"Die PDF-Änderungen sind ungültig.",
		http.StatusBadRequest,
	)
}
