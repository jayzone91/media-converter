package web

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jayzone91/media-converter/internal/converter"
)

const (
	maxPDFRedactions = 1000

	minPDFRedactionSize = 0.005
)

type pdfRedactRequest struct {
	UploadID string `json:"upload_id"`

	Redactions []pdfRedactionRequest `json:"redactions"`
}

type pdfRedactionRequest struct {
	Page int `json:"page"`

	X float64 `json:"x"`
	Y float64 `json:"y"`

	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

func decodePDFRedactRequest(
	value string,
) (pdfRedactRequest, error) {
	if strings.TrimSpace(
		value,
	) == "" {
		return pdfRedactRequest{},
			fmt.Errorf(
				"request body is empty",
			)
	}

	decoder :=
		json.NewDecoder(
			strings.NewReader(
				value,
			),
		)

	decoder.DisallowUnknownFields()

	var request pdfRedactRequest

	if err := decoder.Decode(
		&request,
	); err != nil {
		return pdfRedactRequest{},
			fmt.Errorf(
				"decode PDF redaction request: %w",
				err,
			)
	}

	return request, nil
}

func validatePDFRedactRequest(
	request pdfRedactRequest,
	pageCount int,
) (
	map[int][]converter.PDFRedaction,
	error,
) {
	if request.UploadID == "" {
		return nil,
			fmt.Errorf(
				"upload ID is required",
			)
	}

	if len(request.Redactions) == 0 {
		return nil,
			fmt.Errorf(
				"at least one redaction is required",
			)
	}

	if len(request.Redactions) >
		maxPDFRedactions {
		return nil,
			fmt.Errorf(
				"too many redactions",
			)
	}

	result :=
		make(
			map[int][]converter.PDFRedaction,
		)

	for _, redaction := range request.Redactions {
		if err :=
			validatePDFRedactionRequest(
				redaction,
				pageCount,
			); err != nil {
			return nil, err
		}

		result[redaction.Page] =
			append(
				result[redaction.Page],
				converter.PDFRedaction{
					X: redaction.X,
					Y: redaction.Y,

					Width: redaction.Width,

					Height: redaction.Height,
				},
			)
	}

	return result, nil
}

func validatePDFRedactionRequest(
	redaction pdfRedactionRequest,
	pageCount int,
) error {
	if redaction.Page < 1 ||
		redaction.Page > pageCount {
		return fmt.Errorf(
			"invalid redaction page",
		)
	}

	if redaction.X < 0 ||
		redaction.X > 1 ||
		redaction.Y < 0 ||
		redaction.Y > 1 {
		return fmt.Errorf(
			"invalid redaction position",
		)
	}

	if redaction.Width <
		minPDFRedactionSize ||
		redaction.Width > 1 ||
		redaction.Height <
			minPDFRedactionSize ||
		redaction.Height > 1 {
		return fmt.Errorf(
			"invalid redaction dimensions",
		)
	}

	if redaction.X+
		redaction.Width >
		1 ||
		redaction.Y+
			redaction.Height >
			1 {
		return fmt.Errorf(
			"redaction exceeds page",
		)
	}

	return nil
}
