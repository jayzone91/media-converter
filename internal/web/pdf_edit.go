package web

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/jayzone91/media-converter/internal/converter"
)

const (
	maxPDFEditRequestSize int64 = 1 << 20

	pdfEditTimeout = 30 * time.Minute
)

type pdfEditRequest struct {
	UploadID string `json:"upload_id"`

	Texts []pdfTextEditRequest `json:"texts"`
}

type pdfTextEditRequest struct {
	Page int `json:"page"`

	Text string `json:"text"`

	X float64 `json:"x"`
	Y float64 `json:"y"`

	Size int `json:"size"`

	Color string `json:"color"`
}

func (s *Server) handlePDFEdit(
	w http.ResponseWriter,
	r *http.Request,
) {
	started := time.Now()

	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxPDFEditRequestSize,
	)
	defer r.Body.Close()

	var request pdfEditRequest

	decoder := json.NewDecoder(
		r.Body,
	)

	decoder.DisallowUnknownFields()

	if err := decoder.Decode(
		&request,
	); err != nil {
		s.logWarn(
			r,
			"PDF edit rejected",
			"reason",
			"invalid request",
		)

		http.Error(
			w,
			"Ungültige Anfrage.",
			http.StatusBadRequest,
		)

		return
	}

	if request.UploadID == "" {
		http.Error(
			w,
			"Upload-ID fehlt.",
			http.StatusBadRequest,
		)

		return
	}

	if len(request.Texts) == 0 {
		http.Error(
			w,
			"Es wurden keine Änderungen vorgenommen.",
			http.StatusBadRequest,
		)

		return
	}

	upload, ok :=
		s.pdfUploads.Get(
			request.UploadID,
		)

	if !ok {
		http.Error(
			w,
			"Die PDF ist nicht mehr verfügbar. Bitte erneut hochladen.",
			http.StatusGone,
		)

		return
	}

	edits, err :=
		validatePDFTextEditRequest(
			request.Texts,
			upload.PageCount,
		)
	if err != nil {
		s.logWarn(
			r,
			"PDF edit rejected",
			"reason",
			err.Error(),
			"upload_id",
			upload.ID,
		)

		http.Error(
			w,
			"Die PDF-Änderungen sind ungültig.",
			http.StatusBadRequest,
		)

		return
	}

	if err := s.acquireConversionSlot(
		r.Context(),
	); err != nil {
		http.Error(
			w,
			"Der Server ist momentan ausgelastet. Bitte später erneut versuchen.",
			http.StatusServiceUnavailable,
		)

		return
	}

	defer s.releaseConversionSlot()

	tempDir, err := os.MkdirTemp(
		"",
		"media-converter-pdf-edit-*",
	)
	if err != nil {
		s.logError(
			r,
			"PDF edit temporary directory failed",
			err,
		)

		http.Error(
			w,
			"Temporäres Verzeichnis konnte nicht erstellt werden.",
			http.StatusInternalServerError,
		)

		return
	}

	defer os.RemoveAll(
		tempDir,
	)

	outputPath := filepath.Join(
		tempDir,
		"bearbeitet.pdf",
	)

	if err := s.pdf.AddTextEdits(
		upload.Path,
		outputPath,
		edits,
	); err != nil {
		s.logError(
			r,
			"PDF text edit failed",
			err,
			"upload_id",
			upload.ID,
			"filename",
			upload.Filename,
			"texts",
			len(edits),
		)

		http.Error(
			w,
			"Die PDF konnte nicht bearbeitet werden.",
			http.StatusInternalServerError,
		)

		return
	}

	outputSize, err :=
		downloadFileSize(
			outputPath,
		)
	if err != nil {
		s.logError(
			r,
			"PDF edit output inspection failed",
			err,
			"upload_id",
			upload.ID,
		)

		http.Error(
			w,
			"Die erzeugte PDF konnte nicht gelesen werden.",
			http.StatusInternalServerError,
		)

		return
	}

	if !s.prepareDownloadResponse(
		w,
		r,
		outputPath,
		"bearbeitet.pdf",
	) {
		return
	}

	s.pdfUploads.Delete(
		upload.ID,
	)

	s.logInfo(
		"PDF edited",
		"pages",
		upload.PageCount,
		"texts",
		len(edits),
		"input",
		upload.Size,
		"output",
		outputSize,
		"duration",
		time.Since(started),
	)
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
		edit :=
			converter.PDFTextEdit{
				Page: request.Page,

				Text: request.Text,

				X: request.X,
				Y: request.Y,

				Size: request.Size,

				Color: request.Color,
			}

		if request.Page < 1 ||
			request.Page > pageCount {
			return nil,
				&pdfEditValidationError{
					Message: "invalid page",
				}
		}

		if request.X < 0 ||
			request.X > 1 ||
			request.Y < 0 ||
			request.Y > 1 {
			return nil,
				&pdfEditValidationError{
					Message: "invalid position",
				}
		}

		if request.Size < 6 ||
			request.Size > 144 {
			return nil,
				&pdfEditValidationError{
					Message: "invalid font size",
				}
		}

		if request.Text == "" {
			return nil,
				&pdfEditValidationError{
					Message: "empty text",
				}
		}

		edits = append(
			edits,
			edit,
		)
	}

	return edits, nil
}

type pdfEditValidationError struct {
	Message string
}

func (e *pdfEditValidationError) Error() string {
	return e.Message
}
